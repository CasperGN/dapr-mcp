package metadata

import (
	"context"
	"fmt"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ComponentListWrapper struct {
	Components []ComponentInfo `json:"components" jsonschema:"A list of Dapr components found in the sidecar."`
}

type ComponentInfo struct {
	Name         string   `json:"name" jsonschema:"The unique name of the component."`
	Type         string   `json:"type" jsonschema:"The type of the component (e.g., state.redis, pubsub.redis)."`
	Version      string   `json:"version,omitempty" jsonschema:"The version of the Component (e.g., v1)."`
	Capabilities []string `json:"capabilities" jsonschema:"The capabilities of the Component."`
}

var (
	daprClient dapr.Client
)

func GetLiveComponentList(ctx context.Context, client dapr.Client) ([]ComponentInfo, error) {
	metadata, err := client.GetMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dapr metadata: %w", err)
	}

	var components []ComponentInfo
	for _, component := range metadata.RegisteredComponents {
		if strings.Contains(component.Type, "pubsub") ||
			strings.Contains(component.Type, "state") ||
			strings.Contains(component.Type, "binding") ||
			strings.Contains(component.Type, "conversation") ||
			strings.Contains(component.Type, "secretstores") ||
			strings.Contains(component.Type, "lock") ||
			strings.Contains(component.Type, "cryptography") {

			capabilities := component.Capabilities
			if capabilities == nil {
				capabilities = []string{}
			}

			components = append(components, ComponentInfo{
				Name:         component.Name,
				Type:         component.Type,
				Version:      component.Version,
				Capabilities: capabilities,
			})
		}
	}
	return components, nil
}

func getMetadataTool(ctx context.Context, req *mcp.CallToolRequest, args any) (
	*mcp.CallToolResult,
	ComponentListWrapper,
	error,
) {
	if daprClient == nil {
		toolErrorMessage := "Dapr client not initialized on the server side."
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, ComponentListWrapper{}, nil
	}
	log.Printf("Request: %v", req)

	components, err := GetLiveComponentList(ctx, daprClient)
	if err != nil {
		log.Printf("Error calling getMetadataTool: %v", err)
		toolErrorMessage := fmt.Sprintf("Error fetching live Dapr component list: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, ComponentListWrapper{}, nil
	}
	log.Printf("Components: %s", components)

	wrapper := ComponentListWrapper{
		Components: components,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf("Successfully retrieved %d Dapr component(s). The details are returned in the structured result.", len(components)),
		}},
	}, wrapper, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_components",
		Title:       "Retrieve Live Dapr Component List",
		Description: "Retrieves a detailed list of all currently running Dapr components (state stores, pub/sub brokers, etc.) in the sidecar. Use the structured result of this call to find valid component names (e.g., 'statestore-redis') for use in other tool calls.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, getMetadataTool)
}
