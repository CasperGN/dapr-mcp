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
		return nil, nil, fmt.Errorf("failed to get secret '%s': %w", args.SecretName, err)
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
		return nil, nil, fmt.Errorf("failed to get bulk secrets from store '%s': %w", args.StoreName, err)
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
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_secret",
		Title:       "Retrieve Single Authorized Secret",
		Description: "Retrieves a single, whitelisted secret (e.g., API key, credential) from a configured Dapr secret store. **This is a highly SENSITIVE Data Retrieval operation. Do NOT use unless explicitly asked to retrieve a known, authorized secret.** Requires the store name and the specific secret name.",
	}, getSecretTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_bulk_secrets",
		Title:       "Retrieve All Secrets (HIGHLY RESTRICTED)",
		Description: "Attempts to retrieve ALL secrets that the application has access to from a specific Dapr secret store. **This operation is HIGHLY RESTRICTED due to its massive security risk and is likely disabled in production.** Only use if the request explicitly asks to enumerate all available secrets and you have been authorized to use the specific store.",
	}, getBulkSecretTool)
}
