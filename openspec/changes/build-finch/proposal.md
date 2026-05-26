## Why

Personal finance tracking should be fast enough to use at the command line without introducing a full budgeting app or local configuration burden. Finch provides a minimal CLI backed by Turso so transactions can be captured, queried, and summarized from scripts, terminals, and AI agents.

## What Changes

- Add a Go CLI named `finch` built with Cobra.
- Add transaction recording with `finch add [income|expense] <amount> <category> [description]`.
- Add transaction listing with optional `--month YYYY-MM`, `--category <cat>`, and `--json` output.
- Add transaction summaries with optional `--month YYYY-MM` and `--json` output.
- Store all data in a single Turso-backed SQLite `transactions` table using `libsql-client-go` directly, with no ORM.
- Read database connection settings only from `FINCH_DB_URL` and `FINCH_TOKEN` environment variables.

## Capabilities

### New Capabilities
- `transaction-recording`: Capture income and expense transactions with amount, category, optional description, and date in Turso.
- `transaction-querying`: List and summarize transactions with human-readable output by default and JSON output for read commands.

### Modified Capabilities

None.

## Impact

- Affected code: new Go command implementation, Turso database access, transaction schema initialization, and read/write command output formatting.
- APIs: new CLI commands and flags for `add`, `list`, and `summary`.
- Dependencies: Cobra for CLI commands and `libsql-client-go` for Turso access; no ORM or external configuration framework.
- Systems: requires a Turso database URL and auth token supplied through environment variables.
