## ADDED Requirements

### Requirement: Edit confirmation
The system SHALL require `confirm: true` before `finch_edit_transaction` mutates a transaction.

#### Scenario: Edit without confirmation
- **WHEN** an authorized write MCP client calls `finch_edit_transaction` without `confirm: true`
- **THEN** the system returns a confirmation error and does not update a transaction

### Requirement: Delete confirmation
The system SHALL require `confirm: true` before `finch_delete_transaction` deletes a transaction.

#### Scenario: Delete without confirmation
- **WHEN** an authorized write MCP client calls `finch_delete_transaction` without `confirm: true`
- **THEN** the system returns a confirmation error and does not delete a transaction

### Requirement: Confirmation is explicit
The system SHALL treat omitted, false, null, and non-boolean confirmation values as not confirmed.

#### Scenario: Non-true confirmation
- **WHEN** an authorized write MCP client calls a destructive MCP tool with omitted, false, null, or non-boolean `confirm`
- **THEN** the system rejects the tool call before mutating transactions

### Requirement: Confirmation checked before storage mutation
The system SHALL check destructive confirmation before calling storage mutation logic.

#### Scenario: Destructive tool blocked early
- **WHEN** an authorized write MCP client calls a destructive MCP tool without `confirm: true`
- **THEN** the system does not call Finch update, delete, or import storage methods
