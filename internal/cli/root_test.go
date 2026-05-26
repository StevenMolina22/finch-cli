package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"finch/internal/finch"

	"github.com/spf13/cobra"
)

type fakeStore struct{}

func (fakeStore) Add(context.Context, finch.AddInput) error { return nil }
func (fakeStore) List(context.Context, finch.ListFilter) ([]finch.Transaction, error) {
	return nil, nil
}
func (fakeStore) Summary(context.Context, string) (finch.Summary, error) {
	return finch.NewSummary("", 0, 0), nil
}
func (fakeStore) Close() error { return nil }

func TestInvalidAddArgsDoNotOpenStore(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "invalid type", args: []string{"add", "transfer", "10.00", "savings"}},
		{name: "invalid amount", args: []string{"add", "expense", "0", "groceries"}},
		{name: "too many fractional digits", args: []string{"add", "expense", "1.999", "groceries"}},
		{name: "empty category", args: []string{"add", "expense", "1.99", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			cmd := NewRootCommand(func(context.Context) (Store, error) {
				called = true
				return fakeStore{}, nil
			}, func() time.Time { return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) })
			err := execute(cmd, tt.args...)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if called {
				t.Fatal("store opened for invalid add args")
			}
		})
	}
}

func TestInvalidReadMonthDoesNotOpenStore(t *testing.T) {
	tests := [][]string{
		{"list", "--month", "May-2026"},
		{"summary", "--month", "2026-13"},
	}

	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			called := false
			cmd := NewRootCommand(func(context.Context) (Store, error) {
				called = true
				return fakeStore{}, nil
			}, time.Now)
			err := execute(cmd, args...)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if called {
				t.Fatal("store opened for invalid month")
			}
		})
	}
}

func TestValidAddOpensStore(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, func() time.Time { return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) })

	if err := execute(cmd, "add", "income", "10.00", "salary"); err != nil {
		t.Fatalf("execute add error = %v", err)
	}
	if !called {
		t.Fatal("store was not opened for valid add")
	}
}

func TestStoreOpenErrorReturnsError(t *testing.T) {
	want := errors.New("open failed")
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		return nil, want
	}, time.Now)

	err := execute(cmd, "list")
	if !errors.Is(err, want) {
		t.Fatalf("execute error = %v, want %v", err, want)
	}
}

func execute(cmd *cobra.Command, args ...string) error {
	var out bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	return cmd.Execute()
}
