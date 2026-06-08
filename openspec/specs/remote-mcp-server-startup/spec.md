# remote-mcp-server-startup Specification

## Purpose
TBD - created by archiving change add-remote-mcp-server. Update Purpose after archive.
## Requirements
### Requirement: MCP command
The system SHALL provide `finch mcp` to start Finch as an MCP server without requiring the Fiber HTTP API server to be running.

#### Scenario: Start MCP command
- **WHEN** the user runs `finch mcp --transport http --addr :3333` with valid database and MCP auth environment variables
- **THEN** the system starts an MCP HTTP server and exposes Finch MCP tools through that server

### Requirement: HTTP transport
The system SHALL support HTTP transport for remote MCP clients.

#### Scenario: HTTP transport selected
- **WHEN** the user runs `finch mcp --transport http`
- **THEN** the system starts the MCP server using HTTP transport

### Requirement: Default HTTP address
The system SHALL default the HTTP MCP listen address to `:3333` when `--addr` is not supplied.

#### Scenario: HTTP default address
- **WHEN** the user runs `finch mcp --transport http` without `--addr`
- **THEN** the MCP HTTP server listens on `:3333`

### Requirement: Custom HTTP address
The system SHALL support a custom HTTP listen address through `--addr`.

#### Scenario: HTTP custom address
- **WHEN** the user runs `finch mcp --transport http --addr 127.0.0.1:4444`
- **THEN** the MCP HTTP server listens on `127.0.0.1:4444`

### Requirement: Unsupported transport rejection
The system SHALL reject unsupported `--transport` values before opening the MCP server.

#### Scenario: Unsupported transport
- **WHEN** the user runs `finch mcp --transport websocket`
- **THEN** the command fails with a clear unsupported transport error

### Requirement: HTTP auth startup configuration
The system SHALL fail fast before binding an HTTP MCP listener when HTTP transport is requested and no MCP bearer token is configured.

#### Scenario: HTTP startup without auth tokens
- **WHEN** the user runs `finch mcp --transport http` without `FINCH_MCP_READ_TOKEN` and without `FINCH_MCP_WRITE_TOKEN`
- **THEN** the command fails with a clear startup error before accepting HTTP requests

### Requirement: Stdio transport optionality
The system MAY support stdio transport for local MCP clients, but HTTP transport SHALL remain available and prioritized.

#### Scenario: Stdio not implemented
- **WHEN** the user runs `finch mcp --transport stdio` and stdio support is not included
- **THEN** the command fails with a clear unsupported transport error while `--transport http` remains supported

### Requirement: Remote deployment guidance
The system SHALL document that remote HTTP MCP deployments require HTTPS, including deployments where TLS is terminated by a reverse proxy or hosting platform.

#### Scenario: User reviews MCP usage guidance
- **WHEN** a user reviews MCP command help or project documentation
- **THEN** the system communicates that remote HTTP MCP must be deployed behind HTTPS
