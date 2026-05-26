## Why

Personal finance tracking should be fast enough to use at the command line without introducing a full budgeting app or local configuration burden. Finch provides a minimal CLI backed by Turso so transactions can be captured, maintained, queried, summarized, imported, and exported from scripts, terminals, and AI agents.

## What Changes

- Add a Go CLI named `finch` built with Cobra.
- Add transaction recording with `finch add [income|expense] <amount> <category> [description]`.
- Add transaction deletion with `finch delete <id>`.
- Add transaction editing with `finch edit <id> [--amount] [--category] [--desc] [--tags] [--recurring]`.
- Add transaction listing with optional `--month YYYY-MM`, `--category <cat>`, `--limit N`, and `--json` output.
- Add transaction summaries with optional `--month YYYY-MM` and `--json` output that include income, expenses, net balance, and the top 3 expense categories.
- Add CSV export with `finch export [--csv] [--month YYYY-MM]`.
- Add CSV import with `finch import --csv <file>`.
- Store all data in a single Turso-backed SQLite `transactions` table using `libsql-client-go` directly, with no ORM.
- Add `tags TEXT` for comma-separated tags and `recurring TEXT` for nullable `monthly`, `weekly`, or `yearly` recurrence metadata on `transactions`.
- Read database connection settings only from `FINCH_DB_URL` and `FINCH_TOKEN` environment variables.

## Capabilities

### New Capabilities
- `transaction-recording`: Capture income and expense transactions with amount, category, optional description, optional tags, optional recurrence metadata, and date in Turso.
- `transaction-querying`: List and summarize transactions with human-readable output by default and JSON output for read commands.
- `transaction-maintenance`: Edit and delete existing transactions by id.
- `transaction-portability`: Import and export transactions as CSV.

### Modified Capabilities

None.

## Impact

- Affected code: new Go command implementation, Turso database access, transaction schema initialization, read/write command output formatting, CSV parsing/formatting, and edit/delete persistence operations.
- APIs: new CLI commands and flags for `add`, `list`, `summary`, `delete`, `edit`, `export`, and `import`.
- Dependencies: Cobra for CLI commands and `libsql-client-go` for Turso access; no ORM or external configuration framework.
- Systems: requires a Turso database URL and auth token supplied through environment variables.
