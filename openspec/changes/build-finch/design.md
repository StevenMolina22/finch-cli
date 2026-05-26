## Context

Finch is a new Go module with only `go.mod` present. The change introduces a small CLI for personal finance tracking with a Turso-backed SQLite database, Cobra command routing, direct `libsql-client-go` access, and no ORM or external configuration files.

The CLI must stay simple enough to explain in one `SKILL.md`: three user-facing commands, one database table, environment-based configuration, and readable terminal output with JSON support for read commands.

## Goals / Non-Goals

**Goals:**
- Provide `finch add`, `finch list`, and `finch summary` commands.
- Store transactions in one `transactions` table with `id`, `type`, `amount`, `category`, `desc`, and `date` columns.
- Use `FINCH_DB_URL` and `FINCH_TOKEN` as the only runtime configuration source.
- Keep persistence code direct and explicit with `libsql-client-go`; avoid ORMs and migration frameworks.
- Return human-readable output by default and JSON output for `list` and `summary` when `--json` is set.

**Non-Goals:**
- No budgets, recurring transactions, accounts, tags, imports, exports, or date override flags.
- No config files, interactive setup, local credential storage, or shell completion customization.
- No web server, TUI, background process, or sync layer outside Turso.
- No ORM, application framework, or dependency injection framework.

## Decisions

### Use Cobra for CLI shape

Use Cobra to define the root command and the three subcommands. This satisfies the requested stack while keeping command parsing, usage text, argument validation, and flags straightforward.

Alternative considered: manual `os.Args` parsing. It would reduce one dependency but would make usage text and validation noisier than the requested Cobra-based CLI.

### Use direct libsql calls with a small storage package

Create a small database layer that opens the Turso client from environment variables, ensures the `transactions` table exists, and exposes focused functions for inserting, listing, and summarizing transactions.

Alternative considered: placing SQL directly inside Cobra handlers. That would be fewer files initially, but it would mix CLI formatting, validation, and persistence, making tests and future explanation harder.

### Store amount as integer cents

Parse the CLI `<amount>` as a positive decimal currency value and store it in the `amount` column as integer cents. Format cents back to decimal strings for human and JSON output.

Alternative considered: SQLite `REAL`. It is simpler to insert but risks floating point surprises for finance data.

### Store date as current UTC date string

Set `date` automatically when adding a transaction using the current UTC date in `YYYY-MM-DD` format. Month filtering can then use a `YYYY-MM` prefix comparison without extra date parsing in SQLite.

Alternative considered: RFC3339 timestamps. Timestamps are more precise, but the requested schema and commands only need day and month-level reporting.

### Keep JSON output stable and explicit

For `list --json`, emit an array of transaction objects. For `summary --json`, emit an object with `month`, `income`, `expense`, and `net` fields as decimal strings. Human output can remain optimized for reading and does not need to match the JSON shape exactly.

Alternative considered: JSON numbers for amounts. Decimal strings avoid precision ambiguity for agents and downstream scripts.

## Risks / Trade-offs

- Turso connection errors can make all commands fail → Fail fast with clear messages when `FINCH_DB_URL` or `FINCH_TOKEN` are missing or the database cannot be reached.
- Automatic UTC dates may differ from a user's local calendar day near midnight → Keep the implementation deterministic and document that transactions use UTC until a future date option is added.
- Integer cents assume two decimal places → Validate input and reject values with more than two fractional digits.
- Using `desc` as a column name may be awkward because `DESC` is a SQL keyword → Quote the column as `"desc"` in SQL statements.
- Auto-creating the table on command execution hides schema setup failures until runtime → Run table initialization before operations and surface errors directly.
