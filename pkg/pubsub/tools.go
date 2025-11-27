package pubsub

import (
	"context"
	"fmt"
	"log"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PublishArgs struct {
	PubsubName string `json:"pubsubName" jsonschema:"The name of the Dapr pubsub component (e.g., 'pubsub')."`
	Topic      string `json:"topic" jsonschema:"The topic to publish the message to (e.g., 'orders')."`
	Message    string `json:"message" jsonschema:"The message payload to publish, typically a JSON string."`
}

func publishEventTool(ctx context.Context, req *mcp.CallToolRequest, args PublishArgs) (*mcp.CallToolResult, any, error) {
	client, err := dapr.NewClient()
	if err != nil {
		log.Printf("Failed to create Dapr client: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Error: Could not connect to Dapr sidecar. Ensure Dapr is running. Details: %v", err),
			}},
		}, nil, nil
	}
	defer client.Close()

	data := []byte(args.Message)

	if err := client.PublishEvent(ctx, args.PubsubName, args.Topic, data); err != nil {
		log.Printf("Dapr PublishEvent failed: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Failed to publish event to topic '%s' on pubsub '%s'. Dapr Error: %v", args.Topic, args.PubsubName, err),
			}},
		}, nil, nil
	}

	successMessage := fmt.Sprintf("Successfully published message to topic '%s' on pubsub component '%s'.", args.Topic, args.PubsubName)

	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, nil, nil
}

func RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "publish_event",
		Description: "Publishes a message using Dapr Pub/Sub.",
	}, publishEventTool)
}
