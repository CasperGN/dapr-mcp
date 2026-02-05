// Package auth provides authentication and authorization for the MCP server.
package auth

import (
	"errors"
	"os"
	"strings"
	"time"
)

// Config holds the authentication configuration.
type Config struct {
	// Enabled determines if authentication is required.
	Enabled bool
	// Mode is the authentication mode.
	Mode AuthMode
	// SkipPaths are paths that don't require authentication.
	SkipPaths []string

	// OIDC configuration
	OIDC OIDCConfig

	// SPIFFE configuration
	SPIFFE SPIFFEConfig

	// DaprSentry configuration
	DaprSentry DaprSentryConfig
}

// OIDCConfig holds OIDC-specific configuration.
type OIDCConfig struct {
	// Enabled determines if OIDC authentication is enabled.
	Enabled bool
	// IssuerURL is the OIDC provider URL.
	IssuerURL string
	// ClientID is the expected audience (aud claim).
	ClientID string
	// AllowedAlgorithms are the allowed signing algorithms.
	AllowedAlgorithms []string
	// SkipIssuerCheck skips issuer validation (dev only).
	SkipIssuerCheck bool
}

// SPIFFEConfig holds SPIFFE-specific configuration.
type SPIFFEConfig struct {
	// Enabled determines if SPIFFE authentication is enabled.
	Enabled bool
	// TrustDomain is the SPIFFE trust domain.
	TrustDomain string
	// ServerID is this server's SPIFFE ID.
	ServerID string
	// EndpointSocket is the Workload API socket path.
	EndpointSocket string
	// AllowedClients are the allowed client SPIFFE IDs.
	AllowedClients []string
}

// DaprSentryConfig holds Dapr Sentry JWT-specific configuration.
type DaprSentryConfig struct {
	// Enabled determines if Dapr Sentry authentication is enabled.
	Enabled bool
	// JWKSUrl is the URL to fetch JWKS from Dapr Sentry.
	JWKSUrl string
	// TrustDomain is the expected trust domain in SPIFFE IDs.
	TrustDomain string
	// Audience is the expected audience claim (optional).
	Audience string
	// TokenHeader is the custom header to extract the token from (default: Authorization).
	TokenHeader string
	// RefreshInterval is the interval for refreshing the JWKS cache.
	RefreshInterval time.Duration
}

// DefaultConfig returns configuration from environment variables.
func DefaultConfig() Config {
	enabled := os.Getenv("AUTH_ENABLED") == "true"
	mode := AuthMode(strings.ToLower(os.Getenv("AUTH_MODE")))
	if mode == "" {
		mode = ModeDisabled
	}

	skipPathsStr := os.Getenv("AUTH_SKIP_PATHS")
	skipPaths := []string{"/livez", "/readyz", "/startupz"}
	if skipPathsStr != "" {
		skipPaths = strings.Split(skipPathsStr, ",")
		for i := range skipPaths {
			skipPaths[i] = strings.TrimSpace(skipPaths[i])
		}
	}

	// Get base configs
	oidcConfig := defaultOIDCConfig()
	spiffeConfig := defaultSPIFFEConfig()
	daprSentryConfig := defaultDaprSentryConfig()

	// Auto-enable the appropriate config based on AUTH_MODE.
	// This removes the need for redundant *_ENABLED flags when using single-mode auth.
	// For ModeHybrid, the individual *_ENABLED flags are still read from env vars.
	switch mode {
	case ModeOIDC:
		oidcConfig.Enabled = true
	case ModeSPIFFE:
		spiffeConfig.Enabled = true
	case ModeDaprSentry:
		daprSentryConfig.Enabled = true
	case ModeHybrid:
		// For hybrid mode, keep the individual *_ENABLED flags from env vars
		// (already set in the default*Config() functions)
	}

	return Config{
		Enabled:    enabled,
		Mode:       mode,
		SkipPaths:  skipPaths,
		OIDC:       oidcConfig,
		SPIFFE:     spiffeConfig,
		DaprSentry: daprSentryConfig,
	}
}

func defaultOIDCConfig() OIDCConfig {
	algStr := os.Getenv("OIDC_ALLOWED_ALGORITHMS")
	algs := []string{"RS256", "ES256"}
	if algStr != "" {
		algs = strings.Split(algStr, ",")
		for i := range algs {
			algs[i] = strings.TrimSpace(algs[i])
		}
	}

	return OIDCConfig{
		Enabled:           os.Getenv("OIDC_ENABLED") == "true",
		IssuerURL:         os.Getenv("OIDC_ISSUER_URL"),
		ClientID:          os.Getenv("OIDC_CLIENT_ID"),
		AllowedAlgorithms: algs,
		SkipIssuerCheck:   os.Getenv("OIDC_SKIP_ISSUER_CHECK") == "true",
	}
}

func defaultSPIFFEConfig() SPIFFEConfig {
	allowedStr := os.Getenv("SPIFFE_ALLOWED_CLIENTS")
	var allowed []string
	if allowedStr != "" {
		allowed = strings.Split(allowedStr, ",")
		for i := range allowed {
			allowed[i] = strings.TrimSpace(allowed[i])
		}
	}

	return SPIFFEConfig{
		Enabled:        os.Getenv("SPIFFE_ENABLED") == "true",
		TrustDomain:    os.Getenv("SPIFFE_TRUST_DOMAIN"),
		ServerID:       os.Getenv("SPIFFE_SERVER_ID"),
		EndpointSocket: os.Getenv("SPIFFE_ENDPOINT_SOCKET"),
		AllowedClients: allowed,
	}
}

func defaultDaprSentryConfig() DaprSentryConfig {
	refreshInterval := 5 * time.Minute
	if intervalStr := os.Getenv("DAPR_SENTRY_JWKS_REFRESH_INTERVAL"); intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr); err == nil {
			refreshInterval = parsed
		}
	}

	tokenHeader := os.Getenv("DAPR_SENTRY_TOKEN_HEADER")
	if tokenHeader == "" {
		tokenHeader = "Authorization"
	}

	return DaprSentryConfig{
		Enabled:         os.Getenv("DAPR_SENTRY_ENABLED") == "true",
		JWKSUrl:         os.Getenv("DAPR_SENTRY_JWKS_URL"),
		TrustDomain:     os.Getenv("DAPR_SENTRY_TRUST_DOMAIN"),
		Audience:        os.Getenv("DAPR_SENTRY_AUDIENCE"),
		TokenHeader:     tokenHeader,
		RefreshInterval: refreshInterval,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	switch c.Mode {
	case ModeOIDC:
		if !c.OIDC.Enabled {
			return ErrUnsupportedMethod
		}
		return c.OIDC.Validate()
	case ModeSPIFFE:
		if !c.SPIFFE.Enabled {
			return ErrUnsupportedMethod
		}
		return c.SPIFFE.Validate()
	case ModeDaprSentry:
		if !c.DaprSentry.Enabled {
			return ErrUnsupportedMethod
		}
		return c.DaprSentry.Validate()
	case ModeHybrid:
		// At least one method must be enabled
		if !c.OIDC.Enabled && !c.SPIFFE.Enabled && !c.DaprSentry.Enabled {
			return ErrUnsupportedMethod
		}
		if c.OIDC.Enabled {
			if err := c.OIDC.Validate(); err != nil {
				return err
			}
		}
		if c.SPIFFE.Enabled {
			if err := c.SPIFFE.Validate(); err != nil {
				return err
			}
		}
		if c.DaprSentry.Enabled {
			if err := c.DaprSentry.Validate(); err != nil {
				return err
			}
		}
		return nil
	case ModeDisabled:
		return nil
	default:
		return ErrUnsupportedMethod
	}
}

// Validate validates the OIDC configuration.
func (c *OIDCConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.IssuerURL == "" {
		return errors.New("OIDC_ISSUER_URL is required")
	}
	if c.ClientID == "" {
		return errors.New("OIDC_CLIENT_ID is required")
	}
	return nil
}

// Validate validates the SPIFFE configuration.
func (c *SPIFFEConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.TrustDomain == "" {
		return errors.New("SPIFFE_TRUST_DOMAIN is required")
	}
	if c.ServerID == "" {
		return errors.New("SPIFFE_SERVER_ID is required")
	}
	return nil
}

// Validate validates the Dapr Sentry configuration.
func (c *DaprSentryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.JWKSUrl == "" {
		return errors.New("DAPR_SENTRY_JWKS_URL is required")
	}
	if c.TrustDomain == "" {
		return errors.New("DAPR_SENTRY_TRUST_DOMAIN is required")
	}
	return nil
}
