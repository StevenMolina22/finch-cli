package finch

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tursodatabase/libsql-client-go/libsql"
)

type Store struct {
	db *sql.DB
}

func OpenStoreFromEnv(ctx context.Context) (*Store, error) {
	dbURL := strings.TrimSpace(os.Getenv("FINCH_DB_URL"))
	if dbURL == "" {
		return nil, fmt.Errorf("FINCH_DB_URL is required")
	}
	token := strings.TrimSpace(os.Getenv("FINCH_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("FINCH_TOKEN is required")
	}

	connector, err := libsql.NewConnector(dbURL, libsql.WithAuthToken(token))
	if err != nil {
		return nil, fmt.Errorf("create libsql connector: %w", err)
	}
	return OpenStore(ctx, connector)
}

func OpenStore(ctx context.Context, connector driver.Connector) (*Store, error) {
	db := sql.OpenDB(connector)
	store := &Store{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Add(ctx context.Context, input AddInput) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO transactions (type, amount, category, "desc", date, tags, recurring)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, input.Type, input.AmountCents, input.Category, input.Desc, input.Date, input.Tags, nullableRecurring(input.Recurring))
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context, filter ListFilter) ([]Transaction, error) {
	query := `SELECT id, type, amount, category, "desc", date, tags, recurring FROM transactions`
	args := make([]any, 0, 3)
	where := make([]string, 0, 3)
	if filter.Month != "" {
		where = append(where, "substr(date, 1, 7) = ?")
		args = append(args, filter.Month)
	}
	if filter.Category != "" {
		where = append(where, "category = ?")
		args = append(args, filter.Category)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY date DESC, id DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		var recurring sql.NullString
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.AmountCents, &tx.Category, &tx.Desc, &tx.Date, &tx.Tags, &recurring); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		tx.Recurring = recurring.String
		tx.Amount = FormatAmount(tx.AmountCents)
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read transactions: %w", err)
	}
	return transactions, nil
}

func (s *Store) Summary(ctx context.Context, month string) (Summary, error) {
	query := `
        SELECT
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
            COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
        FROM transactions
    `
	args := []any{}
	if month != "" {
		query += " WHERE substr(date, 1, 7) = ?"
		args = append(args, month)
	}

	var incomeCents int64
	var expenseCents int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&incomeCents, &expenseCents); err != nil {
		return Summary{}, fmt.Errorf("summarize transactions: %w", err)
	}

	topCategories, err := s.topExpenseCategories(ctx, month)
	if err != nil {
		return Summary{}, err
	}

	return NewSummary(month, incomeCents, expenseCents, topCategories), nil
}

func (s *Store) topExpenseCategories(ctx context.Context, month string) ([]TopCategory, error) {
	query := `
        SELECT category, SUM(amount) as total
        FROM transactions
        WHERE type = 'expense'
    `
	args := []any{}
	if month != "" {
		query += " AND substr(date, 1, 7) = ?"
		args = append(args, month)
	}
	query += " GROUP BY category ORDER BY total DESC LIMIT 3"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query top categories: %w", err)
	}
	defer rows.Close()

	categories := []TopCategory{}
	for rows.Next() {
		var tc TopCategory
		var cents int64
		if err := rows.Scan(&tc.Category, &cents); err != nil {
			return nil, fmt.Errorf("scan top category: %w", err)
		}
		tc.Amount = FormatAmount(cents)
		categories = append(categories, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read top categories: %w", err)
	}
	return categories, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("transaction %d: %w", id, ErrTransactionNotFound)
	}
	return nil
}

func (s *Store) Update(ctx context.Context, input EditInput) error {
	query := "UPDATE transactions SET "
	updates := []string{}
	args := []any{}

	if input.AmountCents != nil {
		updates = append(updates, "amount = ?")
		args = append(args, *input.AmountCents)
	}
	if input.Category != nil {
		updates = append(updates, "category = ?")
		args = append(args, *input.Category)
	}
	if input.Desc != nil {
		updates = append(updates, `"desc" = ?`)
		args = append(args, *input.Desc)
	}
	if input.Tags != nil {
		updates = append(updates, "tags = ?")
		args = append(args, *input.Tags)
	}
	if input.Recurring != nil {
		updates = append(updates, "recurring = ?")
		args = append(args, nullableRecurring(*input.Recurring))
	}

	query += strings.Join(updates, ", ")
	query += " WHERE id = ?"
	args = append(args, input.ID)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update transaction: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("transaction %d: %w", input.ID, ErrTransactionNotFound)
	}
	return nil
}

func (s *Store) Export(ctx context.Context, filter ExportFilter) ([]Transaction, error) {
	query := `SELECT id, type, amount, category, "desc", date, tags, recurring FROM transactions`
	args := []any{}
	if filter.Month != "" {
		query += " WHERE substr(date, 1, 7) = ?"
		args = append(args, filter.Month)
	}
	query += " ORDER BY date ASC, id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("export transactions: %w", err)
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		var recurring sql.NullString
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.AmountCents, &tx.Category, &tx.Desc, &tx.Date, &tx.Tags, &recurring); err != nil {
			return nil, fmt.Errorf("scan export transaction: %w", err)
		}
		tx.Recurring = recurring.String
		tx.Amount = FormatAmount(tx.AmountCents)
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func (s *Store) Import(ctx context.Context, rows []ImportRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO transactions (type, amount, category, "desc", date, tags, recurring)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("prepare import statement: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		amountCents, err := ParseAmount(row.Amount)
		if err != nil {
			return fmt.Errorf("parse amount in import: %w", err)
		}
		date := row.Date
		if date == "" {
			date = time.Now().UTC().Format("2006-01-02")
		}
		if _, err := stmt.ExecContext(ctx, row.Type, amountCents, row.Category, row.Desc, date, row.Tags, nullableRecurring(row.Recurring)); err != nil {
			return fmt.Errorf("insert import row: %w", err)
		}
	}

	return tx.Commit()
}

func nullableRecurring(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *Store) ensureSchema(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS transactions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
            amount INTEGER NOT NULL,
            category TEXT NOT NULL,
            "desc" TEXT NOT NULL DEFAULT '',
            date TEXT NOT NULL,
            tags TEXT NOT NULL DEFAULT '',
            recurring TEXT
        )
    `)
	if err != nil {
		return fmt.Errorf("initialize transactions table: %w", err)
	}

	// Schema upgrade: add tags column if missing (pre-metadata schema)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE transactions ADD COLUMN tags TEXT NOT NULL DEFAULT ''`)

	// Schema upgrade: add recurring column if missing
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE transactions ADD COLUMN recurring TEXT`)

	return nil
}
