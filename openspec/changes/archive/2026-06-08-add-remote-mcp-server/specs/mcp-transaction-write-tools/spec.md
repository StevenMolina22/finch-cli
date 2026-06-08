## ADDED Requirements

### Requirement: Add transaction MCP tool
The system SHALL expose `finch_add_transaction` as a write MCP tool accepting `type`, `amount`, `category`, optional `desc`, optional `date`, optional `tags`, and optional `recurring` inputs.

#### Scenario: Add transaction with explicit date
- **WHEN** an authorized write MCP client calls `finch_add_transaction` with valid inputs including `date`
- **THEN** the system stores the transaction and returns a structured success response

#### Scenario: Add transaction without date
- **WHEN** an authorized write MCP client calls `finch_add_transaction` with valid inputs and omits `date`
- **THEN** the system stores the transaction using the current UTC date and returns a structured success response

#### Scenario: Reject invalid add input
- **WHEN** an authorized write MCP client calls `finch_add_transaction` with invalid `type`, `amount`, `category`, `date`, or `recurring`
- **THEN** the system returns a validation error and does not create a transaction

### Requirement: Edit transaction MCP tool
The system SHALL expose `finch_edit_transaction` as a write MCP tool accepting `id`, optional `amount`, optional `category`, optional `desc`, optional `tags`, optional `recurring`, and required `confirm` inputs.

#### Scenario: Edit transaction fields
- **WHEN** an authorized write MCP client calls `finch_edit_transaction` with a valid `id`, at least one valid editable field, and `confirm: true`
- **THEN** the system updates the transaction and returns a structured success response

#### Scenario: Reject edit without editable fields
- **WHEN** an authorized write MCP client calls `finch_edit_transaction` with `confirm: true` and no editable fields
- **THEN** the system returns a validation error and does not update a transaction

#### Scenario: Reject invalid edit input
- **WHEN** an authorized write MCP client calls `finch_edit_transaction` with an invalid `id`, `amount`, `category`, or `recurring`
- **THEN** the system returns a validation error and does not update a transaction

#### Scenario: Edit missing transaction
- **WHEN** an authorized write MCP client calls `finch_edit_transaction` with valid inputs and the target transaction does not exist
- **THEN** the system returns a structured not-found response

### Requirement: Delete transaction MCP tool
The system SHALL expose `finch_delete_transaction` as a write MCP tool accepting `id` and required `confirm` inputs.

#### Scenario: Delete transaction
- **WHEN** an authorized write MCP client calls `finch_delete_transaction` with a valid `id` and `confirm: true`
- **THEN** the system deletes the transaction and returns a structured success response

#### Scenario: Reject invalid delete id
- **WHEN** an authorized write MCP client calls `finch_delete_transaction` with an invalid `id`
- **THEN** the system returns a validation error and does not delete a transaction

#### Scenario: Delete missing transaction
- **WHEN** an authorized write MCP client calls `finch_delete_transaction` with valid inputs and the target transaction does not exist
- **THEN** the system returns a structured not-found response

### Requirement: Shared write logic
The system SHALL implement MCP write tools by calling Finch shared validation and storage logic directly instead of shelling out to CLI commands or calling the Fiber HTTP API internally.

#### Scenario: Write tool execution path
- **WHEN** an authorized MCP client calls a transaction write tool
- **THEN** the tool handler uses `internal/finch` validation and storage or service logic directly
