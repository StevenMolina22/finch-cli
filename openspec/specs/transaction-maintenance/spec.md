## ADDED Requirements

### Requirement: Delete transaction command
The system SHALL provide `finch delete <id>` to remove one persisted transaction by id.

#### Scenario: Delete existing transaction
- **WHEN** the user runs `finch delete 42` and transaction `42` exists
- **THEN** the system deletes transaction `42` and reports that it was deleted

#### Scenario: Delete missing transaction
- **WHEN** the user runs `finch delete 42` and transaction `42` does not exist
- **THEN** the command fails with a clear not-found error

### Requirement: Edit transaction command
The system SHALL provide `finch edit <id> [--amount] [--category] [--desc] [--tags] [--recurring]` to update editable fields on an existing transaction.

#### Scenario: Edit amount
- **WHEN** the user runs `finch edit 42 --amount 19.99`
- **THEN** the system updates transaction `42` to store amount `19.99` and leaves all other fields unchanged

#### Scenario: Edit category and description
- **WHEN** the user runs `finch edit 42 --category groceries --desc "weekly shop"`
- **THEN** the system updates transaction `42` category and description and leaves all other fields unchanged

#### Scenario: Edit tags
- **WHEN** the user runs `finch edit 42 --tags "food,household"`
- **THEN** the system stores `food,household` in the transaction `tags` field and leaves all other fields unchanged

#### Scenario: Edit recurring cadence
- **WHEN** the user runs `finch edit 42 --recurring monthly`
- **THEN** the system stores `monthly` in the transaction `recurring` field and leaves all other fields unchanged

### Requirement: Edit validation
The system SHALL validate edit inputs using the same amount, category, tags, and recurring rules used by transaction storage.

#### Scenario: Edit without changed fields
- **WHEN** the user runs `finch edit 42`
- **THEN** the command fails with a clear validation error and no transaction change is persisted

#### Scenario: Edit missing transaction
- **WHEN** the user runs `finch edit 42 --amount 19.99` and transaction `42` does not exist
- **THEN** the command fails with a clear not-found error

#### Scenario: Edit invalid amount
- **WHEN** the user runs `finch edit 42 --amount 1.999`
- **THEN** the command fails with a clear validation error and no transaction change is persisted

#### Scenario: Edit invalid recurring cadence
- **WHEN** the user runs `finch edit 42 --recurring daily`
- **THEN** the command fails with a clear validation error and no transaction change is persisted
