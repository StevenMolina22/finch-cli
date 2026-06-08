package mcp

import "context"

// Permission represents the access level granted to an authenticated MCP
// client. A write permission includes read permission.
type Permission int

const (
	// PermissionAnonymous denies all tool access. It is the default for
	// requests that have not been authenticated.
	PermissionAnonymous Permission = iota
	// PermissionRead allows calling read-only MCP tools.
	PermissionRead
	// PermissionWrite allows calling both read-only and write MCP tools.
	PermissionWrite
)

func (p Permission) allowsRead() bool {
	return p == PermissionRead || p == PermissionWrite
}

func (p Permission) allowsWrite() bool {
	return p == PermissionWrite
}

type permissionKey struct{}

// WithPermission returns a context carrying the supplied permission.
func WithPermission(ctx context.Context, p Permission) context.Context {
	return context.WithValue(ctx, permissionKey{}, p)
}

// PermissionFromContext returns the permission stored in ctx, or
// PermissionAnonymous if none was set.
func PermissionFromContext(ctx context.Context) Permission {
	if ctx == nil {
		return PermissionAnonymous
	}
	v, ok := ctx.Value(permissionKey{}).(Permission)
	if !ok {
		return PermissionAnonymous
	}
	return v
}
