## Why

Finch currently provides personal finance tracking only through the CLI, which limits integrations with local dashboards, automation, and other tools that need a long-running HTTP interface. Adding an HTTP API preserves the existing command-line workflow while making the same Turso-backed finance operations available to clients that cannot or should not shell out to `finch` commands.

## What Changes

- Add a `finch serve` Cobra command that starts a Fiber HTTP server.
- Support `finch serve --addr <address>`, defaulting to `:3000`.
- Add an `internal/server` package for Fiber app construction, routes, request parsing, validation, and JSON responses.
- Reuse existing `internal/finch` store construction, store methods, amount parsing, and validation helpers where practical.
- Add unauthenticated JSON endpoints for health, transaction creation, transaction listing, summary retrieval, transaction updates, and transaction deletion.
- Keep `.env` and environment variable loading behavior centralized in `main.go` and unchanged for existing CLI commands.
- Add handler tests using dependency injection or a fake store so server behavior can be tested without a live Turso database.
- Add Fiber as a Go module dependency.

## Capabilities

### New Capabilities
- `http-api`: Expose Finch transaction and summary operations through a Fiber-powered HTTP API.

### Modified Capabilities

None.

## Impact

- Affected code: Cobra command registration, a new `internal/server` package, request/response DTOs, handler validation, store interfaces or adapters for testability, and server tests.
- APIs: new `finch serve` CLI command and HTTP endpoints under `/health`, `/transactions`, `/transactions/:id`, and `/summary`.
- Dependencies: adds `github.com/gofiber/fiber/v2`; continues using `libsql-client-go` directly through existing Finch storage code.
- Systems: server uses the same `FINCH_DB_URL` and `FINCH_TOKEN` environment variables as the CLI and should be bound to localhost or protected by network/deployment controls because authentication is intentionally out of scope for the first version.
