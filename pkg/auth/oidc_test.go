package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOIDCAuthenticator_Mode(t *testing.T) {
	auth := &OIDCAuthenticator{
		config: OIDCConfig{
			IssuerURL: "https://issuer.example.com",
			ClientID:  "client-id",
		},
	}

	mode := auth.Mode()

	assert.Equal(t, ModeOIDC, mode)
}

func TestOIDCConfigStruct(t *testing.T) {
	cfg := OIDCConfig{
		Enabled:           true,
		IssuerURL:         "https://issuer.example.com",
		ClientID:          "my-client-id",
		AllowedAlgorithms: []string{"RS256", "ES256"},
		SkipIssuerCheck:   true,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "https://issuer.example.com", cfg.IssuerURL)
	assert.Equal(t, "my-client-id", cfg.ClientID)
	assert.Contains(t, cfg.AllowedAlgorithms, "RS256")
	assert.Contains(t, cfg.AllowedAlgorithms, "ES256")
	assert.True(t, cfg.SkipIssuerCheck)
}

func TestOIDCAuthenticatorStruct(t *testing.T) {
	auth := &OIDCAuthenticator{
		provider: nil, // Would be set by NewOIDCAuthenticator
		verifier: nil, // Would be set by NewOIDCAuthenticator
		config: OIDCConfig{
			Enabled:   true,
			IssuerURL: "https://issuer.example.com",
			ClientID:  "client-id",
		},
	}

	assert.Nil(t, auth.provider)
	assert.Nil(t, auth.verifier)
	assert.True(t, auth.config.Enabled)
}

// Note: Testing NewOIDCAuthenticator and Authenticate would require
// mocking the OIDC provider, which is complex. These tests focus on
// the simpler methods and struct validation.

func TestOIDCConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  OIDCConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: OIDCConfig{
				Enabled:   true,
				IssuerURL: "https://issuer.example.com",
				ClientID:  "client-id",
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: OIDCConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing issuer",
			config: OIDCConfig{
				Enabled:  true,
				ClientID: "client-id",
			},
			wantErr: true,
		},
		{
			name: "missing client ID",
			config: OIDCConfig{
				Enabled:   true,
				IssuerURL: "https://issuer.example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOIDCConfigDefaults(t *testing.T) {
	cfg := OIDCConfig{}

	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.IssuerURL)
	assert.Empty(t, cfg.ClientID)
	assert.Nil(t, cfg.AllowedAlgorithms)
	assert.False(t, cfg.SkipIssuerCheck)
}

func TestOIDCAuthenticatorModeReturnsCorrectValue(t *testing.T) {
	auth := &OIDCAuthenticator{}

	// Mode should always return ModeOIDC regardless of config
	assert.Equal(t, ModeOIDC, auth.Mode())
}
