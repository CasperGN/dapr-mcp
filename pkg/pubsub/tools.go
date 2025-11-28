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

var daprClient dapr.Client

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
	structuredResult := map[string]interface{}{
		"status":      "published",
		"pubsub_name": args.PubsubName,
		"topic":       args.Topic,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

type PublishWithMetadataArgs struct {
	PubsubName string            `json:"pubsubName" jsonschema:"The name of the Dapr pubsub component (e.g., 'pubsub')."`
	Topic      string            `json:"topic" jsonschema:"The topic to publish the message to (e.g., 'orders')."`
	Message    string            `json:"message" jsonschema:"The message payload to publish, typically a JSON string."`
	Metadata   map[string]string `json:"metadata" jsonschema:"Optional key-value pairs to send as message headers or routing data (e.g., 'ttlInSeconds': '60')."`
}

func publishEventWithMetadataTool(ctx context.Context, req *mcp.CallToolRequest, args PublishWithMetadataArgs) (*mcp.CallToolResult, any, error) {
	data := []byte(args.Message)

	opts := make([]dapr.PublishEventOption, 0)
	if len(args.Metadata) > 0 {
		opts = append(opts, dapr.PublishEventWithMetadata(args.Metadata))
	}

	if err := daprClient.PublishEvent(ctx, args.PubsubName, args.Topic, data, opts...); err != nil {
		log.Printf("Dapr PublishEventWithMetadata failed: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Failed to publish event to topic '%s' on pubsub '%s' with metadata. Dapr Error: %v", args.Topic, args.PubsubName, err),
			}},
		}, nil, nil
	}

	successMessage := fmt.Sprintf("Successfully published message with %d metadata key(s) to topic '%s' on pubsub component '%s'.", len(args.Metadata), args.Topic, args.PubsubName)

	log.Println(successMessage)
	structuredResult := map[string]interface{}{
		"status":        "published_with_metadata",
		"pubsub_name":   args.PubsubName,
		"topic":         args.Topic,
		"metadata_keys": len(args.Metadata),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "publish_event",
		Title:       "Publish Event (Simple)",
		Description: "Publishes a message to a topic using the Dapr Pub/Sub building block. **This is a SIDE-EFFECT action that triggers decoupled, asynchronous workflows across the system.** Use only to broadcast critical events (e.g., 'new order received', 'status changed'). Arguments: pubsubName, topic, message.",
	}, publishEventTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "publish_event_with_metadata",
		Title:       "Publish Event (With Metadata)",
		Description: "Publishes a message to a topic including optional metadata/headers (e.g., routing headers, 'ttlInSeconds' for Message Time-to-Live). **This is a SIDE-EFFECT action.** Use this tool when you need granular control over message delivery or routing. Arguments: pubsubName, topic, message, metadata.",
	}, publishEventWithMetadataTool)
}
