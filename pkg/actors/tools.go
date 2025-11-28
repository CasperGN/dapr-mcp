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

	structuredResult := map[string]string{
		"actor_type":     args.ActorType,
		"actor_id":       args.ActorID,
		"actor_method":   args.Method,
		"actor_response": resultData,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage + "\n\nResponse:\n" + resultData}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_actor_method",
		Title:       "Execute State-Altering Actor Method",
		Description: "Executes a method on a Dapr Virtual Actor instance. **This is a stateful action that produces a SIDE EFFECT** (e.g., updating an order status, processing a payment, managing user state). Use this tool only when the requested action requires stateful, single-threaded execution. You must provide the Actor Type, the specific Actor ID, the exact Method name, and the necessary JSON payload in the 'Data' field.",
	}, invokeActorMethodTool)
}
