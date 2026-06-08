## Why

Finch currently exposes transaction operations through local CLI commands and a planned Fiber HTTP API, but remote AI clients need a protocol-native way to call Finch tools without shelling out or depending on the HTTP API server. A remote-capable MCP server lets non-local AI tools interact with the existing Turso-backed Finch transaction logic while keeping credentials and storage internals server-side.

## What Changes

- Add a `finch mcp` Cobra command for starting an MCP server.
- Support HTTP transport for remote MCP clients with `--transport http` and `--addr`, defaulting HTTP to `:3333`.
- Optionally support `--transport stdio` for local MCP clients if the selected SDK makes it practical without compromising HTTP support.
- Add an `internal/mcp` package for MCP server construction, transport startup, tool registration, authorization, request validation, and handler tests.
- Prefer the official Go MCP SDK, `github.com/modelcontextprotocol/go-sdk/mcp`; document any tradeoff if an alternative SDK is required for HTTP transport.
- Reuse existing `internal/finch` validation, storage, and service logic directly instead of shelling out to CLI commands or calling the Fiber API internally.
- Require bearer-token authentication for remote HTTP MCP using `FINCH_MCP_READ_TOKEN` and `FINCH_MCP_WRITE_TOKEN`.
- Enforce read/write tool permissions so read tokens can only call read tools and write tokens can call both write and read tools.
- Expose required transaction MCP tools for add, list, summary, edit, and delete operations.
- Require `confirm: true` for destructive MCP tools, including edit and delete.
- Document remote deployment assumptions, including the requirement to use HTTPS via direct TLS, a reverse proxy, or a hosting platform.

## Capabilities

### New Capabilities
- `remote-mcp-server-startup`: Start Finch as a remote-capable MCP server over HTTP and optionally stdio.
- `mcp-bearer-authentication`: Authenticate remote HTTP MCP requests with configured bearer tokens.
- `mcp-tool-permissions`: Authorize MCP tool calls according to read and write token permissions.
- `mcp-transaction-read-tools`: Expose transaction listing and summary retrieval as read-only MCP tools.
- `mcp-transaction-write-tools`: Expose transaction add, edit, and delete operations as write-capable MCP tools.
- `mcp-destructive-confirmation`: Require explicit confirmation for destructive MCP tools before mutating data.

### Modified Capabilities

None.

## Impact

- Affected code: Cobra command registration, a new `internal/mcp` package, MCP tool request/response DTOs, shared validation/service adapters, authentication middleware, authorization checks, and MCP handler tests.
- APIs: new `finch mcp` CLI command with `--transport` and `--addr` flags, plus MCP tools named `finch_add_transaction`, `finch_list_transactions`, `finch_get_summary`, `finch_edit_transaction`, and `finch_delete_transaction`.
- Dependencies: prefer adding `github.com/modelcontextprotocol/go-sdk/mcp`; consider `github.com/mark3labs/mcp-go` only if official SDK HTTP transport is impractical and the tradeoff is documented.
- Systems: remote HTTP MCP uses existing Turso/libSQL database configuration while keeping Turso credentials private to the Finch process; HTTP startup requires configured MCP auth tokens and remote deployments must provide HTTPS.
