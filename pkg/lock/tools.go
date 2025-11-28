package lock

import (
	"context"
	"fmt"
	"log"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AcquireLockArgs struct {
	StoreName       string `json:"storeName" jsonschema:"The name of the Dapr lock store component (e.g., 'redis-lock')."`
	ResourceID      string `json:"resourceID" jsonschema:"The unique name of the resource to lock (e.g., 'inventory-update-lock')."`
	LockOwner       string `json:"lockOwner" jsonschema:"A unique identifier for the entity trying to acquire the lock (e.g., 'ai-agent-42')."`
	ExpiryInSeconds int32  `json:"expiryInSeconds" jsonschema:"The lock duration in seconds. If not released, the lock will automatically expire after this time (recommended to set between 5 and 60 seconds)."`
}

type ReleaseLockArgs struct {
	StoreName  string `json:"storeName" jsonschema:"The name of the Dapr lock store component."`
	ResourceID string `json:"resourceID" jsonschema:"The unique name of the resource whose lock should be released."`
	LockOwner  string `json:"lockOwner" jsonschema:"The unique identifier of the entity that currently holds the lock."`
}

var daprClient dapr.Client

func acquireLockTool(ctx context.Context, req *mcp.CallToolRequest, args AcquireLockArgs) (*mcp.CallToolResult, any, error) {
	lockReq := &dapr.LockRequest{
		LockOwner:       args.LockOwner,
		ResourceID:      args.ResourceID,
		ExpiryInSeconds: args.ExpiryInSeconds,
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := daprClient.TryLockAlpha1(rpcCtx, args.StoreName, lockReq)
	if err != nil {
		log.Printf("Dapr TryLockAlpha1 failed: %v", err)
		return nil, nil, fmt.Errorf("dapr API error while trying to acquire lock: %w", err)
	}

	var successMessage string

	if resp.Success {
		successMessage = fmt.Sprintf("Successfully **acquired** lock for resource **'%s'** on store '%s'. Owner: %s. Expires in %d seconds.",
			args.ResourceID, args.StoreName, args.LockOwner, args.ExpiryInSeconds)
	} else {
		successMessage = fmt.Sprintf("Failed to acquire lock for resource **'%s'** on store '%s'. The lock is currently held by another entity.",
			args.ResourceID, args.StoreName)
	}

	log.Println(successMessage)
	structuredResult := map[string]interface{}{
		"lock_acquired": resp.Success,
		"resource_id":   args.ResourceID,
		"owner_id":      args.LockOwner,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

func releaseLockTool(ctx context.Context, req *mcp.CallToolRequest, args ReleaseLockArgs) (*mcp.CallToolResult, any, error) {
	unlockReq := &dapr.UnlockRequest{
		LockOwner:  args.LockOwner,
		ResourceID: args.ResourceID,
	}

	resp, err := daprClient.UnlockAlpha1(ctx, args.StoreName, unlockReq)
	if err != nil {
		log.Printf("Dapr UnlockAlpha1 failed: %v", err)
		return nil, nil, fmt.Errorf("dapr API error while trying to release lock: %w", err)
	}

	var statusMessage string

	const (
		StatusSuccess            = "SUCCESS"
		StatusLockUnexist        = "LOCK_UNEXIST"
		StatusLockBelongToOthers = "LOCK_BELONG_TO_OTHERS"
		StatusInternalError      = "INTERNAL_ERROR"
	)

	switch resp.Status {
	case StatusSuccess:
		statusMessage = "SUCCESS: The lock was successfully released."
	case StatusLockUnexist:
		statusMessage = "LOCK_UNEXIST: The lock specified by ResourceID does not exist."
	case StatusLockBelongToOthers:
		statusMessage = fmt.Sprintf("LOCK_BELONG_TO_OTHERS: The lock is held by a different owner. Cannot be released by owner '%s'.", args.LockOwner)
	case StatusInternalError:
		statusMessage = "INTERNAL_ERROR: An internal error occurred in the lock component."
	default:
		statusMessage = fmt.Sprintf("UNKNOWN_STATUS: %s", resp.Status)
	}

	finalMessage := fmt.Sprintf("Attempted to release lock on resource '%s' (Owner: %s). Result: %s", args.ResourceID, args.LockOwner, statusMessage)

	log.Println(finalMessage)
	structuredResult := map[string]interface{}{
		"release_status_code": resp.Status,
		"release_status_text": statusMessage,
		"resource_id":         args.ResourceID,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: finalMessage}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "acquire_lock",
		Title:       "Acquire Resource Coordination Lock",
		Description: "Tries to acquire a distributed lock on a named resource for exclusive access. **This is a SIDE-EFFECT action used for critical coordination and concurrency control.** Use only when the agent must ensure no other entity is concurrently modifying a shared resource (e.g., before writing to a database). Requires the store name, unique resource ID, owner ID, and a short expiry time in seconds.",
	}, acquireLockTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_lock",
		Title:       "Release Resource Coordination Lock",
		Description: "Releases a previously acquired distributed lock on a resource. **This is a SIDE-EFFECT action.** It MUST be called immediately after the critical section of code is complete to prevent deadlocks. Only the original owner can release the lock. Requires the store name, resource ID, and lock owner ID.",
	}, releaseLockTool)
}
