package metadata

import (
	"context"
	"fmt"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
)

type ComponentInfo struct {
	Name        string `json:"name" jsonschema:"The unique name of the component."`
	Type        string `json:"type" jsonschema:"The type of the component (e.g., state.redis, pubsub.redis)."`
	BindingType string `json:"bindingType,omitempty" jsonschema:"For bindings, 'input' or 'output'."`
}

func getLiveComponentList(ctx context.Context) ([]ComponentInfo, error) {
	client, err := dapr.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dapr client: %w", err)
	}
	defer client.Close()

	metadata, err := client.GetMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dapr metadata: %w", err)
	}

	var components []ComponentInfo
	for _, component := range metadata.RegisteredComponents {
		if strings.Contains(component.Type, "pubsub") ||
			strings.Contains(component.Type, "state") ||
			strings.Contains(component.Type, "bindings") {

			components = append(components, ComponentInfo{
				Name: component.Name,
				Type: component.Type,
			})
		}
	}
	return components, nil
}

func GetDynamicInstructions(ctx context.Context) string {
	components, err := getLiveComponentList(ctx)
	if err != nil {
		log.Printf("Warning: Could not fetch live Dapr component metadata: %v. Using static instructions.", err)
		return "You are an AI assistant. Use the available Dapr tools. Could not fetch live Dapr component metadata, so you must assume the component names are 'pubsub' and 'statestore'."
	}

	var stateStores []string
	var pubsubBrokers []string
	var bindings []string

	for _, comp := range components {
		if strings.Contains(comp.Type, "state.") {
			stateStores = append(stateStores, fmt.Sprintf("'%s' (Type: %s)", comp.Name, strings.TrimPrefix(comp.Type, "state.")))
		} else if strings.Contains(comp.Type, "pubsub.") {
			pubsubBrokers = append(pubsubBrokers, fmt.Sprintf("'%s' (Type: %s)", comp.Name, strings.TrimPrefix(comp.Type, "pubsub.")))
		} else if strings.Contains(comp.Type, "bindings.") {
			bindings = append(bindings, fmt.Sprintf("'%s' (Type: %s)", comp.Name, strings.TrimPrefix(comp.Type, "bindings.")))
		}
	}

	var body strings.Builder

	body.WriteString("You are an AI assistant capable of interacting with Dapr via the provided tools.\n")
	body.WriteString("When calling a tool, you MUST use one of the names listed below for the respective component argument.\n\n")

	if len(stateStores) > 0 {
		body.WriteString("## Available State Stores (for tools like `save_state`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n\n", strings.Join(stateStores, ", ")))
	}

	if len(pubsubBrokers) > 0 {
		body.WriteString("## Available Pub/Sub Brokers (for tools like `publish_event`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n\n", strings.Join(pubsubBrokers, ", ")))
	}

	if len(bindings) > 0 {
		body.WriteString("## Available Input/Output Bindings (for tools like `invoke_binding`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n\n", strings.Join(bindings, ", ")))
	}

	if body.Len() < 100 {
		body.WriteString("No Dapr components (State Stores, Pub/Sub Brokers, or Bindings) were detected in the running sidecar.")
	}

	return body.String()
}
