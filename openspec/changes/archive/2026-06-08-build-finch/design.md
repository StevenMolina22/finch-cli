## Context

Finch is a new Go module with only `go.mod` present. The change introduces a small CLI for personal finance tracking with a Turso-backed SQLite database, Cobra command routing, direct `libsql-client-go` access, and no ORM or external configuration files.

The CLI must stay simple enough to explain in one `SKILL.md`: user-facing commands for recording, editing, deleting, listing, summarizing, importing, and exporting transactions; one database table; environment-based configuration; and readable terminal output with JSON support for read commands.

## Goals / Non-Goals

**Goals:**
- Provide `finch add`, `finch list`, `finch summary`, `finch delete`, `finch edit`, `finch export`, and `finch import` commands.
- Store transactions in one `transactions` table with `id`, `type`, `amount`, `category`, `desc`, `date`, `tags`, and `recurring` columns.
- Support comma-separated tags and nullable recurrence values of `monthly`, `weekly`, or `yearly`.
- Use `FINCH_DB_URL` and `FINCH_TOKEN` as the only runtime configuration source.
- Keep persistence code direct and explicit with `libsql-client-go`; avoid ORMs and migration frameworks.
- Return human-readable output by default and JSON output for `list` and `summary` when `--json` is set.
- Support CSV import and export for transaction portability.

**Non-Goals:**
- No budgets, accounts, date override flags, or recurrence automation beyond storing the recurring value.
- No config files, interactive setup, local credential storage, or shell completion customization.
- No web server, TUI, background process, or sync layer outside Turso.
- No ORM, application framework, or dependency injection framework.

## Decisions

### Use Cobra for CLI shape

Use Cobra to define the root command and subcommands. This satisfies the requested stack while keeping command parsing, usage text, argument validation, and flags straightforward.

Alternative considered: manual `os.Args` parsing. It would reduce one dependency but would make usage text and validation noisier than the requested Cobra-based CLI.

### Use direct libsql calls with a small storage package

Create a small database layer that opens the Turso client from environment variables, ensures the `transactions` table exists, and exposes focused functions for inserting, updating, deleting, listing, summarizing, importing, and exporting transactions.

Alternative considered: placing SQL directly inside Cobra handlers. That would be fewer files initially, but it would mix CLI formatting, validation, and persistence, making tests and future explanation harder.

### Store amount as integer cents

Parse CLI amount values as positive decimal currency values and store them in the `amount` column as integer cents. Format cents back to decimal strings for human, JSON, and CSV output.

Alternative considered: SQLite `REAL`. It is simpler to insert but risks floating point surprises for finance data.

### Store date as current UTC date string

Set `date` automatically when adding a transaction using the current UTC date in `YYYY-MM-DD` format. Month filtering can then use a `YYYY-MM` prefix comparison without extra date parsing in SQLite.

Alternative considered: RFC3339 timestamps. Timestamps are more precise, but the requested schema and commands only need day and month-level reporting.

### Keep optional metadata simple

Store tags as a comma-separated `TEXT` field and recurring cadence as nullable `TEXT` limited to `monthly`, `weekly`, or `yearly`. The CLI should validate recurring values but does not need to expand recurring transactions automatically.

Alternative considered: separate tag and recurrence tables. That would improve normalization, but it conflicts with the requested single-table schema and adds unnecessary complexity.

### Keep JSON output stable and explicit

For `list --json`, emit an array of transaction objects. For `summary --json`, emit an object with `month`, `income`, `expense`, `net`, and `top_categories` fields using decimal strings for amounts. Human output can remain optimized for reading and does not need to match the JSON shape exactly.

Alternative considered: JSON numbers for amounts. Decimal strings avoid precision ambiguity for agents and downstream scripts.

### Use CSV as an explicit interchange format

Support `finch export --csv` for stdout CSV output and `finch import --csv <file>` for loading transactions from a CSV file. CSV should include all transaction fields so tags and recurrence metadata round-trip.

Alternative considered: implicit CSV export without a flag. Keeping `--csv` explicit leaves room for future export formats without changing command names.

## Risks / Trade-offs

- Turso connection errors can make all commands fail → Fail fast with clear messages when `FINCH_DB_URL` or `FINCH_TOKEN` are missing or the database cannot be reached.
- Automatic UTC dates may differ from a user's local calendar day near midnight → Keep the implementation deterministic and document that transactions use UTC until a future date option is added.
- Integer cents assume two decimal places → Validate input and reject values with more than two fractional digits.
- Using `desc` as a column name may be awkward because `DESC` is a SQL keyword → Quote the column as `"desc"` in SQL statements.
- Auto-creating or updating the table on command execution hides schema setup failures until runtime → Run table initialization before operations and surface errors directly.
- CSV import can introduce invalid data → Validate each imported row using the same rules as CLI input and fail clearly when rows are malformed.
