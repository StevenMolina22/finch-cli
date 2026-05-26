## ADDED Requirements

### Requirement: Export transactions as CSV
The system SHALL provide `finch export [--csv] [--month YYYY-MM]` to write transactions in CSV format.

#### Scenario: Export all transactions as CSV
- **WHEN** the user runs `finch export --csv`
- **THEN** the command writes CSV rows for all transactions including `id`, `type`, `amount`, `category`, `desc`, `date`, `tags`, and `recurring`

#### Scenario: Export defaults to CSV
- **WHEN** the user runs `finch export`
- **THEN** the command writes the same CSV output as `finch export --csv`

#### Scenario: Export transactions for one month
- **WHEN** the user runs `finch export --csv --month 2026-05`
- **THEN** the command writes CSV rows only for transactions whose date is in May 2026

### Requirement: Import transactions from CSV
The system SHALL provide `finch import --csv <file>` to read transactions from a CSV file and persist valid rows.

#### Scenario: Import valid CSV file
- **WHEN** the user runs `finch import --csv transactions.csv` and the file contains valid transaction rows
- **THEN** the system inserts those transactions with their amount, category, description, date, tags, and recurring values preserved

#### Scenario: Import missing CSV flag
- **WHEN** the user runs `finch import transactions.csv`
- **THEN** the command fails with a clear usage error and no transactions are imported

#### Scenario: Import invalid CSV row
- **WHEN** the user runs `finch import --csv transactions.csv` and a row has invalid amount, type, date, or recurring value
- **THEN** the command fails with a clear validation error identifying the invalid row and no partial import is committed

### Requirement: CSV validation
The system SHALL validate CSV import data using the same transaction rules as CLI input and SHALL reject export/import month filters that are not in `YYYY-MM` format.

#### Scenario: Invalid export month filter
- **WHEN** the user runs `finch export --csv --month May-2026`
- **THEN** the command fails with a clear validation error and does not query transactions
