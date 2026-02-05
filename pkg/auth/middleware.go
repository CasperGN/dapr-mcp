// Package auth provides authentication and authorization for the MCP server.
package auth

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// Middleware provides HTTP authentication middleware.
type Middleware struct {
	authenticators []Authenticator
	config         Config
	logger         *slog.Logger
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(cfg Config, authenticators []Authenticator, logger *slog.Logger) *Middleware {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Middleware{
		authenticators: authenticators,
		config:         cfg,
		logger:         logger,
	}
}

// Handler returns an HTTP middleware that authenticates requests.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.logger.Debug("[AUTH] incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Log all headers for debugging
		for name, values := range r.Header {
			// Don't log the actual token value, just that the header exists
			if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, m.config.DaprSentry.TokenHeader) {
				m.logger.Debug("[AUTH] request header (token)",
					"header", name,
					"value_length", len(strings.Join(values, "")),
					"present", len(values) > 0,
				)
			} else {
				m.logger.Debug("[AUTH] request header",
					"header", name,
					"values", values,
				)
			}
		}

		// Check if path should skip authentication
		if m.shouldSkip(r.URL.Path) {
			m.logger.Debug("[AUTH] skipping authentication for path",
				"path", r.URL.Path,
				"skip_paths", m.config.SkipPaths,
			)
			next.ServeHTTP(w, r)
			return
		}

		// If authentication is disabled, proceed without authentication
		if !m.config.Enabled || m.config.Mode == ModeDisabled {
			m.logger.Debug("[AUTH] authentication disabled, passing through",
				"enabled", m.config.Enabled,
				"mode", m.config.Mode,
			)
			next.ServeHTTP(w, r)
			return
		}

		m.logger.Debug("[AUTH] authentication required",
			"mode", m.config.Mode,
			"authenticator_count", len(m.authenticators),
			"custom_token_header", m.config.DaprSentry.TokenHeader,
		)

		// Extract token from headers
		token, headerUsed := m.extractTokenWithSource(r)
		if token == "" {
			m.logger.Debug("[AUTH] no authentication token found",
				"path", r.URL.Path,
				"checked_custom_header", m.config.DaprSentry.TokenHeader,
				"checked_authorization", true,
			)
			http.Error(w, "Unauthorized: no token provided", http.StatusUnauthorized)
			return
		}

		m.logger.Debug("[AUTH] token extracted",
			"header_used", headerUsed,
			"token_length", len(token),
			"token_prefix", safeTokenPrefix(token),
		)

		// Try each authenticator
		var identity *Identity
		var lastErr error

		for i, auth := range m.authenticators {
			m.logger.Debug("[AUTH] trying authenticator",
				"index", i,
				"mode", auth.Mode(),
			)

			id, err := auth.Authenticate(r.Context(), token)
			if err == nil {
				identity = id
				m.logger.Debug("[AUTH] authenticator succeeded",
					"index", i,
					"mode", auth.Mode(),
					"subject", id.Subject,
				)
				break
			}

			m.logger.Debug("[AUTH] authenticator failed",
				"index", i,
				"mode", auth.Mode(),
				"error", err.Error(),
			)
			lastErr = err
		}

		if identity == nil {
			m.logger.Debug("[AUTH] all authenticators failed",
				"path", r.URL.Path,
				"last_error", lastErr,
				"authenticator_count", len(m.authenticators),
			)
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Log successful authentication
		m.logger.Debug("[AUTH] authentication successful",
			"path", r.URL.Path,
			"subject", identity.Subject,
			"method", identity.AuthMethod,
			"audience", identity.Audience,
		)

		// Add identity to context and continue
		ctx := WithIdentity(r.Context(), identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// shouldSkip returns true if the path should skip authentication.
func (m *Middleware) shouldSkip(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
		// Support wildcard suffix
		if strings.HasSuffix(skipPath, "*") {
			prefix := strings.TrimSuffix(skipPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// extractTokenWithSource extracts the token and returns which header it came from.
func (m *Middleware) extractTokenWithSource(r *http.Request) (token string, header string) {
	// Check custom Dapr Sentry header first if configured and different from Authorization
	if m.config.DaprSentry.TokenHeader != "" && m.config.DaprSentry.TokenHeader != "Authorization" {
		if token := r.Header.Get(m.config.DaprSentry.TokenHeader); token != "" {
			// Custom headers typically contain raw tokens without Bearer prefix
			return token, m.config.DaprSentry.TokenHeader
		}
	}

	// Fall back to Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ""
	}

	// Support "Bearer <token>" format
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimPrefix(auth[7:], " "), "Authorization (Bearer)"
	}

	// Also support raw token
	return auth, "Authorization (raw)"
}

// safeTokenPrefix returns the first few characters of a token for debugging
// without exposing the full token.
func safeTokenPrefix(token string) string {
	if len(token) <= 20 {
		return "[token too short to preview safely]"
	}
	return token[:20] + "..."
}

// NoopMiddleware returns a middleware that does nothing (for disabled auth).
func NoopMiddleware(next http.Handler) http.Handler {
	return next
}
