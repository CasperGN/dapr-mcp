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

var ComponentCache []ComponentInfo
var daprClient dapr.Client

func getLiveComponentList(ctx context.Context) ([]ComponentInfo, error) {
	metadata, err := daprClient.GetMetadata(ctx)
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

			components = append(components, ComponentInfo{
				Name:         component.Name,
				Type:         component.Type,
				Version:      component.Version,
				Capabilities: component.Capabilities,
			})
		}
	}
	return components, nil
}

func GetDynamicInstructions(ctx context.Context, client dapr.Client) string {
	daprClient = client
	components, err := getLiveComponentList(ctx)
	if err != nil {
		log.Printf("Warning: Could not fetch live Dapr component metadata: %v. Using static instructions.", err)
		return "You are an AI assistant. Use the available Dapr tools. Could not fetch live Dapr component metadata, so you must assume the component names are 'pubsub', 'statestore', 'secretstore', and 'lock'."
	}

	var stateStores []string
	var pubsubBrokers []string
	var bindings []string
	var conversations []string
	var secretStores []string
	var locks []string
	var cryptographies []string

	ComponentCache = components

	for _, comp := range components {
		compString := fmt.Sprintf("'%s' (Type: %s, Version: %s, Capabilities: %s)",
			comp.Name,
			strings.TrimPrefix(comp.Type, comp.Type[:strings.Index(comp.Type, ".")]+"."), // e.g., 'redis' instead of 'state.redis'
			comp.Version,
			strings.Join(comp.Capabilities, ", "))

		switch {
		case strings.HasPrefix(comp.Type, "state."):
			stateStores = append(stateStores, compString)
		case strings.HasPrefix(comp.Type, "pubsub."):
			pubsubBrokers = append(pubsubBrokers, compString)
		case strings.HasPrefix(comp.Type, "bindings."):
			bindings = append(bindings, compString)
		case strings.HasPrefix(comp.Type, "secretstores."):
			secretStores = append(secretStores, compString)
		case strings.HasPrefix(comp.Type, "lock."):
			locks = append(locks, compString)
		case strings.HasPrefix(comp.Type, "conversation."):
			conversations = append(conversations, compString)
		case strings.HasPrefix(comp.Type, "cryptography."):
			cryptographies = append(cryptographies, compString)
		default:
			log.Printf("Unknown or unhandled component type: %s", comp.Type)
		}
	}

	var body strings.Builder

	// 1. GLOBAL DIRECTIVES: Establish strict rules immediately.
	body.WriteString("You are an expert AI assistant for Dapr microservices. Your primary goal is to translate user requests into precise Dapr tool calls.\n")
	body.WriteString("When generating a tool call, you **MUST** adhere to the component names and argument types specified below.\n")
	body.WriteString("NEVER invent names or arguments not explicitly listed.\n\n")

	// --- State Stores ---
	if len(stateStores) > 0 {
		body.WriteString("## 💾 Available State Stores (for tools like `save_state`, `get_state`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(stateStores, ", ")))

		// 💡 FOOLPROOF SAVE INSTRUCTION: Explicitly addresses missing values and format.
		body.WriteString("   - **SAVE RULE**: To call `save_state`, you **MUST** provide three non-empty arguments: `storeName`, `key`, and **`value`**. The `value` argument must be the exact literal content provided by the user (e.g., `The new server is amazing!`). **DO NOT** use quotes or escape characters (`\"` or `\\`) within the content of the `value` string itself, as the JSON parser adds them automatically.\n\n")
		body.WriteString("   - **FETCH RULE**: To call `get_state`, you **MUST** provide `storeName` and `key`.\n\n")
	}

	// --- Pub/Sub Brokers ---
	if len(pubsubBrokers) > 0 {
		body.WriteString("## 📩 Available Pub/Sub Brokers (for tools like `publish_event`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(pubsubBrokers, ", ")))
		// 💡 FOOLPROOF PUBLISH INSTRUCTION: Clarify content requirement.
		body.WriteString("   - **PUBLISH RULE**: Both `publish_event` tools require non-empty `pubsubName`, `topic`, and `message`. The `message` MUST be the content the user wishes to publish.\n\n")
	}

	// --- Secret Stores ---
	if len(secretStores) > 0 {
		body.WriteString("## 🔑 Available Secret Stores (for tools like `get_secret`, `get_bulk_secrets`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(secretStores, ", ")))
		body.WriteString("   - **SECURITY RULE**: Always call `get_secret` with the most specific `secretName` possible. Avoid `get_bulk_secrets` unless necessary, as it retrieves all accessible secrets.\n\n")
	}

	// --- Distributed Locks ---
	if len(locks) > 0 {
		body.WriteString("## 🔒 Available Distributed Locks (for tools like `acquire_lock`, `release_lock`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(locks, ", ")))
		// 💡 FOOLPROOF LOCK INSTRUCTION: Emphasize ownership and required IDs.
		body.WriteString("   - **LOCK RULE**: `acquire_lock` and `release_lock` require non-empty `storeName`, `resourceID`, and a unique **`lockOwner`** ID.\n\n")
	}

	// --- Input/Output Bindings ---
	if len(bindings) > 0 {
		body.WriteString("## 🔗 Available Input/Output Bindings (for tools like `invoke_output_binding`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(bindings, ", ")))
		body.WriteString("   - **BINDING RULE**: `invoke_output_binding` MUST include a valid `bindingName` and a recognized `operation` (e.g., 'create', 'delete').\n\n")
	}

	// --- Conversation Models ---
	if len(conversations) > 0 {
		body.WriteString("## 💬 Available Conversation Models (for tools like `converse_with_llm`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(conversations, ", ")))
		body.WriteString("   - **CONVERSE RULE**: Use this tool to delegate complex reasoning. The `Inputs` argument must contain the full, current conversation history.\n\n")
	}

	// --- Cryptography Components ---
	if len(cryptographies) > 0 {
		body.WriteString("## 🛡️ Available Cryptography Components (for tools like `encrypt_data`, `decrypt_data`):\n")
		body.WriteString(fmt.Sprintf("- **Names**: %s\n", strings.Join(cryptographies, ", ")))
		body.WriteString("   - **CRYPTO RULE**: `encrypt_data` requires `keyName`, `algorithm`, and the `plainText` to be encrypted. Decryption relies on the ciphertext being valid.\n\n")
	}

	// 2. FAILSAFE INSTRUCTION: Final warning if environment is empty.
	if body.Len() < 200 {
		body.WriteString("\n\n**FAILSAFE WARNING**: No Dapr core components were detected. All tools that rely on a component (State, PubSub, etc.) will fail. Only basic Service and Actor invocation tools may be operational.")
	} else {
		body.WriteString("\n\n**FINAL RULE**: Before returning a text answer, always check if a tool call is necessary. If a tool call fails, inform the user about the failure and the error message (e.g., 'Dapr failed to save state because...').")
	}

	// ... (rest of logging and return remains the same) ...
	return body.String()
}

func getMetadataTool(ctx context.Context, req *mcp.CallToolRequest, args any) (
	*mcp.CallToolResult,
	ComponentListWrapper,
	error,
) {
	components, err := getLiveComponentList(ctx)
	if err != nil {
		log.Printf("Error calling getMetadataTool: %v", err)
		return nil, ComponentListWrapper{}, err
	}

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
		Description: "Retrieves a detailed list of all currently running Dapr components (state stores, pub/sub brokers, etc.) in the sidecar. Use this to confirm valid component names for other tools.",
	}, getMetadataTool)
}
