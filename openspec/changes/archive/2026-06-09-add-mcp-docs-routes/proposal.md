## Why

Finch's remote MCP server currently routes root HTTP traffic into the MCP transport, which makes browser visits and documentation discovery confusing and causes non-MCP documentation paths to behave like MCP requests. Finch should clearly separate the machine protocol endpoint from lightweight public documentation so agents and humans can discover how to connect without initializing an MCP session.

## What Changes

- **BREAKING**: Move the canonical Streamable HTTP MCP endpoint from `/` to `/mcp`.
- Make `/` a lightweight public landing page that explains Finch is an MCP server, not a browser app.
- Add public `/llms.txt` documentation with concise AI-readable connection notes.
- Add optional public `/.well-known/mcp.json` metadata for machine-readable discovery.
- Keep MCP bearer authentication and tool behavior unchanged for requests sent to `/mcp`.
- Ensure docs routes never expose real API keys and do not require MCP session initialization.

## Capabilities

### New Capabilities
- `mcp-docs-discovery`: Public, lightweight documentation and metadata routes for Finch MCP discovery.

### Modified Capabilities
- `remote-mcp-server-startup`: The HTTP MCP transport is exposed at `/mcp`, while `/` becomes a docs landing route.
- `mcp-bearer-authentication`: Bearer authentication remains required for remote HTTP MCP requests, but public docs routes are explicitly outside MCP auth.

## Impact

- Affects `internal/mcp/transport.go` HTTP routing and any tests that assume MCP traffic is served from `/`.
- Requires route-level tests proving docs routes are public and `/mcp` remains authenticated.
- Existing MCP clients configured with the root URL must update their endpoint to `/mcp`.
- No new runtime dependencies or full documentation system are required.
