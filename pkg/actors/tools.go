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
		toolErrorMessage := fmt.Errorf("dapr InvokeActor failed: %w", err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
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

	isDestructive := true
	notReadOnly := false
	notIdempotent := false
	isOpenWorld := true

	mcp.AddTool(server, &mcp.Tool{
		Name:  "invoke_actor_method",
		Title: "Execute Stateful Actor Method",
		Description: "Executes a method on a Dapr Virtual Actor instance, providing durability and concurrency control. **This is a SIDE-EFFECT action that alters state (e.g., creating an order, updating a payment status).** Use this tool exclusively for requests that require stateful, single-threaded execution.\n\n" +
			"**GUIDANCE:**\n" +
			"1. Use `get_components` to find the `ActorType` of the actor.\n" +
			"2. Ensure `ActorID` and `Method` are explicitly provided by the user.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide non-empty values for `ActorType`, `ActorID`, `Method`, and `Data`.\n" +
			"2. **NEVER INVENT**: You must NOT invent the `ActorType`, `ActorID`, or `Method` names; they must be provided by the user or discovered via another tool.\n" +
			"3. **CLARIFICATION**: If any required input is missing, you MUST ask the user for clarification before generating the tool call.\n\n" +
			"**DATA FORMAT**: The `Data` payload MUST be a single string (often JSON) representing the input parameters for the actor method.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &isDestructive,
			ReadOnlyHint:    notReadOnly,
			IdempotentHint:  notIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, invokeActorMethodTool)
}
