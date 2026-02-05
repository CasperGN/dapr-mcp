// Package auth provides authentication and authorization for the MCP server.
package auth

import (
	"context"
	"errors"
)

// AuthMode represents the authentication mode.
type AuthMode string

const (
	// ModeDisabled disables authentication (for local development).
	ModeDisabled AuthMode = "disabled"
	// ModeOIDC uses OAuth2.0/OIDC for authentication.
	ModeOIDC AuthMode = "oidc"
	// ModeSPIFFE uses SPIFFE JWT-SVIDs for service-to-service auth.
	ModeSPIFFE AuthMode = "spiffe"
	// ModeHybrid accepts both OIDC and SPIFFE tokens.
	ModeHybrid AuthMode = "hybrid"
	// ModeDaprSentry uses Dapr Sentry JWT tokens for authentication.
	ModeDaprSentry AuthMode = "dapr-sentry"
)

// Common errors
var (
	ErrNoToken           = errors.New("no authentication token provided")
	ErrInvalidToken      = errors.New("invalid authentication token")
	ErrTokenExpired      = errors.New("authentication token expired")
	ErrInvalidIssuer     = errors.New("invalid token issuer")
	ErrInvalidAudience   = errors.New("invalid token audience")
	ErrAuthDisabled      = errors.New("authentication is disabled")
	ErrUnsupportedMethod = errors.New("unsupported authentication method")
)

// Identity represents an authenticated identity.
type Identity struct {
	// Subject is the unique identifier for the identity (e.g., user ID or SPIFFE ID).
	Subject string
	// Issuer is the identity provider that issued the token.
	Issuer string
	// Audience is the intended audience of the token.
	Audience []string
	// Email is the email address (for OIDC identities).
	Email string
	// Name is the display name (for OIDC identities).
	Name string
	// Claims contains all claims from the token.
	Claims map[string]interface{}
	// AuthMethod indicates how the identity was authenticated.
	AuthMethod AuthMode
}

// Authenticator is the interface for authentication providers.
type Authenticator interface {
	// Authenticate validates a token and returns the identity.
	Authenticate(ctx context.Context, token string) (*Identity, error)
	// Mode returns the authentication mode.
	Mode() AuthMode
}

// contextKey is a type for context keys.
type contextKey string

const identityKey contextKey = "auth.identity"

// WithIdentity adds an identity to the context.
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// GetIdentity retrieves the identity from the context.
func GetIdentity(ctx context.Context) *Identity {
	id, _ := ctx.Value(identityKey).(*Identity)
	return id
}

// IsAuthenticated returns true if the context has an authenticated identity.
func IsAuthenticated(ctx context.Context) bool {
	return GetIdentity(ctx) != nil
}
