## ADDED Requirements

### Requirement: Environment-based database configuration
The system SHALL read the Turso database URL from `FINCH_DB_URL` and the Turso auth token from `FINCH_TOKEN` for every command that accesses persisted transactions.

#### Scenario: Missing database URL
- **WHEN** a transaction command runs without `FINCH_DB_URL` set
- **THEN** the command fails with a clear configuration error and does not attempt a database operation

#### Scenario: Missing auth token
- **WHEN** a transaction command runs without `FINCH_TOKEN` set
- **THEN** the command fails with a clear configuration error and does not attempt a database operation

### Requirement: Single transaction table
The system SHALL persist all finance entries in a single SQLite table named `transactions` with columns `id`, `type`, `amount`, `category`, `desc`, `date`, `tags`, and `recurring`.

#### Scenario: First transaction command initializes storage
- **WHEN** a transaction command runs against a database without the `transactions` table
- **THEN** the system creates the table before completing the requested operation

#### Scenario: Existing transaction table receives metadata columns
- **WHEN** a transaction command runs against a database with `transactions` missing `tags` or `recurring`
- **THEN** the system updates the table schema so both metadata columns are available before completing the requested operation

### Requirement: Transaction metadata fields
The system SHALL store `tags` as comma-separated text and `recurring` as nullable text limited to `monthly`, `weekly`, or `yearly`.

#### Scenario: Transaction without tags or recurrence
- **WHEN** a transaction is recorded without tags or recurrence metadata
- **THEN** the system stores an empty `tags` value and a null `recurring` value

#### Scenario: Invalid recurrence value
- **WHEN** a command attempts to store `recurring` as a value other than `monthly`, `weekly`, or `yearly`
- **THEN** the command fails with a clear validation error and no transaction change is persisted

### Requirement: Add transaction command
The system SHALL provide `finch add [income|expense] <amount> <category> [description]` to record one transaction with the supplied type, amount, category, optional description, and the current date.

#### Scenario: Add valid income
- **WHEN** the user runs `finch add income 1250.00 salary "May paycheck"`
- **THEN** the system records an income transaction with amount `1250.00`, category `salary`, description `May paycheck`, and today's date

#### Scenario: Add valid expense without description
- **WHEN** the user runs `finch add expense 14.50 coffee`
- **THEN** the system records an expense transaction with amount `14.50`, category `coffee`, an empty description, and today's date

### Requirement: Transaction input validation
The system SHALL reject transactions with a type other than `income` or `expense`, a non-positive amount, an amount with more than two fractional digits, or an empty category.

#### Scenario: Invalid transaction type
- **WHEN** the user runs `finch add transfer 10.00 savings`
- **THEN** the command fails and no transaction is recorded

#### Scenario: Invalid amount
- **WHEN** the user runs `finch add expense 0 groceries`
- **THEN** the command fails and no transaction is recorded

#### Scenario: Too many fractional digits
- **WHEN** the user runs `finch add expense 1.999 groceries`
- **THEN** the command fails and no transaction is recorded
