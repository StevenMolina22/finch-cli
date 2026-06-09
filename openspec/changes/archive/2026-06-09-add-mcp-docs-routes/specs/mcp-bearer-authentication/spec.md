## MODIFIED Requirements

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
