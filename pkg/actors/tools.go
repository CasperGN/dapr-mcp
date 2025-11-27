package actors

import (
	"context"
	"fmt"
	"log"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InvokeActorMethodArgs struct {
	ActorType string `json:"actorType" jsonschema:"The registered actor type (e.g., 'payment-processor')."`
	ActorID   string `json:"actorID" jsonschema:"The unique ID of the actor instance (e.g., 'user-1001')."`
	Method    string `json:"method" jsonschema:"The method name on the actor to call (e.g., 'ProcessOrder')."`
	Data      string `json:"data" jsonschema:"The payload to pass to the actor method (e.g., order details)."`
}

var daprClient dapr.Client

func invokeActorMethodTool(ctx context.Context, req *mcp.CallToolRequest, args InvokeActorMethodArgs) (*mcp.CallToolResult, any, error) {
	actorReq := &dapr.InvokeActorRequest{
		ActorType: args.ActorType,
		ActorID:   args.ActorID,
		Method:    args.Method,
		Data:      []byte(args.Data),
	}

	resp, err := daprClient.InvokeActor(ctx, actorReq)
	if err != nil {
		log.Printf("Dapr InvokeActor failed: %v", err)
		return nil, nil, fmt.Errorf("failed to invoke actor method: %w", err)
	}

	resultData := string(resp.Data)

	successMessage := fmt.Sprintf(
		"Successfully invoked method '%s' on actor type '%s' with ID '%s'. Actor responded with data/status.",
		args.Method, args.ActorType, args.ActorID,
	)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage + "\n\nResponse:\n" + resultData}},
	}, resultData, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_actor_method",
		Description: "Calls a specific method on an instance of a Dapr Virtual Actor (e.g., 'user-1001' on type 'user-manager'). Requires Actor Type, ID, Method, and payload.",
	}, invokeActorMethodTool)
}
