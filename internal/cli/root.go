package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"finch/internal/finch"

	"github.com/spf13/cobra"
)

type Store interface {
	Add(context.Context, finch.AddInput) error
	List(context.Context, finch.ListFilter) ([]finch.Transaction, error)
	Summary(context.Context, string) (finch.Summary, error)
	Close() error
}

type OpenStoreFunc func(context.Context) (Store, error)

func NewRootCommand(openStore OpenStoreFunc, now func() time.Time) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "finch",
		Short:         "Minimal personal finance tracking",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newAddCommand(openStore, now))
	cmd.AddCommand(newListCommand(openStore))
	cmd.AddCommand(newSummaryCommand(openStore))
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
			input = finch.AddInput{
				Type:        args[0],
				AmountCents: amountCents,
				Category:    category,
				Desc:        desc,
				Date:        now().UTC().Format("2006-01-02"),
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.Add(ctx, input); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s %s %s on %s\n", input.Type, finch.FormatAmount(input.AmountCents), input.Category, input.Date)
			return nil
		},
	}
	return cmd
}

func newListCommand(openStore OpenStoreFunc) *cobra.Command {
	var month string
	var category string
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
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer store.Close()

			transactions, err := store.List(ctx, finch.ListFilter{Month: month, Category: category})
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
			ctx := cmd.Context()
			store, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer store.Close()

			summary, err := store.Summary(ctx, month)
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

func writeTransactions(w io.Writer, transactions []finch.Transaction) {
	if len(transactions) == 0 {
		fmt.Fprintln(w, "No transactions found.")
		return
	}
	fmt.Fprintf(w, "%-12s %-8s %12s %-16s %s\n", "DATE", "TYPE", "AMOUNT", "CATEGORY", "DESCRIPTION")
	for _, tx := range transactions {
		fmt.Fprintf(w, "%-12s %-8s %12s %-16s %s\n", tx.Date, tx.Type, tx.Amount, tx.Category, tx.Desc)
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
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
