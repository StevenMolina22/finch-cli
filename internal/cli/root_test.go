package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"finch/internal/finch"
	finchmcp "finch/internal/mcp"
	"finch/internal/server"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cobra"
)

type fakeStore struct{}

func (fakeStore) Add(context.Context, finch.AddInput) error { return nil }
func (fakeStore) List(context.Context, finch.ListFilter) ([]finch.Transaction, error) {
	return nil, nil
}
func (fakeStore) Summary(context.Context, string) (finch.Summary, error) {
	return finch.NewSummary("", 0, 0, nil), nil
}
func (fakeStore) Delete(context.Context, int64) error           { return nil }
func (fakeStore) Update(context.Context, finch.EditInput) error { return nil }
func (fakeStore) Export(context.Context, finch.ExportFilter) ([]finch.Transaction, error) {
	return nil, nil
}
func (fakeStore) Import(context.Context, []finch.ImportRow) error { return nil }
func (fakeStore) Close() error                                    { return nil }

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

func TestDeleteCommandValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no id", args: []string{"delete"}},
		{name: "negative id", args: []string{"delete", "-1"}},
		{name: "non-numeric id", args: []string{"delete", "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			cmd := NewRootCommand(func(context.Context) (Store, error) {
				called = true
				return fakeStore{}, nil
			}, time.Now)
			err := execute(cmd, tt.args...)
			if err == nil {
				t.Fatal("expected error")
			}
			if called {
				t.Fatal("store opened for invalid delete args")
			}
		})
	}
}

func TestEditCommandRequiresAtLeastOneFlag(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "edit", "1")
	if err == nil {
		t.Fatal("expected error for edit with no flags")
	}
	if called {
		t.Fatal("store opened for edit with no flags")
	}
}

func TestEditCommandValidatesRecurring(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "edit", "1", "--recurring", "daily")
	if err == nil {
		t.Fatal("expected error for invalid recurring")
	}
	if called {
		t.Fatal("store opened for invalid recurring")
	}
}

func TestExportCommandValidatesMonth(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "export", "--month", "May-2026")
	if err == nil {
		t.Fatal("expected error for invalid month")
	}
	if called {
		t.Fatal("store opened for invalid month")
	}
}

func TestImportCommandRequiresCSVFlag(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "import")
	if err == nil {
		t.Fatal("expected error for import without --csv")
	}
	if called {
		t.Fatal("store opened for import without --csv")
	}
}

func TestListLimitValidation(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "list", "--limit", "0")
	if err == nil {
		t.Fatal("expected error for limit 0")
	}
	if called {
		t.Fatal("store opened for invalid limit")
	}
}

func TestAddCommandValidatesRecurring(t *testing.T) {
	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, func() time.Time { return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) })
	err := execute(cmd, "add", "income", "10.00", "salary", "--recurring", "daily")
	if err == nil {
		t.Fatal("expected error for invalid recurring")
	}
	if called {
		t.Fatal("store opened for invalid recurring")
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

func TestServeCommandOpensStoreAndUsesInjectedListen(t *testing.T) {
	called := 0
	var capturedAddr string
	var capturedApp *fiber.App

	listen := func(app *fiber.App, addr string) error {
		called++
		capturedApp = app
		capturedAddr = addr
		return nil
	}

	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, listen, defaultMCPRun)

	if err := execute(cmd, "serve", "--addr", "127.0.0.1:8080"); err != nil {
		t.Fatalf("execute serve error = %v", err)
	}
	if called != 1 {
		t.Fatalf("listen called %d times, want 1", called)
	}
	if capturedAddr != "127.0.0.1:8080" {
		t.Fatalf("addr = %q, want 127.0.0.1:8080", capturedAddr)
	}
	if capturedApp == nil {
		t.Fatal("listen did not receive a fiber app")
	}
}

func TestServeCommandDefaultAddress(t *testing.T) {
	var capturedAddr string
	listen := func(app *fiber.App, addr string) error {
		capturedAddr = addr
		return nil
	}
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, listen, defaultMCPRun)

	if err := execute(cmd, "serve"); err != nil {
		t.Fatalf("execute serve error = %v", err)
	}
	if capturedAddr != ":3000" {
		t.Fatalf("addr = %q, want :3000", capturedAddr)
	}
}

func TestServeCommandStoreOpenFailure(t *testing.T) {
	want := errors.New("open failed")
	listen := func(app *fiber.App, addr string) error {
		t.Fatal("listen should not be called when store fails to open")
		return nil
	}
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return nil, want
	}, time.Now, listen, defaultMCPRun)

	err := execute(cmd, "serve")
	if !errors.Is(err, want) {
		t.Fatalf("execute error = %v, want %v", err, want)
	}
}

func TestServeCommandHelpMentionsUnauthenticated(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, func(app *fiber.App, addr string) error { return nil }, defaultMCPRun)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"serve", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("serve --help error = %v", err)
	}
	got := out.String()
	if !strings.Contains(strings.ToLower(got), "unauthenticated") {
		t.Fatalf("serve --help missing unauthenticated guidance:\n%s", got)
	}
}

func TestServeCommandWiredToServerStoreAdapter(t *testing.T) {
	var capturedApp *fiber.App
	listen := func(app *fiber.App, addr string) error {
		capturedApp = app
		return nil
	}
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, listen, defaultMCPRun)

	if err := execute(cmd, "serve"); err != nil {
		t.Fatalf("execute serve error = %v", err)
	}

	req := httpRequest(t, "GET", "/health", nil)
	resp, err := capturedApp.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", resp.StatusCode)
	}
}

func httpRequest(t *testing.T, method, path string, body *bytes.Buffer) *http.Request {
	t.Helper()
	var reader *bytes.Buffer
	if body != nil {
		reader = body
	}
	var r io.Reader
	if reader != nil {
		r = reader
	}
	req, err := http.NewRequest(method, path, r)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	return req
}

// Ensure server.Listen is still the production listen function so the
// default NewRootCommand callers exercise the real listener wiring.
var _ = server.Listen

func TestMCPCommandHTTPWithoutAuthFailsFast(t *testing.T) {
	t.Setenv("FINCH_API_KEY", "")

	called := false
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		called = true
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "mcp", "--transport", "http")
	if err == nil {
		t.Fatal("expected error when HTTP transport has no API key")
	}
	if !strings.Contains(err.Error(), "FINCH_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("store should not be opened when API key is missing")
	}
}

func TestMCPCommandHTTPDefaultsToAddrColon3333(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"mcp", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mcp --help error = %v", err)
	}
	if !strings.Contains(out.String(), ":3333") {
		t.Fatalf("mcp --help should document default addr :3333, got:\n%s", out.String())
	}
}

func TestMCPCommandHonorsAddrFlag(t *testing.T) {
	t.Setenv("FINCH_API_KEY", "secret")

	captured := captureMCPAddr(t, []string{"mcp", "--addr", "127.0.0.1:4444"})
	if captured != "127.0.0.1:4444" {
		t.Fatalf("captured addr = %q, want 127.0.0.1:4444", captured)
	}
}

func TestMCPCommandAddrFlagDefault(t *testing.T) {
	t.Setenv("FINCH_API_KEY", "secret")

	captured := captureMCPAddr(t, []string{"mcp"})
	if captured != ":3333" {
		t.Fatalf("captured addr = %q, want :3333", captured)
	}
}

func TestMCPCommandRejectsUnsupportedTransport(t *testing.T) {
	t.Setenv("FINCH_API_KEY", "secret")

	cmd := NewRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now)
	err := execute(cmd, "mcp", "--transport", "websocket")
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
	if !strings.Contains(err.Error(), "unsupported MCP transport") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMCPCommandHelpMentionsHTTPSAndAPIKey(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"mcp", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mcp --help error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "HTTPS") {
		t.Fatalf("mcp --help should mention HTTPS, got:\n%s", got)
	}
	if !strings.Contains(got, "FINCH_API_KEY") {
		t.Fatalf("mcp --help should mention FINCH_API_KEY, got:\n%s", got)
	}
}

func TestMCPCommandWithAPIKey(t *testing.T) {
	t.Setenv("FINCH_API_KEY", "secret")

	called := false
	mcpRun := func(_ context.Context, _ finchmcp.Transport, _ finchmcp.Options) error {
		called = true
		return nil
	}
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, func(_ *fiber.App, _ string) error { return nil }, mcpRun)
	if err := execute(cmd, "mcp"); err != nil {
		t.Fatalf("execute mcp error = %v", err)
	}
	if !called {
		t.Fatal("mcp command should invoke MCPRunFunc when API key is configured")
	}
}

// captureMCPAddr runs the mcp command with the supplied args using a
// captured MCPRunFunc, and returns the addr value that would have been
// passed to the listener.
func captureMCPAddr(t *testing.T, args []string) string {
	t.Helper()
	var captured string
	mcpRun := func(_ context.Context, _ finchmcp.Transport, opts finchmcp.Options) error {
		captured = opts.Addr
		return nil
	}
	cmd := newRootCommand(func(context.Context) (Store, error) {
		return fakeStore{}, nil
	}, time.Now, func(_ *fiber.App, _ string) error { return nil }, mcpRun)
	if err := execute(cmd, args...); err != nil {
		t.Fatalf("execute mcp error = %v", err)
	}
	return captured
}
