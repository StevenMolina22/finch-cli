## 1. Project Setup

- [x] 1.1 Add Cobra and `libsql-client-go` dependencies to the Go module.
- [x] 1.2 Create the CLI entrypoint and root Cobra command for `finch`.
- [x] 1.3 Add command registration for `add`, `list`, and `summary`.
- [x] 1.4 Add command registration for `delete`, `edit`, `export`, and `import`.

## 2. Transaction Model and Storage

- [x] 2.1 Define transaction and summary structs with JSON fields matching the specs.
- [x] 2.2 Implement environment loading and validation for `FINCH_DB_URL` and `FINCH_TOKEN`.
- [x] 2.3 Implement direct Turso client creation using `libsql-client-go`.
- [x] 2.4 Implement `transactions` table initialization with columns `id`, `type`, `amount`, `category`, `desc`, and `date`.
- [x] 2.5 Implement SQL functions to insert, list, and summarize transactions without an ORM.
- [x] 2.6 Add `tags TEXT` and nullable `recurring TEXT` columns to `transactions` initialization and schema upgrade logic.
- [x] 2.7 Extend transaction structs and scan/serialize code to include `tags` and `recurring`.
- [x] 2.8 Implement SQL functions to update and delete transactions by id.
- [x] 2.9 Implement SQL functions to export filtered transactions and import CSV rows atomically.
- [x] 2.10 Extend summary queries to include top 3 expense categories.

## 3. Input Parsing and Validation

- [x] 3.1 Implement amount parsing that accepts positive currency values and stores integer cents.
- [x] 3.2 Implement amount formatting from integer cents to two-decimal strings.
- [x] 3.3 Validate transaction type, amount, category, and optional description arguments for `finch add`.
- [x] 3.4 Validate `--month` values in `YYYY-MM` format for read commands.
- [x] 3.5 Validate `--limit` as a positive integer for `finch list`.
- [x] 3.6 Validate `--recurring` values as `monthly`, `weekly`, `yearly`, or nullable.
- [x] 3.7 Validate edit commands require at least one changed field.
- [x] 3.8 Validate CSV import rows using transaction input rules before persisting any rows.

## 4. Command Behavior

- [x] 4.1 Implement `finch add [income|expense] <amount> <category> [description]` with automatic current UTC date.
- [x] 4.2 Implement `finch list` with optional `--month`, `--category`, and default human-readable output.
- [x] 4.3 Implement `finch list --json` as a JSON array of transaction objects.
- [x] 4.4 Implement `finch summary` with optional `--month` and default human-readable totals.
- [x] 4.5 Implement `finch summary --json` as a JSON object containing `month`, `income`, `expense`, and `net`.
- [x] 4.6 Implement `finch list --limit N` and include `tags` and `recurring` in list output.
- [x] 4.7 Update `finch summary` output to show income, expenses, net balance, and top 3 categories.
- [x] 4.8 Update `finch summary --json` to include income, expense, net, and top categories.
- [x] 4.9 Implement `finch delete <id>`.
- [x] 4.10 Implement `finch edit <id> [--amount] [--category] [--desc] [--tags] [--recurring]`.
- [x] 4.11 Implement `finch export [--csv] [--month YYYY-MM]`.
- [x] 4.12 Implement `finch import --csv <file>`.

## 5. Verification

- [x] 5.1 Add unit tests for amount parsing, amount formatting, and month validation.
- [x] 5.2 Add command-level tests for validation failures that do not require a live Turso database.
- [x] 5.3 Run `go test ./...` and fix any failures.
- [x] 5.4 Run `go build ./...` to verify the CLI builds.
- [x] 5.5 Add tests for recurring validation, list limits, edit validation, and CSV row validation.
- [x] 5.6 Add command-level tests for delete, edit, export, and import usage/validation failures.
- [x] 5.7 Re-run `go test ./...` after implementing the added scope.
- [x] 5.8 Re-run `go build ./...` after implementing the added scope.
