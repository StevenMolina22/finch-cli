## 1. HTTP Routing

- [x] 1.1 Add a shared MCP HTTP handler constructor that builds the production mux without opening a listener.
- [x] 1.2 Mount the Streamable HTTP MCP handler at `/mcp` behind the existing bearer auth middleware.
- [x] 1.3 Stop routing `/` through the MCP transport handler.
- [x] 1.4 Decide and implement whether `/mcp/` is accepted or rejected consistently with tests.

## 2. Documentation Routes

- [x] 2.1 Add a public `/` landing route explaining Finch is an MCP server, not a browser app.
- [x] 2.2 Add a public `/llms.txt` route with base URL guidance, `/mcp` endpoint, bearer auth, tool names, and usage notes.
- [x] 2.3 Add a public `/.well-known/mcp.json` route with machine-readable metadata.
- [x] 2.4 Add a single helper for Finch MCP tool names and use it in documentation responses.
- [x] 2.5 Ensure documentation examples use placeholders and never include configured API key values.

## 3. Tests

- [x] 3.1 Add route tests proving `GET /`, `GET /llms.txt`, and `GET /.well-known/mcp.json` succeed without `Authorization`.
- [x] 3.2 Add route tests proving unauthenticated MCP requests to `/mcp` are rejected.
- [x] 3.3 Update Streamable HTTP tests to send MCP requests to `/mcp` instead of `/`.
- [x] 3.4 Add a test proving `/` no longer initializes or handles MCP protocol requests.
- [x] 3.5 Add response content tests for required docs fields and tool names.

## 4. Documentation And Verification

- [x] 4.1 Update package or command help documentation that mentions the remote MCP endpoint.
- [x] 4.2 Run OpenSpec validation for `add-mcp-docs-routes`.
- [x] 4.3 Run the Go test suite.
