# mcp-tool-permissions Specification

## Purpose
TBD - created by archiving change add-remote-mcp-server. Update Purpose after archive.
## Requirements
### Requirement: Read tool permission
The system SHALL classify `finch_list_transactions` and `finch_get_summary` as read tools.

#### Scenario: Read token calls read tool
- **WHEN** an HTTP MCP client authenticates with `FINCH_MCP_READ_TOKEN` and calls `finch_list_transactions` or `finch_get_summary`
- **THEN** the system allows the tool call subject to normal input validation

### Requirement: Write tool permission
The system SHALL classify `finch_add_transaction`, `finch_edit_transaction`, and `finch_delete_transaction` as write tools.

#### Scenario: Write token calls write tool
- **WHEN** an HTTP MCP client authenticates with `FINCH_MCP_WRITE_TOKEN` and calls a write tool
- **THEN** the system allows the tool call subject to normal input validation and confirmation requirements

### Requirement: Read token write denial
The system SHALL prevent clients authenticated with `FINCH_MCP_READ_TOKEN` from calling write tools.

#### Scenario: Read token calls write tool
- **WHEN** an HTTP MCP client authenticates with `FINCH_MCP_READ_TOKEN` and calls `finch_add_transaction`, `finch_edit_transaction`, or `finch_delete_transaction`
- **THEN** the system rejects the tool call without mutating transactions

### Requirement: Unauthenticated tool denial
The system SHALL prevent unauthenticated HTTP MCP clients from calling any MCP tool.

#### Scenario: Unauthenticated client calls read tool
- **WHEN** an HTTP MCP client without a valid bearer token calls `finch_get_summary`
- **THEN** the system rejects the tool call before reading transaction data

### Requirement: Permission-safe errors
The system SHALL return clear authorization errors without exposing configured token values or Turso credentials.

#### Scenario: Unauthorized tool call error
- **WHEN** an MCP tool call is rejected because the caller lacks permission
- **THEN** the error explains that the tool is not permitted for the authenticated token class and does not include secrets
