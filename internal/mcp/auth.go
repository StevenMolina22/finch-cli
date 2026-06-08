package mcp

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
)

// AuthConfig holds the bearer token values used to authenticate HTTP MCP
// requests. Either or both may be empty; startup validation in RunHTTP
// ensures at least one is present when serving HTTP transport.
type AuthConfig struct {
	ReadToken  string
	WriteToken string
}

// IsEmpty reports whether neither token is configured.
func (c AuthConfig) IsEmpty() bool {
	return strings.TrimSpace(c.ReadToken) == "" && strings.TrimSpace(c.WriteToken) == ""
}

// authFailureStatus is the status returned to clients when bearer token
// validation fails. The body never contains the supplied or configured
// token values.
const authFailureStatus = "401 Unauthorized"

// bearerPrefix is the case-insensitive prefix used to extract a bearer
// token from the Authorization header.
const bearerPrefix = "bearer "

// classifyBearerToken compares a supplied bearer token against the
// configured read and write tokens using constant-time comparison. It
// returns the permission granted by the supplied token, or
// PermissionAnonymous if the token is missing, malformed, or does not
// match a configured token.
func classifyBearerToken(supplied string, cfg AuthConfig) Permission {
	trimmed := strings.TrimSpace(supplied)
	if trimmed == "" {
		return PermissionAnonymous
	}
	read := strings.TrimSpace(cfg.ReadToken)
	if read != "" && subtle.ConstantTimeCompare([]byte(trimmed), []byte(read)) == 1 {
		return PermissionRead
	}
	write := strings.TrimSpace(cfg.WriteToken)
	if write != "" && subtle.ConstantTimeCompare([]byte(trimmed), []byte(write)) == 1 {
		return PermissionWrite
	}
	return PermissionAnonymous
}

// extractBearerToken parses the Authorization header and returns the
// supplied bearer token, or an empty string if the header is missing,
// malformed, or not a bearer token. The returned bool indicates whether
// the header was present and well-formed.
func extractBearerToken(header string) (string, bool) {
	if header == "" {
		return "", false
	}
	fields := strings.Fields(header)
	if len(fields) != 2 {
		return "", false
	}
	if !strings.EqualFold(fields[0], "bearer") {
		return "", false
	}
	return fields[1], true
}

// writeAuthFailure writes a generic authorization failure response to w
// without revealing the supplied or configured token values. The status
// code is 401 for missing/invalid bearer tokens.
func writeAuthFailure(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("WWW-Authenticate", `Bearer realm="finch-mcp"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprintln(w, authFailureStatus)
}
