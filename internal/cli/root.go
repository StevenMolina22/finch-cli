package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"finch/internal/finch"
	finchmcp "finch/internal/mcp"
	"finch/internal/server"

	"github.com/spf13/cobra"
)

type Store interface {
	Add(context.Context, finch.AddInput) error
	List(context.Context, finch.ListFilter) ([]finch.Transaction, error)
	Summary(context.Context, string) (finch.Summary, error)
	Delete(context.Context, int64) error
	Update(context.Context, finch.EditInput) error
	Export(context.Context, finch.ExportFilter) ([]finch.Transaction, error)
	Import(context.Context, []finch.ImportRow) error
	Close() error
}

type OpenStoreFunc func(context.Context) (Store, error)

// MCPRunFunc is the entry point invoked by the mcp command. It is a
// separate type so tests can swap in a no-op that captures the resolved
// options without binding to a real port.
type MCPRunFunc func(ctx context.Context, transport finchmcp.Transport, opts finchmcp.Options) error

// defaultMCPRun is the production MCPRunFunc that delegates to
// finchmcp.Run.
func defaultMCPRun(ctx context.Context, transport finchmcp.Transport, opts finchmcp.Options) error {
	return finchmcp.Run(ctx, transport, opts)
}

func NewRootCommand(openStore OpenStoreFunc, now func() time.Time) *cobra.Command {
	return newRootCommand(openStore, now, server.Listen, defaultMCPRun)
}

// newRootCommand is the internal constructor used by tests to inject a
// custom listen function for the serve command and a custom MCP run
// function for the mcp command.
func newRootCommand(openStore OpenStoreFunc, now func() time.Time, listen ListenFunc, mcpRun MCPRunFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "finch",
		Short:         "Minimal personal finance tracking",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newAddCommand(openStore, now))
	cmd.AddCommand(newListCommand(openStore))
	cmd.AddCommand(newSummaryCommand(openStore))
	cmd.AddCommand(newDeleteCommand(openStore))
	cmd.AddCommand(newEditCommand(openStore, now))
	cmd.AddCommand(newExportCommand(openStore))
	cmd.AddCommand(newImportCommand(openStore))
	cmd.AddCommand(newServeCommand(openStore, now, listen))
	cmd.AddCommand(newMCPCommand(openStore, mcpRun))
	return cmd
}

func newAddCommand(openStore OpenStoreFunc, now func() time.Time) *cobra.Command {
	var input finch.AddInput
	cmd := &cobra.Command{
		Use:   "add [income|expense] <amount> <category> [description]",
		Short: "Add an income or expense transaction",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 && len(args) != 4 {
				return fmt.Errorf("accepts 3 or 4 args, received %d", len(args))
			}
			if err := finch.ValidateType(args[0]); err != nil {
				return err
			}
			amountCents, err := finch.ParseAmount(args[1])
			if err != nil {
				return err
			}
			category := strings.TrimSpace(args[2])
			if category == "" {
				return fmt.Errorf("category is required")
			}
			desc := ""
			if len(args) == 4 {
				desc = strings.TrimSpace(args[3])
			}
			input.Type = args[0]
			input.AmountCents = amountCents
			input.Category = category
			input.Desc = desc
			input.Date = now().UTC().Format("2006-01-02")
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return finch.ValidateRecurring(input.Recurring)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.Add(cmd.Context(), input); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s %s %s on %s\n", input.Type, finch.FormatAmount(input.AmountCents), input.Category, input.Date)
			return nil
		},
	}
	cmd.Flags().StringVar(&input.Tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&input.Recurring, "recurring", "", "recurrence (monthly, weekly, yearly)")
	return cmd
}

func newListCommand(openStore OpenStoreFunc) *cobra.Command {
	var month string
	var category string
	var limit int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := finch.ValidateMonth(month); err != nil {
				return err
			}
			category = strings.TrimSpace(category)
			if cmd.Flags().Changed("limit") {
				if err := finch.ValidateLimit(limit); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			transactions, err := store.List(cmd.Context(), finch.ListFilter{Month: month, Category: category, Limit: limit})
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(cmd.OutOrStdout(), transactions)
			}
			writeTransactions(cmd.OutOrStdout(), transactions)
			return nil
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "filter by month (YYYY-MM)")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit number of transactions")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON")
	return cmd
}

func newSummaryCommand(openStore OpenStoreFunc) *cobra.Command {
	var month string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize transactions",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return finch.ValidateMonth(month)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			summary, err := store.Summary(cmd.Context(), month)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(cmd.OutOrStdout(), summary)
			}
			writeSummary(cmd.OutOrStdout(), summary)
			return nil
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "filter by month (YYYY-MM)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON")
	return cmd
}

func newDeleteCommand(openStore OpenStoreFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a transaction",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg, received %d", len(args))
			}
			if _, err := parseID(args[0]); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if err := store.Delete(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted transaction %d\n", id)
			return nil
		},
	}
	return cmd
}

func newEditCommand(openStore OpenStoreFunc, now func() time.Time) *cobra.Command {
	var amountStr string
	var category string
	var desc string
	var tags string
	var recurring string
	cmd := &cobra.Command{
		Use:   "edit <id> [--amount] [--category] [--desc] [--tags] [--recurring]",
		Short: "Edit a transaction",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg (id), received %d", len(args))
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			changed := cmd.Flags().Changed("amount") || cmd.Flags().Changed("category") ||
				cmd.Flags().Changed("desc") || cmd.Flags().Changed("tags") || cmd.Flags().Changed("recurring")
			if !changed {
				return fmt.Errorf("at least one field must be changed")
			}
			if cmd.Flags().Changed("recurring") {
				if err := finch.ValidateRecurring(recurring); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			id, err := parseID(args[0])
			if err != nil {
				return err
			}

			input := finch.EditInput{ID: id}
			if cmd.Flags().Changed("amount") {
				cents, err := finch.ParseAmount(amountStr)
				if err != nil {
					return err
				}
				input.AmountCents = &cents
			}
			if cmd.Flags().Changed("category") {
				v := strings.TrimSpace(category)
				if v == "" {
					return fmt.Errorf("category is required")
				}
				input.Category = &v
			}
			if cmd.Flags().Changed("desc") {
				v := strings.TrimSpace(desc)
				input.Desc = &v
			}
			if cmd.Flags().Changed("tags") {
				v := strings.TrimSpace(tags)
				input.Tags = &v
			}
			if cmd.Flags().Changed("recurring") {
				v := strings.TrimSpace(recurring)
				input.Recurring = &v
			}

			if err := store.Update(cmd.Context(), input); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated transaction %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&amountStr, "amount", "", "new amount")
	cmd.Flags().StringVar(&category, "category", "", "new category")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().StringVar(&tags, "tags", "", "new tags")
	cmd.Flags().StringVar(&recurring, "recurring", "", "new recurrence (monthly, weekly, yearly)")
	return cmd
}

func newExportCommand(openStore OpenStoreFunc) *cobra.Command {
	var month string
	var csvFlag bool
	cmd := &cobra.Command{
		Use:   "export [--csv] [--month YYYY-MM]",
		Short: "Export transactions as CSV",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return finch.ValidateMonth(month)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			transactions, err := store.Export(cmd.Context(), finch.ExportFilter{Month: month})
			if err != nil {
				return err
			}
			writeCSV(cmd.OutOrStdout(), transactions)
			return nil
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "filter by month (YYYY-MM)")
	cmd.Flags().BoolVar(&csvFlag, "csv", false, "output CSV (default)")
	return cmd
}

func newImportCommand(openStore OpenStoreFunc) *cobra.Command {
	var csvPath string
	cmd := &cobra.Command{
		Use:   "import --csv <file>",
		Short: "Import transactions from CSV",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("csv") {
				return fmt.Errorf("--csv <file> is required")
			}

			importRows, err := readCSVFile(csvPath)
			if err != nil {
				return err
			}

			if err := finch.ValidateImportRows(importRows); err != nil {
				return err
			}

			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.Import(cmd.Context(), importRows); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d transactions\n", len(importRows))
			return nil
		},
	}
	cmd.Flags().StringVar(&csvPath, "csv", "", "path to CSV file")
	return cmd
}

func writeTransactions(w io.Writer, transactions []finch.Transaction) {
	if len(transactions) == 0 {
		fmt.Fprintln(w, "No transactions found.")
		return
	}
	fmt.Fprintf(w, "%-4s %-12s %-8s %12s %-16s %-20s %-10s %s\n", "ID", "DATE", "TYPE", "AMOUNT", "CATEGORY", "DESCRIPTION", "TAGS", "RECURRING")
	for _, tx := range transactions {
		tags := tx.Tags
		if tags == "" {
			tags = "-"
		}
		rec := tx.Recurring
		if rec == "" {
			rec = "-"
		}
		fmt.Fprintf(w, "%-4d %-12s %-8s %12s %-16s %-20s %-10s %s\n", tx.ID, tx.Date, tx.Type, tx.Amount, tx.Category, tx.Desc, tags, rec)
	}
}

func writeSummary(w io.Writer, summary finch.Summary) {
	if summary.Month != "" {
		fmt.Fprintf(w, "Summary for %s\n", summary.Month)
	} else {
		fmt.Fprintln(w, "Summary")
	}
	fmt.Fprintf(w, "Income:  %s\n", summary.Income)
	fmt.Fprintf(w, "Expense: %s\n", summary.Expense)
	fmt.Fprintf(w, "Net:     %s\n", summary.Net)
	if len(summary.TopCategories) > 0 {
		fmt.Fprintln(w, "Top expense categories:")
		for _, tc := range summary.TopCategories {
			fmt.Fprintf(w, "  %s: %s\n", tc.Category, tc.Amount)
		}
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeCSV(w io.Writer, transactions []finch.Transaction) {
	fmt.Fprintln(w, "id,type,amount,category,desc,date,tags,recurring")
	for _, tx := range transactions {
		recurring := tx.Recurring
		if recurring == "" {
			recurring = `\N`
		}
		fmt.Fprintf(w, "%d,%s,%s,%s,%s,%s,%s,%s\n",
			tx.ID, tx.Type, tx.Amount, csvEscape(tx.Category), csvEscape(tx.Desc), tx.Date, csvEscape(tx.Tags), recurring)
	}
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func readCSVFile(path string) ([]finch.ImportRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CSV file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("CSV file must have a header and at least one data row")
	}

	header := strings.Split(lines[0], ",")
	colMap := make(map[string]int)
	for i, h := range header {
		colMap[strings.TrimSpace(h)] = i
	}

	required := []string{"type", "amount", "category"}
	for _, r := range required {
		if _, ok := colMap[r]; !ok {
			return nil, fmt.Errorf("CSV missing required column: %s", r)
		}
	}

	rows := make([]finch.ImportRow, 0, len(lines)-1)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		row := finch.ImportRow{
			Type:      getCSVField(fields, colMap, "type"),
			Amount:    getCSVField(fields, colMap, "amount"),
			Category:  getCSVField(fields, colMap, "category"),
			Desc:      getCSVField(fields, colMap, "desc"),
			Date:      getCSVField(fields, colMap, "date"),
			Tags:      getCSVField(fields, colMap, "tags"),
			Recurring: getCSVField(fields, colMap, "recurring"),
		}
		if row.Recurring == `\N` {
			row.Recurring = ""
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func getCSVField(fields []string, colMap map[string]int, name string) string {
	idx, ok := colMap[name]
	if !ok || idx >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[idx])
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id: %q", s)
	}
	return id, nil
}

// ListenFunc is the listen function used by the serve command. It is
// declared here (instead of inline inside the command) so tests can swap
// in a no-op or a controlled listener without binding to a real port.
type ListenFunc = server.ListenFunc

func newServeCommand(openStore OpenStoreFunc, now func() time.Time, listen ListenFunc) *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		Long: "Start the Finch HTTP API server on the given address.\n\n" +
			"The first version of the HTTP API is unauthenticated. " +
			"Bind to localhost, a private interface, or protect it with network controls.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			app := server.NewApp(storeAdapter{store: store}, now)
			return listen(app, addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":3000", "address to listen on (e.g. 127.0.0.1:3000)")
	return cmd
}

// storeAdapter exposes a Store as a server.Store, satisfying the smaller
// interface the HTTP handlers require.
type storeAdapter struct {
	store Store
}

func (a storeAdapter) Add(ctx context.Context, input finch.AddInput) error {
	return a.store.Add(ctx, input)
}

func (a storeAdapter) List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error) {
	return a.store.List(ctx, filter)
}

func (a storeAdapter) Summary(ctx context.Context, month string) (finch.Summary, error) {
	return a.store.Summary(ctx, month)
}

func (a storeAdapter) Update(ctx context.Context, input finch.EditInput) error {
	return a.store.Update(ctx, input)
}

func (a storeAdapter) Delete(ctx context.Context, id int64) error {
	return a.store.Delete(ctx, id)
}

// mcpStoreAdapter exposes a Store as a finchmcp.Store, satisfying the
// smaller interface the MCP tool handlers require.
type mcpStoreAdapter struct {
	store Store
}

func (a mcpStoreAdapter) Add(ctx context.Context, input finch.AddInput) error {
	return a.store.Add(ctx, input)
}

func (a mcpStoreAdapter) List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error) {
	return a.store.List(ctx, filter)
}

func (a mcpStoreAdapter) Summary(ctx context.Context, month string) (finch.Summary, error) {
	return a.store.Summary(ctx, month)
}

func (a mcpStoreAdapter) Update(ctx context.Context, input finch.EditInput) error {
	return a.store.Update(ctx, input)
}

func (a mcpStoreAdapter) Delete(ctx context.Context, id int64) error {
	return a.store.Delete(ctx, id)
}

// loadMCPAuthConfig reads the bearer API key used by the HTTP MCP transport.
// The returned config may be empty; callers must reject HTTP startup when no
// key is configured.
func loadMCPAuthConfig() finchmcp.AuthConfig {
	return finchmcp.AuthConfig{
		APIKey: strings.TrimSpace(os.Getenv("FINCH_API_KEY")),
	}
}

func newMCPCommand(openStore OpenStoreFunc, mcpRun MCPRunFunc) *cobra.Command {
	var transport string
	var addr string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP (Model Context Protocol) server",
		Long: "Start Finch as an MCP server. The default transport is HTTP on :3333, " +
			"intended for remote AI clients. Use --transport http --addr to change the listener.\n\n" +
			"Remote HTTP MCP requires HTTPS in deployment (use a reverse proxy or a hosting " +
			"platform that terminates TLS). HTTP startup refuses to bind when no API key is " +
			"configured via FINCH_API_KEY. The API key can call all read and write tools.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var mcpTransport finchmcp.Transport
			switch strings.ToLower(strings.TrimSpace(transport)) {
			case string(finchmcp.TransportHTTP), "":
				mcpTransport = finchmcp.TransportHTTP
			case string(finchmcp.TransportStdio):
				mcpTransport = finchmcp.TransportStdio
			default:
				return fmt.Errorf("%w: %q (supported: %s, %s)",
					finchmcp.ErrUnsupportedTransport, transport,
					finchmcp.TransportHTTP, finchmcp.TransportStdio)
			}

			if mcpTransport == finchmcp.TransportHTTP {
				auth := loadMCPAuthConfig()
				if auth.IsEmpty() {
					return errors.New("HTTP MCP transport requires FINCH_API_KEY")
				}
			}

			store, err := openStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			return mcpRun(cmd.Context(), mcpTransport, finchmcp.Options{
				Store: mcpStoreAdapter{store: store},
				Auth:  loadMCPAuthConfig(),
				Addr:  addr,
			})
		},
	}

	cmd.Flags().StringVar(&transport, "transport", string(finchmcp.TransportHTTP), "MCP transport: http or stdio")
	cmd.Flags().StringVar(&addr, "addr", ":3333", "address to listen on for HTTP transport (e.g. 127.0.0.1:3333)")
	return cmd
}
