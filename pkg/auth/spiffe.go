// Package auth provides authentication and authorization for the MCP server.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// SPIFFEConnectionTimeout is the maximum time to wait for SPIRE agent connection.
const SPIFFEConnectionTimeout = 10 * time.Second

// SPIFFEAuthenticator authenticates requests using SPIFFE JWT-SVIDs.
type SPIFFEAuthenticator struct {
	config   SPIFFEConfig
	source   *workloadapi.JWTSource
	serverID spiffeid.ID
	allowed  map[string]struct{}
}

// NewSPIFFEAuthenticator creates a new SPIFFE authenticator.
func NewSPIFFEAuthenticator(ctx context.Context, cfg SPIFFEConfig) (*SPIFFEAuthenticator, error) {
	// Parse the server ID
	serverID, err := spiffeid.FromString(cfg.ServerID)
	if err != nil {
		return nil, fmt.Errorf("invalid server SPIFFE ID: %w", err)
	}

	// Create JWT source options
	opts := []workloadapi.JWTSourceOption{}
	if cfg.EndpointSocket != "" {
		opts = append(opts, workloadapi.WithClientOptions(
			workloadapi.WithAddr(cfg.EndpointSocket),
		))
	}

	// Create a timeout context for connecting to the SPIRE agent
	connectCtx, cancel := context.WithTimeout(ctx, SPIFFEConnectionTimeout)
	defer cancel()

	// Create JWT source for validating JWTs
	source, err := workloadapi.NewJWTSource(connectCtx, opts...)
	if err != nil {
		if connectCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("timeout connecting to SPIRE agent at %s (waited %v): %w", cfg.EndpointSocket, SPIFFEConnectionTimeout, err)
		}
		return nil, fmt.Errorf("failed to create JWT source: %w", err)
	}

	// Build allowed clients map
	allowed := make(map[string]struct{})
	for _, client := range cfg.AllowedClients {
		allowed[client] = struct{}{}
	}

	return &SPIFFEAuthenticator{
		config:   cfg,
		source:   source,
		serverID: serverID,
		allowed:  allowed,
	}, nil
}

// Authenticate validates a SPIFFE JWT-SVID and returns the identity.
func (a *SPIFFEAuthenticator) Authenticate(ctx context.Context, token string) (*Identity, error) {
	// Parse and validate the JWT-SVID
	svid, err := jwtsvid.ParseAndValidate(token, a.source, []string{a.serverID.String()})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Check trust domain
	if svid.ID.TrustDomain().Name() != a.config.TrustDomain {
		return nil, fmt.Errorf("%w: trust domain mismatch", ErrInvalidIssuer)
	}

	// Check if client is allowed (if allowlist is configured)
	if len(a.allowed) > 0 {
		if _, ok := a.allowed[svid.ID.String()]; !ok {
			return nil, fmt.Errorf("%w: client not in allowlist", ErrInvalidToken)
		}
	}

	return &Identity{
		Subject:    svid.ID.String(),
		Issuer:     svid.ID.TrustDomain().String(),
		Audience:   svid.Audience,
		Claims:     svid.Claims,
		AuthMethod: ModeSPIFFE,
	}, nil
}

// Mode returns the authentication mode.
func (a *SPIFFEAuthenticator) Mode() AuthMode {
	return ModeSPIFFE
}

// Close closes the SPIFFE authenticator and releases resources.
func (a *SPIFFEAuthenticator) Close() error {
	return a.source.Close()
}
