# mcp-bearer-authentication Specification

## Purpose
TBD - created by archiving change add-remote-mcp-server. Update Purpose after archive.
## Requirements
### Requirement: HTTP bearer authentication
The system SHALL require `Authorization: Bearer <token>` authentication for every remote HTTP MCP request sent to the `/mcp` endpoint.

#### Scenario: Missing bearer token
- **WHEN** an HTTP MCP client sends a request to `/mcp` without an authorization bearer token
- **THEN** the system rejects the request before executing any MCP tool

#### Scenario: Malformed bearer token
- **WHEN** an HTTP MCP client sends a request to `/mcp` with an authorization header that is not a bearer token
- **THEN** the system rejects the request before executing any MCP tool

#### Scenario: Public docs do not require bearer token
- **WHEN** an HTTP client sends a request to `/`, `/llms.txt`, or `/.well-known/mcp.json` without an authorization bearer token
- **THEN** the system does not reject the request using MCP bearer authentication

### Requirement: Token sources
The system SHALL read MCP HTTP authentication from `FINCH_API_KEY`.

#### Scenario: API key loaded from environment
- **WHEN** the MCP HTTP server starts with `FINCH_API_KEY` configured
- **THEN** the system uses that key value for HTTP MCP authentication

### Requirement: Invalid token rejection
The system SHALL reject bearer tokens that do not match the configured API key.

#### Scenario: Invalid bearer token
- **WHEN** an HTTP MCP client sends a bearer token that does not match the configured API key
- **THEN** the system rejects the request before executing any MCP tool

### Requirement: Secret-safe authentication
The system SHALL NOT log token values or include token values in authentication errors.

#### Scenario: Authentication failure message
- **WHEN** an HTTP MCP request is rejected because of a missing, malformed, or invalid token
- **THEN** the response and logs do not contain the supplied token or configured token values

### Requirement: Constant-time token comparison
The system SHALL compare supplied bearer tokens to configured token values using constant-time comparison.

#### Scenario: Token comparison
- **WHEN** the system validates a supplied bearer token
- **THEN** token equality is evaluated without early-return string comparison that leaks matching prefixes

### Requirement: API key includes full access
The system SHALL allow a valid `FINCH_API_KEY` token to authenticate both read and write MCP tool requests.

#### Scenario: API key configured
- **WHEN** HTTP MCP starts with `FINCH_API_KEY` configured
- **THEN** clients authenticated with the API key can call both read and write MCP tools
