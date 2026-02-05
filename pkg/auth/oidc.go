// Package auth provides authentication and authorization for the MCP server.
package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCAuthenticator authenticates requests using OIDC tokens.
type OIDCAuthenticator struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   OIDCConfig
}

// NewOIDCAuthenticator creates a new OIDC authenticator.
func NewOIDCAuthenticator(ctx context.Context, cfg OIDCConfig) (*OIDCAuthenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifierConfig := &oidc.Config{
		ClientID:             cfg.ClientID,
		SkipIssuerCheck:      cfg.SkipIssuerCheck,
		SupportedSigningAlgs: cfg.AllowedAlgorithms,
	}

	verifier := provider.Verifier(verifierConfig)

	return &OIDCAuthenticator{
		provider: provider,
		verifier: verifier,
		config:   cfg,
	}, nil
}

// Authenticate validates an OIDC token and returns the identity.
func (a *OIDCAuthenticator) Authenticate(ctx context.Context, token string) (*Identity, error) {
	idToken, err := a.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		Audience      []string `json:"aud"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Get all claims as a map
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		allClaims = make(map[string]interface{})
	}

	return &Identity{
		Subject:    idToken.Subject,
		Issuer:     idToken.Issuer,
		Audience:   idToken.Audience,
		Email:      claims.Email,
		Name:       claims.Name,
		Claims:     allClaims,
		AuthMethod: ModeOIDC,
	}, nil
}

// Mode returns the authentication mode.
func (a *OIDCAuthenticator) Mode() AuthMode {
	return ModeOIDC
}
