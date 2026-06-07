## Context

Finch is a Go personal finance utility with a Cobra CLI, shared transaction logic in `internal/finch`, and a planned Fiber HTTP API in `internal/server`. Transactions are stored in a remote Turso/libSQL database and runtime configuration is loaded from `.env` or environment variables. The MCP server must be a separate interface that serves non-local AI clients without shelling out to CLI commands and without requiring the Fiber server to run.

The official Go MCP SDK currently documents Streamable HTTP server support through `mcp.NewStreamableHTTPHandler` and client support through `mcp.StreamableClientTransport`, so the first implementation should use `github.com/modelcontextprotocol/go-sdk/mcp` unless implementation exposes a concrete blocker.

## Goals / Non-Goals

**Goals:**
- Add `finch mcp --transport http --addr :3333` for remote MCP usage.
- Keep MCP-specific transport, auth, tool registration, and handler code in `internal/mcp`.
- Reuse `internal/finch` types, validation helpers, and store operations directly.
- Require bearer-token authentication for HTTP transport before any MCP tool executes.
- Enforce read and write permissions per tool.
- Return structured MCP tool results using the same transaction and summary JSON shapes Finch already exposes.
- Make destructive tools require explicit `confirm: true`.
- Add tests around handlers, authorization, startup auth configuration, validation, and destructive confirmation.

**Non-Goals:**
- Replace the CLI or Fiber HTTP API.
- Add OAuth, multi-user accounts, web UI, direct remote access to Turso credentials, local DB support, an ORM, a migration framework, budgets, accounts, recurrence automation, or schema redesign.
- Include import/export MCP tools in the first implementation unless CSV handling is first corrected to use Go's `encoding/csv`.
- Terminate TLS in Finch itself; deployments can provide HTTPS through a reverse proxy or hosting platform.

## Decisions

### Use the official Go MCP SDK first

Use `github.com/modelcontextprotocol/go-sdk/mcp` for server construction and Streamable HTTP transport. Current docs show `mcp.NewStreamableHTTPHandler` for HTTP MCP sessions, which satisfies the remote transport requirement.

Alternative considered: `github.com/mark3labs/mcp-go`. It remains a fallback only if the official SDK cannot practically support the required HTTP server behavior during implementation. If the fallback is used, document the issue and the reason in code comments or project docs.

### Add an `internal/mcp` package

Place MCP tool definitions, request structs, auth helpers, permission checks, server construction, and transport startup in `internal/mcp`. The CLI package should only register `finch mcp`, parse flags, open the store, and delegate startup to `internal/mcp`.

Alternative considered: implement MCP wiring directly in `internal/cli`. That would couple command parsing with protocol and authorization behavior, making tests and future transport changes harder.

### Use dependency injection around storage

Define a narrow MCP store interface with the methods needed by MCP tools: `Add`, `List`, `Summary`, `Update`, and `Delete`. Production code adapts the existing Finch store; tests provide fakes.

Alternative considered: use `*finch.Store` directly in handlers. That would make handler tests depend on libSQL setup and reduce isolation.

### Authenticate HTTP before MCP handling

Wrap the HTTP MCP handler with bearer-token middleware that reads the `Authorization: Bearer <token>` header, rejects missing or malformed headers, compares against configured tokens with constant-time comparison, and records the authenticated permission level for tool authorization. Token values must never be logged or included in errors.

If HTTP transport is requested and neither MCP token is configured, startup fails before binding. If only `FINCH_MCP_WRITE_TOKEN` is configured, read tools are available through the write token because write permission includes read permission. A read token alone allows the server to start but only read tools can be used.

Alternative considered: per-tool token fields in tool input. That would expose secrets to MCP tool logs and model context and is not acceptable.

### Map tools directly to Finch operations

Register required tools:
- `finch_add_transaction` maps to `finch.AddInput` and `Store.Add`.
- `finch_list_transactions` maps to `finch.ListFilter` and `Store.List`.
- `finch_get_summary` maps to `Store.Summary`.
- `finch_edit_transaction` maps to `finch.EditInput` and `Store.Update`.
- `finch_delete_transaction` maps to `Store.Delete`.

Handlers should trim string inputs, parse amount strings through `finch.ParseAmount`, validate months with `finch.ValidateMonth`, validate dates with `finch.ValidateDate`, validate recurring values with `finch.ValidateRecurring`, validate limits with `finch.ValidateLimit` when provided, require positive numeric IDs, and call `finch.ValidateEditFields` for edit requests.

Alternative considered: call existing Cobra commands or Fiber endpoints internally. That would duplicate protocol conversion, complicate errors, and violate the requirement that MCP reuse shared logic directly.

### Return structured results and safe errors

Tool responses should be structured JSON-compatible objects. Successful add, edit, and delete responses should include a success indicator, the affected transaction id when available, and a clear message. Missing edit/delete targets should return structured not-found results rather than leaking storage internals. Validation errors should be clear and specific but must not include environment variable values or token material.

## Risks / Trade-offs

- Official SDK transport behavior changes while it is still evolving -> Pin the module version during implementation, add handler tests, and document any SDK-specific assumptions.
- HTTP bearer tokens protect only the MCP endpoint, not transport secrecy -> Document that remote deployments MUST use HTTPS through direct TLS, a reverse proxy, or the hosting platform.
- Long-lived server holds one store instance -> Ensure the command closes the store on shutdown and handler code is safe for concurrent requests using the existing `database/sql` backed store.
- Ambiguous MCP auth context propagation through the SDK -> Enforce auth at HTTP middleware and pass permission context through request context before the Streamable HTTP handler receives the request.
- Existing CSV import/export parsing is manual -> Exclude import/export MCP tools from the first version unless CSV handling is corrected to `encoding/csv` first.

## Migration Plan

1. Add the MCP SDK dependency and `internal/mcp` package without changing existing CLI behavior.
2. Add `finch mcp` command and flags alongside existing commands.
3. Add HTTP auth startup validation and middleware.
4. Register required MCP tools and connect them to shared Finch logic.
5. Add tests for handler validation, authorization, startup token checks, and destructive confirmation.
6. Run `go test ./...` and `go build ./...`.

Rollback is removing the new `finch mcp` command, `internal/mcp` package, dependency entries, and related tests. No database migration or schema rollback is required.

## Open Questions

- Whether stdio transport should be included in the first implementation depends on the official SDK surface and test cost after HTTP is complete.
- Whether add responses should include the created transaction id depends on whether existing storage returns it or should be minimally extended to do so.
