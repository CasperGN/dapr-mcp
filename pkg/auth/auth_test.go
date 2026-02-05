package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthModeConstants(t *testing.T) {
	assert.Equal(t, AuthMode("disabled"), ModeDisabled)
	assert.Equal(t, AuthMode("oidc"), ModeOIDC)
	assert.Equal(t, AuthMode("spiffe"), ModeSPIFFE)
	assert.Equal(t, AuthMode("hybrid"), ModeHybrid)
	assert.Equal(t, AuthMode("dapr-sentry"), ModeDaprSentry)
}

func TestErrorConstants(t *testing.T) {
	assert.NotNil(t, ErrNoToken)
	assert.NotNil(t, ErrInvalidToken)
	assert.NotNil(t, ErrTokenExpired)
	assert.NotNil(t, ErrInvalidIssuer)
	assert.NotNil(t, ErrInvalidAudience)
	assert.NotNil(t, ErrAuthDisabled)
	assert.NotNil(t, ErrUnsupportedMethod)

	assert.Equal(t, "no authentication token provided", ErrNoToken.Error())
	assert.Equal(t, "invalid authentication token", ErrInvalidToken.Error())
	assert.Equal(t, "authentication token expired", ErrTokenExpired.Error())
	assert.Equal(t, "invalid token issuer", ErrInvalidIssuer.Error())
	assert.Equal(t, "invalid token audience", ErrInvalidAudience.Error())
	assert.Equal(t, "authentication is disabled", ErrAuthDisabled.Error())
	assert.Equal(t, "unsupported authentication method", ErrUnsupportedMethod.Error())
}

func TestIdentityStruct(t *testing.T) {
	identity := &Identity{
		Subject:    "user-123",
		Issuer:     "https://issuer.example.com",
		Audience:   []string{"api.example.com"},
		Email:      "user@example.com",
		Name:       "Test User",
		Claims:     map[string]interface{}{"role": "admin"},
		AuthMethod: ModeOIDC,
	}

	assert.Equal(t, "user-123", identity.Subject)
	assert.Equal(t, "https://issuer.example.com", identity.Issuer)
	assert.Equal(t, []string{"api.example.com"}, identity.Audience)
	assert.Equal(t, "user@example.com", identity.Email)
	assert.Equal(t, "Test User", identity.Name)
	assert.Equal(t, "admin", identity.Claims["role"])
	assert.Equal(t, ModeOIDC, identity.AuthMethod)
}

func TestWithIdentity(t *testing.T) {
	ctx := context.Background()
	identity := &Identity{
		Subject:    "test-subject",
		AuthMethod: ModeOIDC,
	}

	newCtx := WithIdentity(ctx, identity)

	assert.NotNil(t, newCtx)
	assert.NotEqual(t, ctx, newCtx)
}

func TestGetIdentity(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected *Identity
	}{
		{
			name: "identity exists in context",
			setup: func() context.Context {
				identity := &Identity{Subject: "test-user"}
				return WithIdentity(context.Background(), identity)
			},
			expected: &Identity{Subject: "test-user"},
		},
		{
			name: "identity does not exist in context",
			setup: func() context.Context {
				return context.Background()
			},
			expected: nil,
		},
		{
			name: "wrong type in context",
			setup: func() context.Context {
				return context.WithValue(context.Background(), identityKey, "not-an-identity")
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetIdentity(ctx)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Subject, result.Subject)
			}
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected bool
	}{
		{
			name: "authenticated - identity exists",
			setup: func() context.Context {
				identity := &Identity{Subject: "test-user"}
				return WithIdentity(context.Background(), identity)
			},
			expected: true,
		},
		{
			name: "not authenticated - no identity",
			setup: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
		{
			name: "not authenticated - nil identity",
			setup: func() context.Context {
				return WithIdentity(context.Background(), nil)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := IsAuthenticated(ctx)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIdentityWithSPIFFE(t *testing.T) {
	identity := &Identity{
		Subject:    "spiffe://cluster.local/ns/default/sa/myapp",
		AuthMethod: ModeSPIFFE,
	}

	assert.Equal(t, ModeSPIFFE, identity.AuthMethod)
	assert.Contains(t, identity.Subject, "spiffe://")
}

func TestIdentityWithDaprSentry(t *testing.T) {
	identity := &Identity{
		Subject:    "spiffe://public/ns/default/app-id",
		AuthMethod: ModeDaprSentry,
	}

	assert.Equal(t, ModeDaprSentry, identity.AuthMethod)
	assert.Contains(t, identity.Subject, "spiffe://public")
}

func TestContextKeyType(t *testing.T) {
	// Verify that context key is properly typed
	assert.Equal(t, contextKey("auth.identity"), identityKey)
}

// testContextKey is a custom type for test context keys to avoid collisions.
type testContextKey string

func TestGetIdentityFromDifferentContextValues(t *testing.T) {
	// Test that other context values don't interfere
	ctx := context.Background()
	ctx = context.WithValue(ctx, testContextKey("other-key"), "other-value")

	identity := &Identity{Subject: "test"}
	ctx = WithIdentity(ctx, identity)

	// Both values should coexist
	result := GetIdentity(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, "test", result.Subject)

	otherValue := ctx.Value(testContextKey("other-key"))
	assert.Equal(t, "other-value", otherValue)
}

func TestIdentityEmptyFields(t *testing.T) {
	identity := &Identity{}

	assert.Empty(t, identity.Subject)
	assert.Empty(t, identity.Issuer)
	assert.Empty(t, identity.Audience)
	assert.Empty(t, identity.Email)
	assert.Empty(t, identity.Name)
	assert.Nil(t, identity.Claims)
	assert.Empty(t, identity.AuthMethod)
}

func TestAuthModeValues(t *testing.T) {
	modes := []AuthMode{
		ModeDisabled,
		ModeOIDC,
		ModeSPIFFE,
		ModeHybrid,
		ModeDaprSentry,
	}

	// Ensure all modes are unique
	seen := make(map[AuthMode]bool)
	for _, mode := range modes {
		assert.False(t, seen[mode], "duplicate mode: %s", mode)
		seen[mode] = true
	}
}
