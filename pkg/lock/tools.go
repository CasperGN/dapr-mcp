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

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, resp.Success, nil
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: finalMessage}},
	}, statusMessage, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client
	mcp.AddTool(server, &mcp.Tool{
		Name:        "acquire_lock",
		Description: "Tries to acquire a distributed lock on a named resource for a specific duration. Essential for coordination in distributed systems.",
	}, acquireLockTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_lock",
		Description: "Releases a distributed lock on a resource. Only the owner who acquired the lock can release it.",
	}, releaseLockTool)
}
