## ADDED Requirements

### Requirement: List transactions MCP tool
The system SHALL expose `finch_list_transactions` as a read-only MCP tool accepting optional `month`, optional `category`, and optional `limit` inputs.

#### Scenario: List transactions without filters
- **WHEN** an authorized MCP client calls `finch_list_transactions` without filters
- **THEN** the system returns transactions using the existing Finch transaction JSON shape

#### Scenario: List transactions with filters
- **WHEN** an authorized MCP client calls `finch_list_transactions` with valid `month`, `category`, and `limit` inputs
- **THEN** the system applies those filters and returns matching transactions using the existing Finch transaction JSON shape

#### Scenario: Reject invalid list filters
- **WHEN** an authorized MCP client calls `finch_list_transactions` with an invalid `month` or `limit`
- **THEN** the system returns a validation error and does not query with invalid filters

### Requirement: Summary MCP tool
The system SHALL expose `finch_get_summary` as a read-only MCP tool accepting an optional `month` input.

#### Scenario: Get summary without month
- **WHEN** an authorized MCP client calls `finch_get_summary` without `month`
- **THEN** the system returns the overall summary using the existing Finch summary JSON shape

#### Scenario: Get summary with month
- **WHEN** an authorized MCP client calls `finch_get_summary` with a valid `month`
- **THEN** the system returns the summary for that month using the existing Finch summary JSON shape

#### Scenario: Reject invalid summary month
- **WHEN** an authorized MCP client calls `finch_get_summary` with an invalid `month`
- **THEN** the system returns a validation error and does not query with the invalid month

### Requirement: Shared read logic
The system SHALL implement MCP read tools by calling Finch shared validation and storage logic directly instead of shelling out to CLI commands or calling the Fiber HTTP API internally.

#### Scenario: Read tool execution path
- **WHEN** an authorized MCP client calls a transaction read tool
- **THEN** the tool handler uses `internal/finch` validation and storage or service logic directly
