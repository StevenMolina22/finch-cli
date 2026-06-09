## ADDED Requirements

### Requirement: Root documentation landing page
The system SHALL expose `/` as a public documentation landing page for the Finch MCP server.

#### Scenario: User opens root URL
- **WHEN** an HTTP client sends `GET /` without an authorization header
- **THEN** the system responds successfully with documentation explaining that Finch is an MCP server, not a browser app

#### Scenario: Root docs identify MCP endpoint
- **WHEN** an HTTP client reads the `/` documentation
- **THEN** the documentation identifies `/mcp` as the MCP endpoint and `Authorization: Bearer <API_KEY>` as the authentication method

#### Scenario: Root docs list tools
- **WHEN** an HTTP client reads the `/` documentation
- **THEN** the documentation lists the available Finch MCP tool names

### Requirement: AI-readable llms documentation
The system SHALL expose `/llms.txt` as public, concise, AI-readable documentation for Finch MCP usage.

#### Scenario: Agent reads llms docs
- **WHEN** an HTTP client sends `GET /llms.txt` without an authorization header
- **THEN** the system responds successfully with documentation containing the base URL guidance, MCP endpoint, authentication method, available tools, and basic usage notes

#### Scenario: llms docs avoid secrets
- **WHEN** an HTTP client reads `/llms.txt`
- **THEN** the documentation uses placeholders for API keys and does not expose configured secret values

### Requirement: Machine-readable MCP metadata
The system SHALL expose `/.well-known/mcp.json` as public machine-readable metadata for Finch MCP discovery.

#### Scenario: Client reads MCP metadata
- **WHEN** an HTTP client sends `GET /.well-known/mcp.json` without an authorization header
- **THEN** the system responds successfully with JSON containing the server name, version, transport, endpoint path, auth type, docs URLs, and available tool names

#### Scenario: Metadata uses relative endpoint
- **WHEN** an HTTP client reads `/.well-known/mcp.json`
- **THEN** the metadata identifies the MCP endpoint as `/mcp`

### Requirement: Docs routes do not initialize MCP sessions
The system SHALL serve documentation and metadata routes without invoking the MCP transport handler or requiring MCP session initialization.

#### Scenario: Docs request without MCP headers
- **WHEN** an HTTP client sends a request to `/`, `/llms.txt`, or `/.well-known/mcp.json` without MCP protocol headers
- **THEN** the system serves the documentation route instead of returning an MCP protocol or session initialization error
