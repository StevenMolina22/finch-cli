## ADDED Requirements

### Requirement: HTTP bearer authentication
The system SHALL require `Authorization: Bearer <token>` authentication for every remote HTTP MCP request.

#### Scenario: Missing bearer token
- **WHEN** an HTTP MCP client sends a request without an authorization bearer token
- **THEN** the system rejects the request before executing any MCP tool

#### Scenario: Malformed bearer token
- **WHEN** an HTTP MCP client sends an authorization header that is not a bearer token
- **THEN** the system rejects the request before executing any MCP tool

### Requirement: Token sources
The system SHALL read MCP HTTP authentication tokens from `FINCH_MCP_READ_TOKEN` and `FINCH_MCP_WRITE_TOKEN`.

#### Scenario: Tokens loaded from environment
- **WHEN** the MCP HTTP server starts with one or both MCP token environment variables configured
- **THEN** the system uses those token values for HTTP MCP authentication

### Requirement: Invalid token rejection
The system SHALL reject bearer tokens that do not match a configured read or write token.

#### Scenario: Invalid bearer token
- **WHEN** an HTTP MCP client sends a bearer token that does not match any configured MCP token
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

### Requirement: Write token includes read access
The system SHALL allow a valid `FINCH_MCP_WRITE_TOKEN` token to authenticate both read and write MCP tool requests.

#### Scenario: Only write token configured
- **WHEN** HTTP MCP starts with only `FINCH_MCP_WRITE_TOKEN` configured
- **THEN** clients authenticated with the write token can call both read and write MCP tools
