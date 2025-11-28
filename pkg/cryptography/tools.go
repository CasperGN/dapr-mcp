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
		return nil, nil, fmt.Errorf("dapr API error during encryption: %w", err)
	}

	var cipherBuf bytes.Buffer
	if _, err := io.Copy(&cipherBuf, cipherStream); err != nil {
		return nil, nil, fmt.Errorf("failed to read encrypted stream: %w", err)
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
		return nil, nil, fmt.Errorf("dapr API error during decryption: %w", err)
	}

	var plainBuf bytes.Buffer
	if _, err := io.Copy(&plainBuf, plainStream); err != nil {
		return nil, nil, fmt.Errorf("failed to read decrypted stream: %w", err)
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
	mcp.AddTool(server, &mcp.Tool{
		Name:        "encrypt_data",
		Title:       "Encrypt Sensitive Data for Confidentiality",
		Description: "Encrypts arbitrary plain text data using a specified key and algorithm from a Dapr cryptography component or vault. **This tool is used for ensuring confidentiality and is a critical security primitive.** It has a **SIDE EFFECT** on the data's representation. Use only when the task requires securing sensitive information before persistence or transmission. Requires component name, a whitelisted key name, and algorithm.",
	}, encryptTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "decrypt_data",
		Title:       "Decrypt Sensitive Cipher Text",
		Description: "Decrypts encrypted cipher text data back into its original plain text form using a Dapr cryptography component. **This tool is a critical security primitive and has a SIDE EFFECT** on the data's representation. Use only when the task requires reading data that was previously encrypted by the system. Note: The cipher text must have been generated using a compatible Dapr component.",
	}, decryptTool)
}
