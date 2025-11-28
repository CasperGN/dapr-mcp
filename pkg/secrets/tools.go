package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetSecretArgs struct {
	StoreName  string            `json:"storeName" jsonschema:"The name of the configured Dapr secret store component (e.g., 'vault')."`
	SecretName string            `json:"secretName" jsonschema:"The specific name of the secret to retrieve (e.g., 'db-credentials')."`
	Metadata   map[string]string `json:"metadata" jsonschema:"Optional per-request metadata (e.g., 'version_id'). Check the secret store documentation for supported fields."`
}

type GetBulkSecretArgs struct {
	StoreName string            `json:"storeName" jsonschema:"The name of the configured Dapr secret store component (e.g., 'vault')."`
	Metadata  map[string]string `json:"metadata" jsonschema:"Optional per-request metadata for the bulk retrieval operation."`
}

var daprClient dapr.Client

func getSecretTool(ctx context.Context, req *mcp.CallToolRequest, args GetSecretArgs) (*mcp.CallToolResult, map[string]string, error) {
	secrets, err := daprClient.GetSecret(ctx, args.StoreName, args.SecretName, args.Metadata)
	if err != nil {
		log.Printf("Dapr GetSecret failed: %v", err)
		toolErrorMessage := fmt.Errorf("failed to get secret '%s': %w", args.SecretName, err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	var secretKeys []string
	for key := range secrets {
		secretKeys = append(secretKeys, key)
	}

	secretsJSON, _ := json.MarshalIndent(secrets, "", "  ")

	successMessage := fmt.Sprintf(
		"Successfully retrieved secret '%s' from store '%s'. It contains the following key(s): %s.\n\nJSON Value:\n%s",
		args.SecretName,
		args.StoreName,
		strings.Join(secretKeys, ", "),
		string(secretsJSON),
	)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, secrets, nil
}

func getBulkSecretTool(ctx context.Context, req *mcp.CallToolRequest, args GetBulkSecretArgs) (*mcp.CallToolResult, map[string]map[string]string, error) {
	secretsBulk, err := daprClient.GetBulkSecret(ctx, args.StoreName, args.Metadata)
	if err != nil {
		log.Printf("Dapr GetBulkSecret failed: %v", err)
		toolErrorMessage := fmt.Errorf("failed to get bulk secrets from store '%s': %w", args.StoreName, err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	var secretNames []string
	for name := range secretsBulk {
		secretNames = append(secretNames, name)
	}

	secretsJSON, _ := json.MarshalIndent(secretsBulk, "", "  ")

	successMessage := fmt.Sprintf(
		"Successfully retrieved %d secret(s) in bulk from store '%s'. Names retrieved: %s.\n\nJSON Value:\n%s",
		len(secretsBulk),
		args.StoreName,
		strings.Join(secretNames, ", "),
		string(secretsJSON),
	)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, secretsBulk, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client

	isReadOnly := true
	isIdempotent := true
	notDestructive := false

	mcp.AddTool(server, &mcp.Tool{
		Name:  "get_secret",
		Title: "Retrieve Single Authorized Secret",
		Description: "Retrieves a single, whitelisted secret (e.g., API key, credential) from a configured Dapr secret store. **This is a highly SENSITIVE Data Retrieval operation.** Use ONLY when the user explicitly requests a specific secret.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide non-empty values for `StoreName` and `SecretName`.\n" +
			"2. **NEVER INVENT**: You must NOT invent `SecretName` or `StoreName` names; they must be provided by the user or discovered.\n" +
			"3. **CLARIFICATION**: If any required input is missing, you MUST ask the user for clarification.\n\n" +
			"**SECURITY WARNING**: This tool provides access to critical credentials. NEVER guess secret names, and NEVER store retrieved secrets without explicit authorization.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    isReadOnly,
			DestructiveHint: &notDestructive,
			IdempotentHint:  isIdempotent,
		},
	}, getSecretTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:  "get_bulk_secrets",
		Title: "Retrieve All Secrets (HIGHLY RESTRICTED)",
		Description: "Attempts to retrieve ALL secrets the application has access to from a specific Dapr secret store. **This operation is HIGHLY RESTRICTED and extremely high-risk.**\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide a non-empty value for `StoreName`.\n" +
			"2. **RISK WARNING**: Avoid this tool unless the user explicitly requests enumeration of all accessible secrets, as it provides a broad view of the system's credentials.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    isReadOnly,
			DestructiveHint: &notDestructive,
			IdempotentHint:  isIdempotent,
		},
	}, getBulkSecretTool)
}
