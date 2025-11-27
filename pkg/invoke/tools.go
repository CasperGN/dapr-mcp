package invoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InvokeServiceArgs struct {
	AppID    string            `json:"appID" jsonschema:"The Dapr application ID of the service to call (e.g., 'order-processor')."`
	Method   string            `json:"method" jsonschema:"The method/endpoint on the target service to call (e.g., 'status')."`
	Data     string            `json:"data" jsonschema:"The body payload for the request, typically a JSON string."`
	HTTPVerb string            `json:"httpVerb" jsonschema:"The HTTP verb to use (e.g., 'GET', 'POST', 'PUT'). Default is 'POST'."`
	Metadata map[string]string `json:"metadata,omitempty" jsonschema:"Optional key-value pairs to send as HTTP headers."`
}

var daprClient dapr.Client

func invokeServiceTool(ctx context.Context, req *mcp.CallToolRequest, args InvokeServiceArgs) (*mcp.CallToolResult, any, error) {
	if args.HTTPVerb == "" {
		args.HTTPVerb = "POST"
	}

	content := &dapr.DataContent{
		ContentType: "application/json",
		Data:        []byte(args.Data),
	}

	resp, err := daprClient.InvokeMethodWithContent(ctx, args.AppID, args.Method, args.HTTPVerb, content)
	if err != nil {
		log.Printf("Dapr InvokeMethod failed for app %s/%s: %v", args.AppID, args.Method, err)
		return nil, nil, fmt.Errorf("failed to invoke service method: %w", err)
	}

	var resultData bytes.Buffer
	if json.Indent(&resultData, resp, "", "  ") != nil {
		resultData.Write(resp)
	}

	successMessage := fmt.Sprintf(
		"Successfully invoked service '%s' method '%s' (%s). The service responded with data/status.",
		args.AppID, args.Method, args.HTTPVerb,
	)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage + "\n\nResponse:\n" + resultData.String()}},
	}, resultData.String(), nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_service",
		Description: "Calls a method (endpoint) on another Dapr-enabled service using its Dapr App ID. Requires App ID, method, HTTP verb, and payload.",
	}, invokeServiceTool)
}
