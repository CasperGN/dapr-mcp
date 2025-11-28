package cryptography

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EncryptArgs struct {
	ComponentName string `json:"componentName" jsonschema:"The name of the Dapr Cryptography component."`
	KeyName       string `json:"keyName" jsonschema:"The name of the key to use for encryption, stored in the component/vault."`
	Algorithm     string `json:"algorithm" jsonschema:"The algorithm used for wrapping the key (e.g., 'RSA', 'AES')."`
	PlainText     string `json:"plainText" jsonschema:"The plain text message to be encrypted."`
}

type DecryptArgs struct {
	ComponentName string `json:"componentName" jsonschema:"The name of the Dapr Cryptography component."`
	CipherText    string `json:"cipherText" jsonschema:"The base64-encoded encrypted message to be decrypted."`
	KeyName       string `json:"keyName,omitempty" jsonschema:"Optional: The name of the key to use for decryption, if not embedded in the cipher text header."`
}

var daprClient dapr.Client

func encryptTool(ctx context.Context, req *mcp.CallToolRequest, args EncryptArgs) (*mcp.CallToolResult, any, error) {
	plainStream := strings.NewReader(args.PlainText)

	encryptOpts := dapr.EncryptOptions{
		ComponentName:    args.ComponentName,
		KeyName:          args.KeyName,
		KeyWrapAlgorithm: args.Algorithm,
	}

	cipherStream, err := daprClient.Encrypt(ctx, plainStream, encryptOpts)
	if err != nil {
		log.Printf("Dapr Encrypt failed: %v", err)
		toolErrorMessage := fmt.Errorf("dapr Encrypt failed: %w", err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	var cipherBuf bytes.Buffer
	if _, err := io.Copy(&cipherBuf, cipherStream); err != nil {
		toolErrorMessage := fmt.Errorf("failed to read encrypted stream: %w", err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	cipherText := cipherBuf.String()

	successMessage := fmt.Sprintf(
		"Successfully encrypted message using component '%s' and key '%s'. Cipher Text is returned in the tool result.",
		args.ComponentName,
		args.KeyName,
	)
	log.Println(successMessage)
	structuredResult := map[string]string{
		"cipher_text": cipherText,
		"key_name":    args.KeyName,
		"algorithm":   args.Algorithm,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

func decryptTool(ctx context.Context, req *mcp.CallToolRequest, args DecryptArgs) (*mcp.CallToolResult, any, error) {
	cipherStream := strings.NewReader(args.CipherText)

	decryptOpts := dapr.DecryptOptions{
		ComponentName: args.ComponentName,
	}
	if args.KeyName != "" {
		decryptOpts.KeyName = args.KeyName
	}

	plainStream, err := daprClient.Decrypt(ctx, cipherStream, decryptOpts)
	if err != nil {
		log.Printf("Dapr Decrypt failed: %v", err)
		toolErrorMessage := fmt.Errorf("dapr Decrypt failed: %v", err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	var plainBuf bytes.Buffer
	if _, err := io.Copy(&plainBuf, plainStream); err != nil {
		toolErrorMessage := fmt.Errorf("failed to read decrypted stream: %w", err).Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: toolErrorMessage}},
			IsError: true,
		}, nil, nil
	}

	plainText := plainBuf.String()

	successMessage := fmt.Sprintf(
		"Successfully decrypted message using component '%s'. Plain text is returned in the tool result.",
		args.ComponentName,
	)
	log.Println(successMessage)
	structuredResult := map[string]string{
		"plain_text":     plainText,
		"component_name": args.ComponentName,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: successMessage}},
	}, structuredResult, nil
}

func RegisterTools(server *mcp.Server, client dapr.Client) {
	daprClient = client

	// Encrypt Annotations
	notIdempotent := false
	isDestructive := true
	notReadOnly := false
	isOpenWorld := true

	// Decrypt Annotations
	isIdempotent := true
	isReadOnly := true
	notDestructive := false

	mcp.AddTool(server, &mcp.Tool{
		Name:  "encrypt_data",
		Title: "Encrypt Sensitive Data for Confidentiality",
		Description: "Encrypts arbitrary plain text data using a specified key and algorithm from a Dapr cryptography component. **This is a SIDE-EFFECT action (mutates data form) that is NOT IDEMPOTENT.** Use ONLY when the user explicitly says 'encrypt this' or 'store this encrypted'.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide non-empty values for `ComponentName`, `KeyName`, `Algorithm`, and `PlainText`.\n" +
			"2. **NEVER INVENT**: You must NOT invent `KeyName` or `Algorithm` names; they must be provided by the user or retrieved from another source.\n" +
			"3. **CLARIFICATION**: If any required input is missing, you MUST ask the user for clarification.\n" +
			"4. **WORKFLOW RULE**: The output from `encrypt_data` is the ciphertext to be used in subsequent storage or publication steps. DO NOT look up keys using the secret store unless explicitly authorized.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &isDestructive,
			ReadOnlyHint:    notReadOnly,
			IdempotentHint:  notIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, encryptTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:  "decrypt_data",
		Title: "Decrypt Sensitive Cipher Text",
		Description: "Decrypts encrypted cipher text data back into its original plain text form. **This is a Data Retrieval operation (Read-Only) that IS IDEMPOTENT.** Use ONLY when the user explicitly requests to read data that was previously encrypted.\n\n" +
			"**ARGUMENT RULES:**\n" +
			"1. **REQUIRED INPUTS**: You MUST provide non-empty values for `ComponentName` and `CipherText`.\n" +
			"2. **CLARIFICATION**: If the required key is not embedded in the ciphertext header, you MUST ask the user for the explicit `KeyName`.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &notDestructive,
			ReadOnlyHint:    isReadOnly,
			IdempotentHint:  isIdempotent,
			OpenWorldHint:   &isOpenWorld,
		},
	}, decryptTool)
}
