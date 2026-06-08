## 1. Dependencies and Interfaces

- [x] 1.1 Add `github.com/gofiber/fiber/v2` to `go.mod` and refresh `go.sum`.
- [x] 1.2 Add any minimal shared validation or error helpers needed in `internal/finch` without changing existing CLI behavior.
- [x] 1.3 Define a server-side store interface in `internal/server` covering `Add`, `List`, `Summary`, `Update`, `Delete`, and `Close` only where needed.

## 2. Server Package

- [x] 2.1 Create `internal/server` with a `NewApp` or equivalent constructor that accepts a store dependency and a clock function.
- [x] 2.2 Register `GET /health` without requiring database access.
- [x] 2.3 Implement shared JSON response and JSON error helpers with `{ "error": "message" }` error shape.
- [x] 2.4 Implement request parsing and validation helpers for transaction IDs, JSON bodies, date strings, month filters, limits, categories, amounts, and recurring values.

## 3. HTTP Routes

- [x] 3.1 Implement `POST /transactions` using `finch.AddInput`, current UTC date fallback, `finch.ParseAmount`, and existing transaction validation helpers.
- [x] 3.2 Implement `GET /transactions` using `finch.ListFilter` and existing transaction JSON shape.
- [x] 3.3 Implement `GET /summary` using the existing `finch.Summary` JSON shape.
- [x] 3.4 Implement `PATCH /transactions/:id` using `finch.EditInput`, changed-field validation, and existing update validation helpers.
- [x] 3.5 Implement `DELETE /transactions/:id` using the existing store delete method.
- [x] 3.6 Map validation failures to `400`, detectable missing transaction IDs to `404`, create success to `201`, successful reads/updates/deletes to `200`, and unexpected store failures to `500`.

## 4. CLI Integration

- [x] 4.1 Add `finch serve` to the Cobra root command with `--addr` defaulting to `:3000`.
- [x] 4.2 Have `finch serve` open the store with the existing injected store opener, defer close, build the Fiber app through `internal/server`, and call `Listen(addr)`.
- [x] 4.3 Preserve `.env` loading in `main.go` without adding Fiber-specific code there.
- [x] 4.4 Document in command help or related project docs that the first HTTP API is unauthenticated and should bind to localhost or be protected by network controls.

## 5. Tests

- [x] 5.1 Add server handler tests using a fake store so no live Turso database is required.
- [x] 5.2 Test `GET /health` returns JSON and does not require or mutate store state.
- [x] 5.3 Test successful create/list/summary/update/delete JSON responses and status codes.
- [x] 5.4 Test validation failures return JSON error responses and do not call mutating store methods where practical.
- [x] 5.5 Test detectable missing transaction IDs return `404` for update and delete.
- [x] 5.6 Add or update CLI command tests for `serve --addr` wiring without starting a real long-running listener where practical.

## 6. Verification

- [x] 6.1 Run `go test ./...`.
- [x] 6.2 Run `go build ./...`.
- [x] 6.3 Manually review that existing CLI commands and tests preserve current behavior.
