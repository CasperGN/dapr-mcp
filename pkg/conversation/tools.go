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
		return nil, nil, fmt.Errorf("dapr API error while conversing with LLM '%s': %w", args.Name, err)
	}

	if len(resp.Outputs) == 0 {
		return nil, nil, fmt.Errorf("LLM '%s' returned an empty outputs list", args.Name)
	}
	lastOutput := resp.Outputs[len(resp.Outputs)-1]

	if len(lastOutput.Choices) == 0 {
		return nil, nil, fmt.Errorf("LLM '%s' returned no choices in the last output", args.Name)
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
	mcp.AddTool(server, &mcp.Tool{
		Name:        "converse_with_llm",
		Title:       "Delegate Task to External Reasoning Engine",
		Description: "Delegates a complex reasoning, summarization, or text generation task to a secondary Large Language Model (LLM) configured via a Dapr conversation component (e.g., OpenAI, Mistral). **This tool is Computational and Stateless (no side effect).** Use this only when the primary agent needs to outsource a specialized task, like: generating code, performing a creative writing exercise, or utilizing a model with a different capability set. Requires the Dapr component 'Name' and the list of 'Inputs' (messages).",
	}, converseTool)
}
