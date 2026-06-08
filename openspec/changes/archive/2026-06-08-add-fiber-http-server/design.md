## Context

Finch is a Go CLI for personal finance tracking. Cobra commands currently parse user input, validate it with helpers in `internal/finch`, and persist transactions through a Turso/libSQL-backed `finch.Store`. `main.go` loads `.env` before building the root command, and database access is configured only through `FINCH_DB_URL` and `FINCH_TOKEN`.

The HTTP server is an additional interface over the same finance model. It must not replace the CLI, duplicate storage rules, introduce an ORM, or change the transaction schema. The server should live outside `main.go`, use Fiber only inside a server-focused package, and remain testable without a live Turso database.

## Goals / Non-Goals

**Goals:**
- Add `finch serve --addr <address>` with a default address of `:3000`.
- Add an `internal/server` package that constructs a Fiber app and registers routes.
- Reuse `finch.OpenStoreFromEnv`, `finch.ParseAmount`, validation helpers, existing types, and existing store methods where practical.
- Return JSON for all API responses except any explicit CSV endpoints.
- Provide consistent JSON error responses shaped as `{ "error": "message" }`.
- Test handlers with fake storage or dependency injection so tests do not require Turso.
- Preserve all existing CLI behavior and `.env` loading behavior.

**Non-Goals:**
- Replace or redesign CLI commands.
- Add a web UI, MCP interface, budgets, accounts, recurrence automation, local DB support, an ORM, a migration framework, or persistent config files.
- Add full authentication in this first version.
- Redesign the transaction table or persisted transaction JSON shape.
- Implement CSV import/export endpoints unless they remain simple and use Go's `encoding/csv` package.

## Decisions

### Server Package Boundary

Create `internal/server` with a constructor such as `NewApp` or `New` that accepts dependencies and returns `*fiber.App`. Fiber route definitions, request DTOs, response helpers, validation-to-status mapping, and handler functions stay in this package.

Alternative considered: define Fiber setup in `main.go` or `internal/cli`. That would couple transport details to program startup or CLI parsing and make tests harder to isolate.

### Store Dependency Injection

Define a small server-side store interface that includes the methods used by handlers: `Add`, `List`, `Summary`, `Update`, `Delete`, and optionally `Export`/`Import` if CSV endpoints are included. The `finch.Store` type already satisfies these methods. `finch serve` should open the store once during command execution, defer close, construct the Fiber app with that store, and call `Listen(addr)`.

Alternative considered: open `finch.OpenStoreFromEnv` inside every handler. That would make validation failures unnecessarily depend on database configuration, complicate testing, and create avoidable connection churn. The exception is `/health`, which should be available without opening the database if the command/app is constructed in tests without a store.

### CLI Integration

Add a `serve` subcommand to the existing root command. The command will use the injected `OpenStoreFunc` already passed into `NewRootCommand`, preserving testability and ensuring the production path continues to use `finch.OpenStoreFromEnv` from `main.go`. `.env` loading remains only in `main.go`.

Alternative considered: create a separate binary or package-level environment loader. That would add operational complexity and risk changing existing CLI startup behavior.

### Request Parsing and Validation

Handlers should decode JSON bodies into explicit request structs and translate them into `finch.AddInput`, `finch.ListFilter`, and `finch.EditInput`. Shared validation should use existing helpers where available:

- `finch.ValidateType` for transaction type.
- `finch.ParseAmount` for decimal amount strings.
- `finch.ValidateMonth` for `month` query parameters.
- `finch.ValidateLimit` for positive `limit` values when provided.
- `finch.ValidateRecurring` for recurrence metadata.
- Existing category and changed-field checks should be reused or factored into `internal/finch` only if doing so keeps the change minimal.

For `POST /transactions`, omitted `date` maps to `time.Now().UTC().Format("2006-01-02")`, matching current CLI behavior. If `date` is provided, handlers validate `YYYY-MM-DD` and pass the provided value through to storage.

Alternative considered: let the store validate all request values. Current CLI validation happens before store calls, so keeping validation at the transport boundary better preserves existing behavior and enables tests that verify invalid requests do not mutate the store.

### Response Shapes

Successful transaction list responses return `[]finch.Transaction`, and summary responses return `finch.Summary`, preserving the existing JSON tags and shapes. Mutation responses should return simple JSON objects, such as the created transaction input or a status/id object, while using `201` for creation and `200` for updates/deletes.

Errors should consistently return `{ "error": "message" }`. Validation and parse errors map to `400`. Missing transaction IDs map to `404` when detectable from existing store errors. Other store errors map to `500`.

Alternative considered: expose raw Fiber errors or plain text errors. That would make clients harder to implement and would not match the API's JSON-first contract.

### Health Endpoint

`GET /health` should return a static service health payload and not require database access. This keeps readiness for the HTTP process separate from database availability and avoids requiring Turso configuration for basic route tests.

Alternative considered: include database health. That can be a future endpoint or optional readiness check, but it would force `/health` to open or ping the database and conflict with the desired lightweight behavior.

### Security Posture

The first version is intentionally unauthenticated. The design and user-facing documentation should state that the server should bind to localhost, a private interface, or be protected by deployment/network controls. Bearer-token authentication can be added later without changing the core storage interface.

Alternative considered: add bearer token auth immediately. That expands scope and configuration surface before the HTTP API shape is proven.

## Risks / Trade-offs

- Unauthenticated API can expose finance data if bound publicly → Document local/private binding guidance and default operational expectations; consider bearer-token auth as a future enhancement.
- Store-level not-found errors are currently plain errors → Map known `transaction <id> not found` errors to `404` carefully, and avoid broad string matching if a sentinel error can be introduced minimally.
- Opening the store once at `finch serve` startup makes database configuration required before the server listens → This matches transaction endpoint needs and existing CLI configuration rules, while `/health` remains independently testable through the app constructor.
- Optional CSV endpoints could distract from the JSON API → Defer them unless implementation remains small; if included, use `encoding/csv` and consider sharing/fixing CLI CSV handling separately.
- Adding Fiber introduces a new dependency and test surface → Keep Fiber isolated in `internal/server` and verify with `go test ./...` and `go build ./...`.
