package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	metadata "github.com/CasperGN/daprmcp/pkg/metadata"
	pubsub "github.com/CasperGN/daprmcp/pkg/pubsub"
)

var (
	httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	opts := &mcp.ServerOptions{
		// We're dynamically loading the available tools from the Metadata API
		// to construct the MCP server's Instruction set
		// TODO: Figure out how to interact with conversation & workflow
		Instructions:      metadata.GetDynamicInstructions(ctx),
		CompletionHandler: complete,
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "daprmcp"}, opts)

	// Tool registrations
	pubsub.RegisterTools(server)

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, handler))
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}
}

func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	// A simple placeholder completion handler
	// TODO: figure out if we need this..
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Total:  1,
			Values: []string{req.Params.Argument.Value + "x"},
		},
	}, nil
}
