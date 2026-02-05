package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	actor "github.com/dapr/dapr-mcp-server/pkg/actors"
	"github.com/dapr/dapr-mcp-server/pkg/auth"
	binding "github.com/dapr/dapr-mcp-server/pkg/bindings"
	conversation "github.com/dapr/dapr-mcp-server/pkg/conversation"
	crypto "github.com/dapr/dapr-mcp-server/pkg/crypto"
	"github.com/dapr/dapr-mcp-server/pkg/health"
	invoke "github.com/dapr/dapr-mcp-server/pkg/invoke"
	lock "github.com/dapr/dapr-mcp-server/pkg/lock"
	metadata "github.com/dapr/dapr-mcp-server/pkg/metadata"
	pubsub "github.com/dapr/dapr-mcp-server/pkg/pubsub"
	secret "github.com/dapr/dapr-mcp-server/pkg/secrets"
	state "github.com/dapr/dapr-mcp-server/pkg/state"
	"github.com/dapr/dapr-mcp-server/pkg/telemetry"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	httpAddr   = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	DaprClient dapr.Client
)

func initializeDaprClient(ctx context.Context, logger *slog.Logger) error {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	var err error

	for i := 0; i < maxRetries; i++ {
		DaprClient, err = dapr.NewClient()
		if err == nil {
			logger.Info("Dapr client established successfully")
			return nil
		}
		logger.Warn("Dapr client initialization failed",
			"attempt", i+1,
			"max_retries", maxRetries,
			"error", err,
		)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	return fmt.Errorf("failed to create Dapr client after %d attempts: %w", maxRetries, err)
}

func main() {
	flag.Parse()

	// Initialize structured logging
	logLevel := os.Getenv("DAPR_MCP_SERVER_LOG_LEVEL")
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Info("Starting dapr-mcp-server",
		"version", Version,
	)

	ctx := context.Background()

	// Initialize OpenTelemetry
	shutdown, err := telemetry.Initialize(ctx)
	if err != nil {
		logger.Warn("Failed to initialize telemetry, continuing without observability",
			"error", err,
		)
	} else {
		defer func() {
			if shutdownErr := shutdown(ctx); shutdownErr != nil {
				logger.Error("Error shutting down telemetry", "error", shutdownErr)
			}
		}()
		logger.Info("OpenTelemetry initialized successfully")
	}

	// Initialize metrics
	metrics, err := telemetry.NewToolMetrics()
	if err != nil {
		logger.Warn("Failed to initialize metrics", "error", err)
	}
	_ = metrics // Will be used for tool instrumentation in future

	// Set up OpenTelemetry propagator for trace context and baggage
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)

	// Initialize Dapr client
	if initErr := initializeDaprClient(ctx, logger); initErr != nil {
		logger.Error("Fatal error: could not initialize Dapr client", "error", initErr)
		os.Exit(1)
	}

	// Build server instructions
	var instructions strings.Builder
	instructions.WriteString("You are an expert AI assistant for Dapr microservices. Your role is to translate user requests into precise, deterministic, and safe Dapr MCP tool calls.\n\n")

	instructions.WriteString("### Global Safety Rules\n")
	instructions.WriteString("- **Clarity Before Acting**: If ANY required argument is missing (store name, key, topic, etc.), you **MUST run the get_components tool to enrich the information before proceeding**. If arguments are still missing first try the tool with sensible defaults, if this fails ask the user for clarification.\n")
	instructions.WriteString("- **Serialization**: Metadata fields MUST be a dictionary/map (e.g., `{}`) and NEVER a quoted string (e.g., `\"{}\"`).\n")
	instructions.WriteString("- **Multi-Step Workflow**: When multiple operations are requested, execute them sequentially — **one tool call at a time**.\n")
	instructions.WriteString("- **Forbidden Actions**: NEVER invent component names, keys, topics, or cryptographic parameters.\n\n")
	instructions.WriteString("### Tool Call Validity\n")
	instructions.WriteString("Consult the tool's Description for specific component rules (e.g., key formatting, security warnings).\n")

	opts := &mcp.ServerOptions{
		Instructions:      instructions.String(),
		CompletionHandler: complete,
		HasTools:          true,
	}
	logger.Debug("Server instructions configured", "instructions", instructions.String())

	server := mcp.NewServer(&mcp.Implementation{Name: "dapr-mcp-server", Version: Version}, opts)

	// Register core tools
	metadata.RegisterTools(server, DaprClient)
	invoke.RegisterTools(server, DaprClient)
	actor.RegisterTools(server, DaprClient)

	// Discover components and register conditional tools
	componentPresence := make(map[string]bool)
	components, err := metadata.GetLiveComponentList(ctx, DaprClient)
	if err != nil {
		logger.Error("Fatal error: could not get components", "error", err)
		os.Exit(1)
	}
	for _, comp := range components {
		if strings.HasPrefix(comp.Type, "state.") {
			componentPresence["state"] = true
		} else if strings.HasPrefix(comp.Type, "pubsub.") {
			componentPresence["pubsub"] = true
		} else if strings.HasPrefix(comp.Type, "bindings.") {
			componentPresence["bindings"] = true
		} else if strings.HasPrefix(comp.Type, "secretstores.") {
			componentPresence["secrets"] = true
		} else if strings.HasPrefix(comp.Type, "lock.") {
			componentPresence["lock"] = true
		} else if strings.HasPrefix(comp.Type, "conversation.") {
			componentPresence["conversation"] = true
		} else if strings.HasPrefix(comp.Type, "crypto.") {
			componentPresence["crypto"] = true
		}
	}

	logger.Info("Discovered Dapr components", "components", componentPresence)

	if componentPresence["pubsub"] {
		pubsub.RegisterTools(server, DaprClient)
	}
	if componentPresence["bindings"] {
		binding.RegisterTools(server, DaprClient)
	}
	if componentPresence["state"] {
		state.RegisterTools(server, DaprClient)
	}
	if componentPresence["secrets"] {
		secret.RegisterTools(server, DaprClient)
	}
	if componentPresence["conversation"] {
		conversation.RegisterTools(server, DaprClient)
	}
	if componentPresence["crypto"] {
		crypto.RegisterTools(server, DaprClient)
	}
	if componentPresence["lock"] {
		lock.RegisterTools(server, DaprClient)
	}

	if *httpAddr != "" {
		// Initialize health checker
		healthChecker := health.NewChecker(DaprClient, logger)

		// Initialize authentication
		authConfig := auth.DefaultConfig()
		if err := authConfig.Validate(); err != nil {
			logger.Error("Invalid authentication configuration", "error", err)
			os.Exit(1)
		}

		var authMiddleware func(http.Handler) http.Handler
		if authConfig.Enabled && authConfig.Mode != auth.ModeDisabled {
			authenticators, err := buildAuthenticators(ctx, authConfig, logger)
			if err != nil {
				logger.Error("Failed to initialize authenticators", "error", err)
				os.Exit(1)
			}
			middleware := auth.NewMiddleware(authConfig, authenticators, logger)
			authMiddleware = middleware.Handler
			logger.Info("Authentication enabled",
				"mode", authConfig.Mode,
				"skip_paths", authConfig.SkipPaths,
			)
		} else {
			authMiddleware = auth.NoopMiddleware
			logger.Info("Authentication disabled")
		}

		// Create HTTP mux for health endpoints
		mux := http.NewServeMux()

		// Register health endpoints (no auth required)
		mux.HandleFunc("/livez", healthChecker.LivenessHandler)
		mux.HandleFunc("/readyz", healthChecker.ReadinessHandler)
		mux.HandleFunc("/startupz", healthChecker.StartupHandler)

		// Create MCP SSE handler
		mcpHandler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
			return server
		}, nil)

		// Wrap with telemetry and auth middleware
		wrappedMCPHandler := authMiddleware(telemetry.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			carrier := propagation.HeaderCarrier(r.Header)
			ctx := prop.Extract(r.Context(), carrier)
			prop.Inject(ctx, propagation.HeaderCarrier(w.Header()))
			r = r.WithContext(ctx)
			mcpHandler.ServeHTTP(w, r)
		})))

		// Handle Dapr subscription endpoint
		mux.HandleFunc("/dapr/subscribe", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		})

		// Route all other requests to MCP handler
		mux.Handle("/", wrappedMCPHandler)

		logger.Info("MCP HTTP server starting",
			"address", *httpAddr,
			"auth_enabled", authConfig.Enabled,
		)
		srv := &http.Server{
			Addr:              *httpAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}
}

// buildAuthenticators creates the appropriate authenticators based on configuration.
func buildAuthenticators(ctx context.Context, cfg auth.Config, logger *slog.Logger) ([]auth.Authenticator, error) {
	var authenticators []auth.Authenticator

	switch cfg.Mode {
	case auth.ModeOIDC:
		oidc, err := auth.NewOIDCAuthenticator(ctx, cfg.OIDC)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC authenticator: %w", err)
		}
		authenticators = append(authenticators, oidc)

	case auth.ModeSPIFFE:
		spiffe, err := auth.NewSPIFFEAuthenticator(ctx, cfg.SPIFFE)
		if err != nil {
			return nil, fmt.Errorf("failed to create SPIFFE authenticator: %w", err)
		}
		authenticators = append(authenticators, spiffe)

	case auth.ModeHybrid:
		if cfg.OIDC.Enabled {
			oidc, err := auth.NewOIDCAuthenticator(ctx, cfg.OIDC)
			if err != nil {
				return nil, fmt.Errorf("failed to create OIDC authenticator: %w", err)
			}
			authenticators = append(authenticators, oidc)
		}
		if cfg.SPIFFE.Enabled {
			spiffe, err := auth.NewSPIFFEAuthenticator(ctx, cfg.SPIFFE)
			if err != nil {
				return nil, fmt.Errorf("failed to create SPIFFE authenticator: %w", err)
			}
			authenticators = append(authenticators, spiffe)
		}
	}

	logger.Info("Authenticators initialized", "count", len(authenticators))
	return authenticators, nil
}

func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Total:  1,
			Values: []string{req.Params.Argument.Value + "x"},
		},
	}, nil
}
