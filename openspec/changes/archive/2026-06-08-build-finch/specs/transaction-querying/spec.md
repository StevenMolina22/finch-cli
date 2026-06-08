## ADDED Requirements

### Requirement: List transactions command
The system SHALL provide `finch list` to display persisted transactions in a human-readable format by default.

#### Scenario: List all transactions
- **WHEN** the user runs `finch list`
- **THEN** the command displays all transactions ordered by date with type, amount, category, description, date, tags, and recurring value visible when present

### Requirement: List transaction filters
The system SHALL support `--month YYYY-MM`, `--category <cat>`, and `--limit N` filters on `finch list`, and SHALL combine filters when more than one is provided.

#### Scenario: List by month
- **WHEN** the user runs `finch list --month 2026-05`
- **THEN** the command displays only transactions whose date is in May 2026

#### Scenario: List by category
- **WHEN** the user runs `finch list --category groceries`
- **THEN** the command displays only transactions whose category is `groceries`

#### Scenario: List by month and category
- **WHEN** the user runs `finch list --month 2026-05 --category groceries`
- **THEN** the command displays only `groceries` transactions whose date is in May 2026

#### Scenario: Limit listed transactions
- **WHEN** the user runs `finch list --limit 5`
- **THEN** the command displays no more than 5 transactions

### Requirement: List JSON output
The system SHALL support `--json` on `finch list` and emit machine-readable JSON for AI agents and scripts.

#### Scenario: List as JSON
- **WHEN** the user runs `finch list --json`
- **THEN** the command emits a JSON array of transaction objects containing `id`, `type`, `amount`, `category`, `desc`, `date`, `tags`, and `recurring`

### Requirement: Summary command
The system SHALL provide `finch summary` to display total income, total expenses, net balance, and the top 3 expense categories in a human-readable format by default.

#### Scenario: Summarize all transactions
- **WHEN** the user runs `finch summary`
- **THEN** the command displays income total, expense total, net balance, and the top 3 categories by expense total across all transactions

### Requirement: Summary month filter
The system SHALL support `--month YYYY-MM` on `finch summary` to restrict totals to a single month.

#### Scenario: Summarize one month
- **WHEN** the user runs `finch summary --month 2026-05`
- **THEN** the command displays income total, expense total, net balance, and the top 3 expense categories for May 2026 only

### Requirement: Summary JSON output
The system SHALL support `--json` on `finch summary` and emit machine-readable JSON for AI agents and scripts.

#### Scenario: Summary as JSON
- **WHEN** the user runs `finch summary --month 2026-05 --json`
- **THEN** the command emits a JSON object containing `month`, `income`, `expense`, `net`, and `top_categories`

### Requirement: Read command filter validation
The system SHALL reject read command `--month` values that are not in `YYYY-MM` format and `--limit` values that are not positive integers.

#### Scenario: Invalid month filter
- **WHEN** the user runs `finch list --month May-2026`
- **THEN** the command fails with a clear validation error and does not query transactions

#### Scenario: Invalid list limit
- **WHEN** the user runs `finch list --limit 0`
- **THEN** the command fails with a clear validation error and does not query transactions
