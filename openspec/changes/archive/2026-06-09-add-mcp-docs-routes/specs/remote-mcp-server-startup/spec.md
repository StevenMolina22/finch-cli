## MODIFIED Requirements

### Requirement: HTTP transport
The system SHALL support HTTP transport for remote MCP clients at the `/mcp` endpoint.

#### Scenario: HTTP transport selected
- **WHEN** the user runs `finch mcp --transport http`
- **THEN** the system starts the MCP server using HTTP transport and exposes MCP protocol traffic at `/mcp`

### Requirement: Remote deployment guidance
The system SHALL document that remote HTTP MCP deployments require HTTPS, including deployments where TLS is terminated by a reverse proxy or hosting platform, and SHALL identify `/mcp` as the remote MCP endpoint.

#### Scenario: User reviews MCP usage guidance
- **WHEN** a user reviews MCP command help or project documentation
- **THEN** the system communicates that remote HTTP MCP must be deployed behind HTTPS and that MCP clients should connect to `/mcp`
