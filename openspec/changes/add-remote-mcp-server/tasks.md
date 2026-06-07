## 1. SDK and Package Setup

- [ ] 1.1 Verify the official MCP Go SDK HTTP server APIs against the pinned module version and add `github.com/modelcontextprotocol/go-sdk/mcp` to `go.mod`.
- [ ] 1.2 Create `internal/mcp` with server options, a narrow store interface, permission types, and transport startup entry points.
- [ ] 1.3 Document any fallback decision if the official SDK cannot satisfy HTTP transport and an alternate MCP SDK is selected.

## 2. CLI Startup

- [ ] 2.1 Add `finch mcp` to the Cobra root command without changing existing commands.
- [ ] 2.2 Add `--transport` with `http` support and optional `stdio` support if practical.
- [ ] 2.3 Add `--addr` defaulting to `:3333` for HTTP transport.
- [ ] 2.4 Ensure unsupported transports fail before server startup with a clear error.
- [ ] 2.5 Ensure HTTP transport fails fast before binding when neither MCP auth token is configured.

## 3. HTTP Authentication and Authorization

- [ ] 3.1 Load `FINCH_MCP_READ_TOKEN` and `FINCH_MCP_WRITE_TOKEN` for HTTP MCP startup without logging token values.
- [ ] 3.2 Implement bearer-token middleware that rejects missing, malformed, and invalid tokens before MCP tool handling.
- [ ] 3.3 Compare supplied and configured tokens using constant-time comparison.
- [ ] 3.4 Propagate authenticated read or write permission into tool-call context.
- [ ] 3.5 Enforce read token access only for read tools and write token access for both read and write tools.

## 4. MCP Tool Handlers

- [ ] 4.1 Register `finch_add_transaction` with input parsing, existing Finch validation, current UTC date defaulting, storage creation, and structured success output.
- [ ] 4.2 Register `finch_list_transactions` with optional `month`, `category`, and `limit` validation and existing transaction JSON output.
- [ ] 4.3 Register `finch_get_summary` with optional `month` validation and existing summary JSON output.
- [ ] 4.4 Register `finch_edit_transaction` with positive id validation, editable field validation, `confirm: true` enforcement, update handling, and structured success or not-found output.
- [ ] 4.5 Register `finch_delete_transaction` with positive id validation, `confirm: true` enforcement, delete handling, and structured success or not-found output.
- [ ] 4.6 Keep MCP handlers calling `internal/finch` validation and storage logic directly, with no CLI shell-out and no Fiber API calls.

## 5. Documentation and Safety

- [ ] 5.1 Add command help or project documentation showing `finch mcp --transport http --addr :3333`.
- [ ] 5.2 Document that remote HTTP MCP requires HTTPS in deployment, including reverse proxy or platform TLS termination.
- [ ] 5.3 Document that a write token can call read tools and write tools, including the behavior when only a write token is configured.
- [ ] 5.4 Exclude import/export MCP tools from the first implementation unless CSV import/export is first corrected to use Go's `encoding/csv`.

## 6. Tests and Verification

- [ ] 6.1 Add MCP handler tests with fake storage or dependency injection for add, list, summary, edit, and delete.
- [ ] 6.2 Test read token can call read tools.
- [ ] 6.3 Test read token cannot call write tools and no mutation occurs.
- [ ] 6.4 Test write token can call write tools and read tools.
- [ ] 6.5 Test missing, malformed, and invalid bearer tokens are rejected for HTTP transport.
- [ ] 6.6 Test HTTP transport startup fails fast when no MCP auth tokens are configured.
- [ ] 6.7 Test validation errors for invalid tool inputs.
- [ ] 6.8 Test `confirm: true` is required before edit and delete storage mutations.
- [ ] 6.9 Run `go test ./...`.
- [ ] 6.10 Run `go build ./...`.
