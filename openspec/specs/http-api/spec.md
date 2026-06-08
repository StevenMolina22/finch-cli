# http-api Specification

## Purpose
TBD - created by archiving change add-fiber-http-server. Update Purpose after archive.
## Requirements
### Requirement: Serve command
The system SHALL provide `finch serve` to start a Fiber-powered HTTP API server as an additional interface to Finch.

#### Scenario: Serve with default address
- **WHEN** the user runs `finch serve` with valid database environment variables
- **THEN** the system starts the HTTP server listening on `:3000`

#### Scenario: Serve with custom address
- **WHEN** the user runs `finch serve --addr 127.0.0.1:8080` with valid database environment variables
- **THEN** the system starts the HTTP server listening on `127.0.0.1:8080`

### Requirement: Server database configuration
The system SHALL use the same `FINCH_DB_URL` and `FINCH_TOKEN` environment variable configuration path for transaction HTTP endpoints that existing CLI transaction commands use.

#### Scenario: Missing database configuration on serve
- **WHEN** the user runs `finch serve` without required database environment variables
- **THEN** the command fails with a clear configuration error before starting the server

### Requirement: Health endpoint
The system SHALL expose `GET /health` returning JSON service health without requiring database access.

#### Scenario: Health check
- **WHEN** a client requests `GET /health`
- **THEN** the system responds with status `200` and a JSON health object

### Requirement: Create transaction endpoint
The system SHALL expose `POST /transactions` to create one income or expense transaction from a JSON body containing `type`, `amount`, `category`, optional `desc`, optional `date`, optional `tags`, and optional `recurring`.

#### Scenario: Create valid transaction with explicit date
- **WHEN** a client posts a valid transaction JSON body with `date` set to `2026-05-31`
- **THEN** the system stores the transaction with the supplied date and responds with status `201` and JSON

#### Scenario: Create valid transaction without date
- **WHEN** a client posts a valid transaction JSON body without `date`
- **THEN** the system stores the transaction using the current UTC date and responds with status `201` and JSON

#### Scenario: Reject invalid create request
- **WHEN** a client posts a transaction JSON body with invalid `type`, `amount`, `category`, `date`, or `recurring`
- **THEN** the system responds with status `400`, returns `{ "error": "message" }`, and does not create a transaction

### Requirement: List transactions endpoint
The system SHALL expose `GET /transactions` returning a JSON array of transactions using the existing transaction JSON shape and supporting `month`, `category`, and `limit` query parameters.

#### Scenario: List transactions without filters
- **WHEN** a client requests `GET /transactions`
- **THEN** the system responds with status `200` and a JSON array of transactions

#### Scenario: List transactions with filters
- **WHEN** a client requests `GET /transactions?month=2026-05&category=groceries&limit=5`
- **THEN** the system applies all supplied filters and responds with status `200` and a JSON array of transactions

#### Scenario: Reject invalid list filter
- **WHEN** a client requests `GET /transactions` with an invalid `month` or `limit` query parameter
- **THEN** the system responds with status `400` and returns `{ "error": "message" }`

### Requirement: Summary endpoint
The system SHALL expose `GET /summary` returning a JSON summary using the existing summary JSON shape and supporting an optional `month` query parameter.

#### Scenario: Summary without month filter
- **WHEN** a client requests `GET /summary`
- **THEN** the system responds with status `200` and a JSON summary object

#### Scenario: Summary with month filter
- **WHEN** a client requests `GET /summary?month=2026-05`
- **THEN** the system responds with status `200` and a JSON summary object for May 2026

#### Scenario: Reject invalid summary month
- **WHEN** a client requests `GET /summary?month=May-2026`
- **THEN** the system responds with status `400` and returns `{ "error": "message" }`

### Requirement: Update transaction endpoint
The system SHALL expose `PATCH /transactions/:id` to update editable transaction fields `amount`, `category`, `desc`, `tags`, and `recurring`.

#### Scenario: Update valid fields
- **WHEN** a client requests `PATCH /transactions/42` with at least one valid editable field
- **THEN** the system updates transaction `42` and responds with status `200` and JSON

#### Scenario: Reject update without fields
- **WHEN** a client requests `PATCH /transactions/42` without any editable field
- **THEN** the system responds with status `400`, returns `{ "error": "message" }`, and does not update a transaction

#### Scenario: Reject invalid update request
- **WHEN** a client requests `PATCH /transactions/not-an-id` or supplies an invalid editable field value
- **THEN** the system responds with status `400` and returns `{ "error": "message" }`

#### Scenario: Update missing transaction
- **WHEN** a client requests `PATCH /transactions/42` with valid fields and transaction `42` does not exist
- **THEN** the system responds with status `404` and returns `{ "error": "message" }`

### Requirement: Delete transaction endpoint
The system SHALL expose `DELETE /transactions/:id` to delete one transaction by id.

#### Scenario: Delete existing transaction
- **WHEN** a client requests `DELETE /transactions/42` and transaction `42` exists
- **THEN** the system deletes transaction `42` and responds with status `200` and JSON

#### Scenario: Reject invalid delete id
- **WHEN** a client requests `DELETE /transactions/not-an-id`
- **THEN** the system responds with status `400` and returns `{ "error": "message" }`

#### Scenario: Delete missing transaction
- **WHEN** a client requests `DELETE /transactions/42` and transaction `42` does not exist
- **THEN** the system responds with status `404` and returns `{ "error": "message" }`

### Requirement: JSON error responses
The system SHALL return JSON error responses shaped as `{ "error": "message" }` for HTTP API validation, not-found, and internal errors.

#### Scenario: Internal store failure
- **WHEN** a transaction HTTP endpoint encounters an unexpected store error
- **THEN** the system responds with status `500` and returns `{ "error": "message" }`

### Requirement: HTTP API security guidance
The system SHALL document that the first HTTP API version is unauthenticated and must be bound to localhost, a private interface, or protected by deployment or network controls.

#### Scenario: User reviews server usage guidance
- **WHEN** a user reviews the server command help or project documentation for the HTTP API
- **THEN** the system communicates that the API is unauthenticated and should not be exposed directly to untrusted networks

