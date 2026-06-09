## Context

The remote MCP server is implemented in `internal/mcp/transport.go` using the official Go MCP SDK's Streamable HTTP handler. Today `RunHTTP` mounts that handler at `/` and wraps the root route with bearer authentication, so any request to the server is treated as MCP traffic. This makes the root URL confusing for humans and agents, and prevents simple public documentation routes such as `/llms.txt` from working independently of MCP initialization.

Finch remains an agent-only MCP server. The change should make discovery clearer without introducing a separate docs system, changing tool behavior, or weakening authentication for actual MCP requests.

## Goals / Non-Goals

**Goals:**
- Make `/mcp` the canonical Streamable HTTP MCP endpoint.
- Make `/`, `/llms.txt`, and `/.well-known/mcp.json` public documentation/discovery routes.
- Keep bearer authentication required for MCP traffic and unchanged in behavior.
- Keep route content static, small, and safe to expose publicly.
- Avoid duplicating tool names in a way that can drift from MCP registration.

**Non-Goals:**
- Build a full documentation site or template system.
- Add browser application behavior beyond a small landing page.
- Add new authentication modes or per-tool permission changes.
- Preserve `/` as an MCP endpoint alias.
- Infer or configure a public deployment URL in this change.

## Decisions

1. Mount MCP only at `/mcp`.

   The production HTTP mux should route `/mcp` to the SDK Streamable HTTP handler wrapped in `bearerAuthMiddleware`. This cleanly separates protocol traffic from docs traffic. Keeping `/` as an MCP alias was considered, but it would preserve the original ambiguity and make root documentation impossible without method-specific special cases.

2. Add a shared HTTP handler constructor.

   Introduce a small helper such as `NewHTTPHandler(store Store, auth AuthConfig) http.Handler` that builds the mux with docs routes and the authenticated MCP route. `RunHTTP` can then handle address binding and shutdown only, while tests can exercise routing without opening a socket.

3. Keep documentation content generated in Go, not external assets.

   The docs are intentionally short and static. In-code handlers avoid adding file embedding, asset directories, or a docs build step. The landing page may return simple HTML, while `/llms.txt` returns `text/plain` and `/.well-known/mcp.json` returns JSON.

4. Expose tool names from one source.

   Add a helper such as `ToolNames() []string` backed by the existing tool constants in `server.go`. Docs handlers should use this helper so the public route content stays aligned with registered tools.

5. Use relative endpoints in machine-readable metadata.

   `/.well-known/mcp.json` should report `endpoint: "/mcp"` and docs URLs as relative paths. The server only knows its bind address, which is often not the public HTTPS origin behind a reverse proxy, so absolute URLs would frequently be wrong.

## Risks / Trade-offs

- Existing clients configured against `/` will fail until updated to `/mcp` -> Mark the endpoint move as breaking in the proposal and include the new endpoint in docs.
- Some MCP clients may send requests to `/mcp/` with a trailing slash -> Either support both `/mcp` and `/mcp/`, or add a test and decide intentionally during implementation.
- Tool names can drift if hard-coded in multiple docs handlers -> Use a single `ToolNames()` helper.
- Public docs could accidentally imply an actual API key -> Use placeholder text only, such as `<API_KEY>`.
- Public docs could be accidentally wrapped by auth middleware -> Add route tests proving docs paths return without `Authorization`.

## Migration Plan

Update deployed MCP client configs from the server root URL to `/mcp`. If a deployment must roll back, returning the MCP handler to `/` restores the old endpoint shape, but removes the public docs landing page.

## Open Questions

- Should `/mcp/` be accepted as an alias for `/mcp`, or should clients be required to use the exact canonical endpoint?
