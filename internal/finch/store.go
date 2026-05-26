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
		INSERT INTO transactions (type, amount, category, "desc", date)
		VALUES (?, ?, ?, ?, ?)
	`, input.Type, input.AmountCents, input.Category, input.Desc, input.Date)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context, filter ListFilter) ([]Transaction, error) {
	query := `SELECT id, type, amount, category, "desc", date FROM transactions`
	args := make([]any, 0, 2)
	where := make([]string, 0, 2)
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

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.AmountCents, &tx.Category, &tx.Desc, &tx.Date); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
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
	return NewSummary(month, incomeCents, expenseCents), nil
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
			date TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("initialize transactions table: %w", err)
	}
	return nil
}
