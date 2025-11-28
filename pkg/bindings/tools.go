package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InvokeBindingArgs struct {
	BindingName string            `json:"bindingName" jsonschema:"The name of the Dapr output binding component (e.g., 'storage-binding')."`
	Operation   string            `json:"operation" jsonschema:"The operation to perform on the binding (e.g., 'create', 'get', 'delete'). Must be supported by the component."`
	Data        string            `json:"data" jsonschema:"The message or data payload to send to the external system, typically a JSON string."`
	Metadata    map[string]string `json:"metadata" jsonschema:"Optional key-value pairs required by the specific binding component for the operation (e.g., 'key' for a storage binding)."`
}

var daprClient dapr.Client

func invokeOutputBindingTool(ctx context.Context, req *mcp.CallToolRequest, args InvokeBindingArgs) (*mcp.CallToolResult, any, error) {
	data := []byte(args.Data)

	if args.Data == "" {
		data = nil
	}

	bindingReq := &dapr.InvokeBindingRequest{
		Name:      args.BindingName,
		Operation: args.Operation,
		Data:      data,
		Metadata:  args.Metadata,
	}

	resp, err := daprClient.InvokeBinding(ctx, bindingReq)
	if err != nil {
		log.Printf("Dapr InvokeOutputBinding failed for binding %s: %v", args.BindingName, err)
		toolErrorMessage := fmt.Sprintf("Failed to invoke binding '%s' with operation '%s'. Dapr Error: %v", args.BindingName, args.Operation, err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	resultData := ""
	if resp != nil && len(resp.Data) > 0 {
		var prettyJSON bytes.Buffer
		if json.Indent(&prettyJSON, resp.Data, "", "  ") == nil {
			resultData = "\n\nResponse Data:\n" + prettyJSON.String()
		} else {
			resultData = "\n\nResponse Data (Raw):\n" + string(resp.Data)
		}
	}

	successMessage := fmt.Sprintf("Successfully invoked output binding '%s' with operation '%s'.%s", args.BindingName, args.Operation, resultData)

	log.Println(successMessage)
	structuredResult := map[string]string{
		"binding_name":  args.BindingName,
		"operation":     args.Operation,
		"response_data": string(resp.Data),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client

	isDestructive := true
	notReadOnly := false
	notIdempotent := false
	isOpenWorld := true

	mcp.AddTool(server, &mcp.Tool{
		Name:  "invoke_output_binding",
		Title: "Interact with External System via Binding",
		Description: "Invokes an operation on a Dapr output binding component to interact with external systems (e.g., queues, databases, webhooks). **This is a SIDE-EFFECT action that can be DESTRUCTIVE.** Use this tool to perform I/O actions.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide non-empty values for `BindingName`, `Operation`, and the `Data` payload.\n" +
			"2. **NEVER INVENT**: You must NOT invent `BindingName` or `Operation` names; they must be provided by the user or discovered.\n" +
			"3. **CLARIFICATION**: If any required input is missing, you MUST ask the user for clarification before generating the tool call.\n" +
			"4. **METADATA**: The `Metadata` field MUST be used to pass headers or component-specific settings (e.g., overriding the target URL for an HTTP binding).\n\n" +
			"**SECURITY WARNING**: This tool allows interaction with external resources, potentially causing irreversible changes (e.g., placing an order, deleting a resource). Ensure user intent is clear and the operation is authorized.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &isDestructive,
			ReadOnlyHint:    notReadOnly,
			IdempotentHint:  notIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, invokeOutputBindingTool)
}
