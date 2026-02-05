package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/types/known/anypb"
)

type ConverseArgs struct {
	Name        string  `json:"name" jsonschema:"The Dapr component name of the LLM service (e.g., 'ollama', 'openai')."`
	Prompt      string  `json:"prompt" jsonschema:"The user's direct question or instruction to the LLM."`
	ContextID   string  `json:"contextId,omitempty" jsonschema:"Optional: Unique ID for continuing a specific conversation context/history."`
	Temperature float64 `json:"temperature,omitempty" jsonschema:"Optional: LLM temperature setting (0.0 to 1.0). Default is 0.7."`
}

var daprClient dapr.Client

func converseTool(ctx context.Context, req *mcp.CallToolRequest, args ConverseArgs) (*mcp.CallToolResult, any, error) {
	ctx, span := otel.Tracer("dapr-mcp-server").Start(ctx, "converse")
	defer span.End()

	contextID := args.ContextID
	if contextID == "" {
		newUUID, err := uuid.NewRandom()
		if err != nil {
			log.Printf("Failed to generate UUID: %v", err)
			contextID = "default-session-id"
		} else {
			contextID = newUUID.String()
			log.Printf("Generated new ContextID: %s", contextID)
		}
	}

	var contextIDPtr *string
	if contextID != "" {
		contextIDPtr = &contextID
	}

	temperature := args.Temperature
	if temperature == 0.0 {
		temperature = 0.7
	}
	temperaturePtr := &temperature

	scrubPIIFalse := false
	scrubPIIPtr := &scrubPIIFalse

	messageContent := dapr.ConversationMessageAlpha2{
		ConversationMessageOfUser: &dapr.ConversationMessageOfUserAlpha2{
			Content: []*dapr.ConversationMessageContentAlpha2{{Text: &args.Prompt}},
		},
	}

	inputs := []*dapr.ConversationInputAlpha2{
		{
			Messages: []*dapr.ConversationMessageAlpha2{&messageContent},
		},
	}

	params := make(map[string]*anypb.Any)
	metadata := make(map[string]string)
	tools := make([]*dapr.ConversationToolsAlpha2, 0)
	toolChoice := dapr.ToolChoiceNoneAlpha2

	converseReq := dapr.ConversationRequestAlpha2{
		Name:        args.Name,
		ContextID:   contextIDPtr,
		Inputs:      inputs,
		ScrubPII:    scrubPIIPtr,
		Temperature: temperaturePtr,
		Parameters:  params,
		Metadata:    metadata,
		Tools:       tools,
		ToolChoice:  &toolChoice,
	}

	resp, err := daprClient.ConverseAlpha2(ctx, converseReq)
	if err != nil {
		log.Printf("Dapr Converse failed: %v", err)
		toolErrorMessage := fmt.Errorf("dapr API error while conversing with LLM '%s': %w", args.Name, err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	if len(resp.Outputs) == 0 {
		toolErrorMessage := fmt.Sprintf("LLM '%s' returned an empty outputs list", args.Name)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}
	lastOutput := resp.Outputs[len(resp.Outputs)-1]

	if len(lastOutput.Choices) == 0 {
		toolErrorMessage := fmt.Sprintf("LLM '%s' returned no choices in the last output", args.Name)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf(
		"LLM Conversation completed successfully with component '%s'.\n",
		args.Name,
	))

	for i, choice := range lastOutput.Choices {
		if choice.Message == nil {
			continue
		}

		result.WriteString(fmt.Sprintf("\n--- Choice %d ---\n", i))

		if len(choice.Message.ToolCalls) > 0 {
			result.WriteString(fmt.Sprintf("Status: **TOOL CALL** (Reason: %s)\n", choice.FinishReason))

			toolCallsJson, _ := json.MarshalIndent(choice.Message.ToolCalls, "", "  ")
			result.WriteString(fmt.Sprintf("Tool Calls:\n%s\n", toolCallsJson))
		}

		if choice.Message.Content != "" {
			result.WriteString(fmt.Sprintf("Status: **MESSAGE** (Reason: %s)\n", choice.FinishReason))
			result.WriteString(fmt.Sprintf("Response Content:\n%s\n", choice.Message.Content))
		}
	}

	finalMessage := result.String()
	log.Println(finalMessage)

	responseJSON, _ := json.Marshal(resp)

	var structuredResult map[string]interface{}

	if err := json.Unmarshal(responseJSON, &structuredResult); err != nil {
		log.Printf("Warning: Failed to unmarshal response into structured map: %v", err)
		structuredResult = nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: finalMessage}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client

	isDestructive := false
	isReadOnly := true
	isIdempotent := true
	isOpenWorld := true

	mcp.AddTool(server, &mcp.Tool{
		Name:  "converse_with_llm",
		Title: "Delegate Task to External Reasoning Engine",
		Description: "Delegates a single, immediate reasoning or text generation task to a secondary LLM component. The server handles complex message history formatting internally, accepting only the user's direct prompt, the component name, and an optional context ID for session continuity.\n\n" +
			"**GUIDANCE:**\n" +
			"1. Use `get_components` to find the `name` of the LLM component.\n" +
			"2. For `Temperature`, use a value between 0.0 (deterministic) and 1.0 (creative). Default is 0.7.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide the Dapr component `name` and the user's `prompt`.\n" +
			"2. **NEVER INVENT**: You must NOT invent the component `name`; it must be provided by the user or discovered via the `get_components` tool.\n" +
			"3. **CONTEXT**: If provided, the `contextId` is used to maintain history. If omitted, a new session is started.",

		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &isDestructive,
			ReadOnlyHint:    isReadOnly,
			IdempotentHint:  isIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, converseTool)
}
