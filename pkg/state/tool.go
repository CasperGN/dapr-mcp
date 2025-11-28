package state

import (
	"context"
	"fmt"
	"log"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SaveStateArgs struct {
	StoreName string `json:"storeName" jsonschema:"The name of the Dapr state store component (e.g., 'statestore')."`
	Key       string `json:"key" jsonschema:"The key under which to save the state."`
	Value     string `json:"value" jsonschema:"The value (typically a JSON string) to save."`
}

type GetStateArgs struct {
	StoreName string `json:"storeName" jsonschema:"The name of the Dapr state store component (e.g., 'statestore')."`
	Key       string `json:"key" jsonschema:"The key whose value should be retrieved."`
}

type DeleteStateArgs struct {
	StoreName string `json:"storeName" jsonschema:"The name of the Dapr state store component (e.g., 'statestore')."`
	Key       string `json:"key" jsonschema:"The key to delete."`
}

type TransactionItem struct {
	Key      string `json:"key" jsonschema:"The state key."`
	Value    string `json:"value" jsonschema:"The value to set (or empty for delete)."`
	IsDelete bool   `json:"isDelete" jsonschema:"Set to true to delete the key, false to save/update it."`
}

type ExecuteTransactionArgs struct {
	StoreName string            `json:"storeName" jsonschema:"The name of the Dapr state store component."`
	Items     []TransactionItem `json:"items" jsonschema:"A list of save and/or delete operations to execute atomically."`
}

var daprClient dapr.Client

func saveStateTool(ctx context.Context, req *mcp.CallToolRequest, args SaveStateArgs) (*mcp.CallToolResult, any, error) {
	data := []byte(args.Value)

	log.Printf("DEBUG: State Save requested. Store: %s, Key: %s, Payload Size: %d",
		args.StoreName, args.Key, len(data))

	var err error

	if err = daprClient.SaveState(ctx, args.StoreName, args.Key, data, nil); err == nil {
		successMessage := fmt.Sprintf("Successfully saved key '%s' to state store '%s'.", args.Key, args.StoreName)
		log.Println(successMessage)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
		}, map[string]string{"key_saved": args.Key, "store_name": args.StoreName}, nil
	}
	return nil, nil, fmt.Errorf("failed to save state to store '%s'. Final error: %v",
		args.StoreName, err)
}

func getStateTool(ctx context.Context, req *mcp.CallToolRequest, args GetStateArgs) (*mcp.CallToolResult, any, error) {
	item, err := daprClient.GetState(ctx, args.StoreName, args.Key, nil)
	if err != nil {
		log.Printf("Dapr GetState failed: %v", err)
		return nil, nil, fmt.Errorf("failed to get state: %w", err)
	}

	result := string(item.Value)
	log.Println(result)

	var structuredResult map[string]string

	if result == "" {
		result = fmt.Sprintf("Key '%s' not found in state store '%s'.", args.Key, args.StoreName)
		structuredResult = nil
	} else {
		result = fmt.Sprintf("Retrieved key '%s' from '%s'. Value:\n%s", args.Key, args.StoreName, result)
		structuredResult = map[string]string{
			"key":   args.Key,
			"value": string(item.Value),
		}
	}

	log.Println(result)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, structuredResult, nil
}

func deleteStateTool(ctx context.Context, req *mcp.CallToolRequest, args DeleteStateArgs) (*mcp.CallToolResult, any, error) {
	if err := daprClient.DeleteState(ctx, args.StoreName, args.Key, nil); err != nil {
		log.Printf("Dapr DeleteState failed: %v", err)
		return nil, nil, fmt.Errorf("failed to delete state: %w", err)
	}

	successMessage := fmt.Sprintf("Successfully deleted key '%s' from state store '%s'.", args.Key, args.StoreName)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, map[string]string{"key_deleted": args.Key, "store_name": args.StoreName}, nil
}

func executeTransactionTool(ctx context.Context, req *mcp.CallToolRequest, args ExecuteTransactionArgs) (*mcp.CallToolResult, any, error) {
	ops := make([]*dapr.StateOperation, 0)

	for _, item := range args.Items {
		var opType dapr.OperationType
		var setItem *dapr.SetStateItem

		if item.IsDelete {
			opType = dapr.StateOperationTypeDelete
			setItem = &dapr.SetStateItem{Key: item.Key}
		} else {
			opType = dapr.StateOperationTypeUpsert
			setItem = &dapr.SetStateItem{
				Key:   item.Key,
				Value: []byte(item.Value),
			}
		}

		ops = append(ops, &dapr.StateOperation{
			Type: opType,
			Item: setItem,
		})
	}

	if err := daprClient.ExecuteStateTransaction(ctx, args.StoreName, nil, ops); err != nil {
		log.Printf("Dapr ExecuteStateTransaction failed: %v", err)
		return nil, nil, fmt.Errorf("failed to execute state transaction: %w", err)
	}

	successMessage := fmt.Sprintf("Successfully executed %d state operations in a transaction on store '%s'.", len(args.Items), args.StoreName)
	log.Println(successMessage)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, map[string]interface{}{"operations_executed": len(args.Items), "store_name": args.StoreName}, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_state",
		Title:       "Save Single Key-Value State",
		Description: "Saves a single key-value pair to a Dapr state store. **This is a SIDE-EFFECT action that alters application state.** Use only when the agent needs to persist data or update an entity. The value MUST be a string, typically a JSON object. Requires a whitelisted store name and key.",
	}, saveStateTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_state",
		Title:       "Retrieve Single Key State",
		Description: "Retrieves the value for a single key from a Dapr state store. **This is a Data Retrieval operation and is non-mutating.** Use to access current application state or previously saved context. Requires a whitelisted store name and key.",
	}, getStateTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_state",
		Title:       "Delete State Key",
		Description: "Deletes a key-value pair from a Dapr state store. **This is a critical SIDE-EFFECT action that should be used with caution to avoid data loss.** Use only when instructed to remove specific, whitelisted application data. Requires a whitelisted store name and key.",
	}, deleteStateTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "execute_transaction",
		Title:       "Execute Atomic State Transaction",
		Description: "Executes multiple save and/or delete operations atomically (all or nothing) on state stores that support transactions. **This is a complex, high-impact SIDE-EFFECT action.** Use only for batch updates or when strict data consistency is required across multiple keys. ALL keys in the transaction must be explicitly whitelisted.",
	}, executeTransactionTool)
}
