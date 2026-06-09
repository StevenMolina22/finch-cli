# Finch MCP server

The `internal/mcp` package implements Finch's remote MCP (Model Context
Protocol) server using the official Go SDK
(`github.com/modelcontextprotocol/go-sdk/mcp`).

## SDK choice

The first implementation uses the official Go MCP SDK pinned at
`github.com/modelcontextprotocol/go-sdk/mcp@v0.8.0`. The SDK exposes
`mcp.NewStreamableHTTPHandler`, which satisfies the Streamable HTTP
transport required for remote MCP clients. No alternate SDK
(`github.com/mark3labs/mcp-go` or otherwise) is required at this time.

If a future upgrade reveals that the official SDK cannot satisfy HTTP
transport or required auth behavior, document the blocker in
`design.md` and update the dependency in `go.mod` before falling back.

## Package layout

| File                | Purpose                                                                 |
|---------------------|-------------------------------------------------------------------------|
| `permissions.go`    | Permission types and context propagation for authenticated MCP clients. |
| `store.go`          | Narrow `Store` interface used by tool handlers; adapter for `*finch.Store`. |
| `auth.go`           | Bearer token extraction and constant-time classification.               |
| `server.go`         | `mcp.Server` construction and tool registration.                        |
| `handlers.go`       | Per-tool handlers that delegate to `internal/finch` validation and storage. |
| `transport.go`      | `Run` entry point and HTTP transport startup with auth middleware.      |

## HTTP routes

- `/mcp` is the Streamable HTTP MCP endpoint and requires `Authorization: Bearer <API_KEY>`.
- `/` is a public landing page explaining how agents should connect.
- `/llms.txt` is public AI-readable usage documentation.
- `/.well-known/mcp.json` is public machine-readable MCP metadata.

## Security notes

- Bearer tokens are compared using `crypto/subtle.ConstantTimeCompare` and
  are never logged or included in error messages.
- HTTP startup fails fast when no API key is configured.
- Tool handlers re-check the permission from context for each call, so
  tool-level enforcement is independent of the HTTP middleware.
