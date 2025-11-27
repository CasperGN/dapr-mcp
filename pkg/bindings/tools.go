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
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Failed to invoke binding '%s' with operation '%s'. Dapr Error: %v", args.BindingName, args.Operation, err),
			}},
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, nil, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_output_binding",
		Description: "Invokes an operation (like 'create' or 'get') on a configured Dapr output binding component. Used to interact with external resources like queues or storage. Requires the binding name, operation, data, and optional metadata.",
	}, invokeOutputBindingTool)
}
