package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	actor "github.com/CasperGN/daprmcp/pkg/actors"
	binding "github.com/CasperGN/daprmcp/pkg/bindings"
	conversation "github.com/CasperGN/daprmcp/pkg/conversation"
	crypto "github.com/CasperGN/daprmcp/pkg/crypto"
	invoke "github.com/CasperGN/daprmcp/pkg/invoke"
	lock "github.com/CasperGN/daprmcp/pkg/lock"
	metadata "github.com/CasperGN/daprmcp/pkg/metadata"
	pubsub "github.com/CasperGN/daprmcp/pkg/pubsub"
	secret "github.com/CasperGN/daprmcp/pkg/secrets"
	state "github.com/CasperGN/daprmcp/pkg/state"
	dapr "github.com/dapr/go-sdk/client"
)

var (
	httpAddr   = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	DaprClient dapr.Client
)

func initializeDaprClient(ctx context.Context) error {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	var err error

	for i := 0; i < maxRetries; i++ {
		DaprClient, err = dapr.NewClient()
		if err == nil {
			log.Println("SUCCESS: Dapr Client established.")
			return nil
		}
		log.Printf("Dapr client initialization failed (%d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	return fmt.Errorf("failed to create Dapr client after %d attempts: %w", maxRetries, err)
}

func main() {
	flag.Parse()

	// Set up OpenTelemetry propagator for trace context and baggage
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)

	// Set up OpenTelemetry exporter based on standard environment variables
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL")
	if protocol == "" {
		protocol = os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	}
	if protocol == "" {
		protocol = "grpc"
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	headersStr := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")

	if endpoint != "" {
		ctx := context.Background()
		var exporter sdktrace.SpanExporter
		var err error
		headers := make(map[string]string)
		if headersStr != "" {
			// Parse headers like "key1=value1,key2=value2"
			for _, pair := range strings.Split(headersStr, ",") {
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}
		for h := range headers {
			log.Printf("OTEL Header: %s: %s", h, headers[h])
		}
		switch protocol {
		case "grpc":
			// Strip http:// or https:// for gRPC
			cleanEndpoint := strings.TrimPrefix(endpoint, "http://")
			cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
			exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cleanEndpoint), otlptracegrpc.WithHeaders(headers))
		case "http/protobuf":
			// Ensure http:// prefix for HTTP
			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				endpoint = "http://" + endpoint
			}
			exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithHeaders(headers))
		case "http/json":
			log.Printf("http/json protocol not supported, falling back to http/protobuf")
			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				endpoint = "http://" + endpoint
			}
			exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithHeaders(headers))
		default:
			log.Printf("Unsupported protocol: %s", protocol)
		}
		if err != nil {
			log.Printf("Failed to create exporter: %v", err)
		} else {
			resource := sdkresource.NewSchemaless(
				attribute.String("service.name", "daprmcp"),
				attribute.String("service.version", "v1.0.0"),
			)
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resource),
			)
			otel.SetTracerProvider(tp)
		}
	}

	ctx := context.Background()

	if err := initializeDaprClient(ctx); err != nil {
		log.Fatalf("Fatal Error: Could not initialize Dapr Client for tools: %v", err)
	}

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
	log.Printf("Instructions sent to client:\n%s", instructions.String())

	server := mcp.NewServer(&mcp.Implementation{Name: "daprmcp", Version: "v1.0.0"}, opts)

	metadata.RegisterTools(server, DaprClient)
	invoke.RegisterTools(server, DaprClient)
	actor.RegisterTools(server, DaprClient)

	componentPresence := make(map[string]bool)
	components, err := metadata.GetLiveComponentList(ctx, DaprClient)
	if err != nil {
		log.Fatalf("Fatal Error: Could not get Components: %v", err)
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
		originalHandler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
			return server
		}, nil)
		wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/dapr/subscribe") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`[]`))
				return
			}
			carrier := propagation.HeaderCarrier(r.Header)
			ctx := prop.Extract(r.Context(), carrier)
			// Inject baggage into response headers
			prop.Inject(ctx, propagation.HeaderCarrier(w.Header()))
			// Set the context with baggage in the request
			r = r.WithContext(ctx)
			originalHandler.ServeHTTP(w, r)
		})
		log.Printf("MCP handler listening at %s", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, wrappedHandler))
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}

func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Total:  1,
			Values: []string{req.Params.Argument.Value + "x"},
		},
	}, nil
}
