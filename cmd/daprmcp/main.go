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

	actor "github.com/CasperGN/daprmcp/pkg/actors"
	binding "github.com/CasperGN/daprmcp/pkg/bindings"
	conversation "github.com/CasperGN/daprmcp/pkg/conversation"
	cryptography "github.com/CasperGN/daprmcp/pkg/cryptography"
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

	ctx := context.Background()

	if err := initializeDaprClient(ctx); err != nil {
		log.Fatalf("Fatal Error: Could not initialize Dapr Client for tools: %v", err)
	}

	instructions := metadata.GetDynamicInstructions(ctx, DaprClient)

	componentPresence := make(map[string]bool)
	for _, comp := range metadata.ComponentCache {
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
		} else if strings.HasPrefix(comp.Type, "cryptography.") {
			componentPresence["cryptography"] = true
		}
	}

	opts := &mcp.ServerOptions{
		Instructions:      instructions,
		CompletionHandler: complete,
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "daprmcp", Version: "v1.0.0"}, opts)

	metadata.RegisterTools(server, DaprClient)
	invoke.RegisterTools(server, DaprClient)
	actor.RegisterTools(server, DaprClient)

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
	if componentPresence["cryptography"] {
		cryptography.RegisterTools(server, DaprClient)
	}
	if componentPresence["lock"] {
		lock.RegisterTools(server, DaprClient)
	}

	if *httpAddr != "" {
		handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
			log.Printf("Handling request for URL %s\n", request.URL.Path)
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, handler))
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
