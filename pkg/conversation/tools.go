package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var daprClient dapr.Client

func converseTool(ctx context.Context, req *mcp.CallToolRequest, args dapr.ConversationRequestAlpha2) (*mcp.CallToolResult, any, error) {
	converseReq := dapr.ConversationRequestAlpha2{
		Name:        args.Name,
		ContextID:   args.ContextID,
		Inputs:      args.Inputs,
		Parameters:  nil, // params
		Metadata:    nil, // metadata
		ScrubPII:    args.ScrubPII,
		Temperature: args.Temperature,
		Tools:       args.Tools,
		ToolChoice:  args.ToolChoice,
	}

	if len(args.Metadata) > 0 {
		converseReq.Metadata = args.Metadata
	}
	if len(args.Parameters) > 0 {
		converseReq.Parameters = args.Parameters
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
		Description: "Delegates a complex reasoning, summarization, or text generation task to a secondary Large Language Model (LLM) configured via a Dapr conversation component (e.g., OpenAI, Mistral). **This tool is Computational and Stateless (Read-Only).** Use this tool only when the primary agent needs to outsource a specialized task (e.g., generating code, complex reasoning, or creative writing).\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide the Dapr component `Name` and the full conversation history in the `Inputs` argument.\n" +
			"2. **NEVER INVENT**: You must NOT invent the component `Name`; it must be provided by the user or discovered via the `get_components` tool.\n" +
			"3. **HISTORY**: The `Inputs` argument MUST contain the full, current conversation history to maintain context during the delegation.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &isDestructive,
			ReadOnlyHint:    isReadOnly,
			IdempotentHint:  isIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, converseTool)
}
