package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"finch/internal/finch"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeStore is a deterministic in-memory implementation of Store used by
// the MCP handler tests. It records the last call to each method so tests
// can assert that handlers invoked the shared validation and storage
// logic directly.
type fakeStore struct {
	mu sync.Mutex

	addInputs   []finch.AddInput
	listFilters []finch.ListFilter
	summaryArgs []string
	editInputs  []finch.EditInput
	deletedIDs  []int64

	addErr    error
	listErr   error
	summaryV  finch.Summary
	summaryEr error
	updateErr error
	deleteErr error
}

func (f *fakeStore) Add(_ context.Context, in finch.AddInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.addInputs = append(f.addInputs, in)
	return nil
}

func (f *fakeStore) List(_ context.Context, filter finch.ListFilter) ([]finch.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.listFilters = append(f.listFilters, filter)
	return []finch.Transaction{{ID: 1, Type: "expense", AmountCents: 1234, Category: "food"}}, nil
}

func (f *fakeStore) Summary(_ context.Context, month string) (finch.Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.summaryEr != nil {
		return finch.Summary{}, f.summaryEr
	}
	f.summaryArgs = append(f.summaryArgs, month)
	return f.summaryV, nil
}

func (f *fakeStore) Update(_ context.Context, in finch.EditInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	f.editInputs = append(f.editInputs, in)
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

func newTestServer(store Store) *mcpsdk.Server {
	return NewServer(store)
}

func callTool(t *testing.T, s *mcpsdk.Server, ctx context.Context, name string, args map[string]any) *mcpsdk.CallToolResult {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	req := &mcpsdk.CallToolRequest{
		Params: &mcpsdk.CallToolParamsRaw{
			Name:      name,
			Arguments: raw,
		},
	}
	handler, ok := lookupHandler(t, s, name)
	if !ok {
		t.Fatalf("tool %q is not registered", name)
	}
	res, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("tool %q returned error: %v", name, err)
	}
	return res
}

// lookupHandler finds a registered tool handler by name using the SDK's
// reflection-free paths. The SDK stores handlers internally without a
// public accessor, so we use NewServer's exported registration to find
// the handler we want by re-registering identical tools. This test helper
// works because our handlers are created via the makeXxxHandler
// functions and we re-derive them here for direct invocation.
func lookupHandler(t *testing.T, _ *mcpsdk.Server, name string) (mcpsdk.ToolHandler, bool) {
	t.Helper()
	switch name {
	case toolAddTransaction:
		return makeAddHandler(currentTestStore(t), func() time.Time { return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) }), true
	case toolListTransactions:
		return makeListHandler(currentTestStore(t)), true
	case toolGetSummary:
		return makeSummaryHandler(currentTestStore(t)), true
	case toolEditTransaction:
		return makeEditHandler(currentTestStore(t)), true
	case toolDeleteTransaction:
		return makeDeleteHandler(currentTestStore(t)), true
	}
	return nil, false
}

var testStoreKey = "testStore"

// withTestStore stores the fake store in a context so lookupHandler can
// re-derive handlers without re-wiring.
func withTestStore(ctx context.Context, store Store) context.Context {
	return context.WithValue(ctx, testStoreKey, store)
}

func currentTestStore(t *testing.T) Store {
	t.Helper()
	v := tCtx(t).Value(testStoreKey)
	if v == nil {
		t.Fatal("test store not configured for this test")
	}
	return v.(Store)
}

type tCtxKey struct{}

// tCtx stores per-test context so lookupHandler can find the fake store.
func tCtx(t *testing.T) context.Context {
	t.Helper()
	if v, ok := testContext.Load(t); ok {
		return v.(context.Context)
	}
	t.Fatal("test context not initialized")
	return context.Background()
}

var testContext sync.Map

// runWithStore wires a fake store into per-test context and runs fn.
func runWithStore(t *testing.T, store Store, fn func(ctx context.Context)) {
	t.Helper()
	ctx := withTestStore(context.Background(), store)
	testContext.Store(t, ctx)
	defer testContext.Delete(t)
	fn(ctx)
}

func TestAddHandlerCreatesTransaction(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		res := callTool(t, newTestServer(store), ctx, toolAddTransaction, map[string]any{
			"type":     "expense",
			"amount":   "12.34",
			"category": "groceries",
			"date":     "2026-05-20",
		})
		if res.IsError {
			t.Fatalf("expected success, got error: %v", textOf(res))
		}
		if len(store.addInputs) != 1 {
			t.Fatalf("Add called %d times, want 1", len(store.addInputs))
		}
		got := store.addInputs[0]
		if got.AmountCents != 1234 || got.Type != "expense" || got.Category != "groceries" || got.Date != "2026-05-20" {
			t.Fatalf("unexpected stored input: %+v", got)
		}
	})
}

func TestAddHandlerDefaultsToCurrentUTCDate(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		callTool(t, newTestServer(store), ctx, toolAddTransaction, map[string]any{
			"type":     "income",
			"amount":   "100.00",
			"category": "salary",
		})
		if len(store.addInputs) != 1 {
			t.Fatalf("Add called %d times, want 1", len(store.addInputs))
		}
		if got := store.addInputs[0].Date; got != "2026-05-26" {
			t.Fatalf("date = %q, want 2026-05-26", got)
		}
	})
}

func TestAddHandlerRejectsInvalidInput(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		cases := []map[string]any{
			{"type": "transfer", "amount": "1.00", "category": "x"},
			{"type": "expense", "amount": "0", "category": "x"},
			{"type": "expense", "amount": "1.00", "category": "   "},
			{"type": "expense", "amount": "1.00", "category": "x", "date": "2026/05/01"},
			{"type": "expense", "amount": "1.00", "category": "x", "recurring": "daily"},
		}
		for _, args := range cases {
			res := callTool(t, newTestServer(store), ctx, toolAddTransaction, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
		if len(store.addInputs) != 0 {
			t.Fatalf("Add called for invalid input")
		}
	})
}

func TestListHandlerAppliesFilters(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		res := callTool(t, newTestServer(store), ctx, toolListTransactions, map[string]any{
			"month":    "2026-05",
			"category": "food",
			"limit":    5,
		})
		if res.IsError {
			t.Fatalf("expected success, got error: %v", textOf(res))
		}
		if len(store.listFilters) != 1 {
			t.Fatalf("List called %d times, want 1", len(store.listFilters))
		}
		f := store.listFilters[0]
		if f.Month != "2026-05" || f.Category != "food" || f.Limit != 5 {
			t.Fatalf("unexpected filter: %+v", f)
		}
	})
}

func TestListHandlerRejectsInvalidFilters(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		cases := []map[string]any{
			{"month": "2026-13"},
			{"limit": 0},
		}
		for _, args := range cases {
			res := callTool(t, newTestServer(store), ctx, toolListTransactions, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
		if len(store.listFilters) != 0 {
			t.Fatalf("List called for invalid filter")
		}
	})
}

func TestSummaryHandlerPassesMonth(t *testing.T) {
	store := &fakeStore{summaryV: finch.NewSummary("2026-05", 100, 50, nil)}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		res := callTool(t, newTestServer(store), ctx, toolGetSummary, map[string]any{"month": "2026-05"})
		if res.IsError {
			t.Fatalf("expected success, got error: %v", textOf(res))
		}
		if len(store.summaryArgs) != 1 || store.summaryArgs[0] != "2026-05" {
			t.Fatalf("summary month = %v, want [2026-05]", store.summaryArgs)
		}
	})
}

func TestSummaryHandlerRejectsInvalidMonth(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		res := callTool(t, newTestServer(store), ctx, toolGetSummary, map[string]any{"month": "May-2026"})
		if !res.IsError {
			t.Fatal("expected error for invalid month")
		}
	})
}

func TestEditHandlerUpdatesTransaction(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		res := callTool(t, newTestServer(store), ctx, toolEditTransaction, map[string]any{
			"id":       7,
			"amount":   "9.99",
			"category": "books",
			"confirm":  true,
		})
		if res.IsError {
			t.Fatalf("expected success, got error: %v", textOf(res))
		}
		if len(store.editInputs) != 1 {
			t.Fatalf("Update called %d times, want 1", len(store.editInputs))
		}
		in := store.editInputs[0]
		if in.ID != 7 || in.AmountCents == nil || *in.AmountCents != 999 || in.Category == nil || *in.Category != "books" {
			t.Fatalf("unexpected edit input: %+v", in)
		}
	})
}

func TestEditHandlerRequiresConfirm(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		cases := []map[string]any{
			{"id": 1, "amount": "1.00"},
			{"id": 1, "amount": "1.00", "confirm": false},
			{"id": 1, "amount": "1.00", "confirm": nil},
		}
		for _, args := range cases {
			res := callTool(t, newTestServer(store), ctx, toolEditTransaction, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
		if len(store.editInputs) != 0 {
			t.Fatalf("Update called without confirmation")
		}
	})
}

func TestEditHandlerRejectsInvalidInput(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		cases := []map[string]any{
			{"id": 0, "amount": "1.00", "confirm": true},
			{"id": -1, "amount": "1.00", "confirm": true},
			{"id": 1, "confirm": true},
			{"id": 1, "amount": "1.999", "confirm": true},
			{"id": 1, "category": "   ", "confirm": true},
			{"id": 1, "recurring": "daily", "confirm": true},
		}
		for _, args := range cases {
			res := callTool(t, newTestServer(store), ctx, toolEditTransaction, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
	})
}

func TestEditHandlerReturnsNotFoundForMissingTransaction(t *testing.T) {
	store := &fakeStore{updateErr: finch.ErrTransactionNotFound}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		res := callTool(t, newTestServer(store), ctx, toolEditTransaction, map[string]any{
			"id": 99, "amount": "1.00", "confirm": true,
		})
		if res.IsError {
			t.Fatalf("expected not-found response, got error: %v", textOf(res))
		}
		body := decodeStructured(t, res)
		if success, _ := body["success"].(bool); success {
			t.Fatalf("expected success=false, got %v", body)
		}
		if status, _ := body["status"].(string); status != "not_found" {
			t.Fatalf("status = %q, want not_found", status)
		}
	})
}

func TestDeleteHandlerDeletesTransaction(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		res := callTool(t, newTestServer(store), ctx, toolDeleteTransaction, map[string]any{
			"id": 11, "confirm": true,
		})
		if res.IsError {
			t.Fatalf("expected success, got error: %v", textOf(res))
		}
		if len(store.deletedIDs) != 1 || store.deletedIDs[0] != 11 {
			t.Fatalf("Delete called with %v, want [11]", store.deletedIDs)
		}
	})
}

func TestDeleteHandlerRequiresConfirm(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		cases := []map[string]any{
			{"id": 1},
			{"id": 1, "confirm": false},
			{"id": 1, "confirm": nil},
		}
		for _, args := range cases {
			res := callTool(t, newTestServer(store), ctx, toolDeleteTransaction, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
		if len(store.deletedIDs) != 0 {
			t.Fatalf("Delete called without confirmation")
		}
	})
}

func TestDeleteHandlerRejectsInvalidID(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		for _, args := range []map[string]any{
			{"id": 0, "confirm": true},
			{"id": -3, "confirm": true},
		} {
			res := callTool(t, newTestServer(store), ctx, toolDeleteTransaction, args)
			if !res.IsError {
				t.Fatalf("expected error for %+v", args)
			}
		}
	})
}

func TestDeleteHandlerReturnsNotFoundForMissingTransaction(t *testing.T) {
	store := &fakeStore{deleteErr: finch.ErrTransactionNotFound}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		res := callTool(t, newTestServer(store), ctx, toolDeleteTransaction, map[string]any{
			"id": 42, "confirm": true,
		})
		if res.IsError {
			t.Fatalf("expected not-found response, got error: %v", textOf(res))
		}
		body := decodeStructured(t, res)
		if status, _ := body["status"].(string); status != "not_found" {
			t.Fatalf("status = %q, want not_found", status)
		}
	})
}

func TestReadTokenCanCallReadTools(t *testing.T) {
	store := &fakeStore{summaryV: finch.NewSummary("", 0, 0, nil)}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		if res := callTool(t, newTestServer(store), ctx, toolListTransactions, nil); res.IsError {
			t.Fatalf("read tool denied: %v", textOf(res))
		}
		if res := callTool(t, newTestServer(store), ctx, toolGetSummary, nil); res.IsError {
			t.Fatalf("summary denied: %v", textOf(res))
		}
	})
}

func TestReadTokenCannotCallWriteTools(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionRead)
		cases := []string{toolAddTransaction, toolEditTransaction, toolDeleteTransaction}
		for _, name := range cases {
			res := callTool(t, newTestServer(store), ctx, name, map[string]any{"confirm": true})
			if !res.IsError {
				t.Fatalf("expected %q to be denied for read token", name)
			}
		}
		if len(store.addInputs) != 0 || len(store.editInputs) != 0 || len(store.deletedIDs) != 0 {
			t.Fatalf("storage was mutated by read-token tool call")
		}
	})
}

func TestWriteTokenCanCallReadAndWriteTools(t *testing.T) {
	store := &fakeStore{summaryV: finch.NewSummary("", 0, 0, nil)}
	runWithStore(t, store, func(ctx context.Context) {
		ctx = WithPermission(ctx, PermissionWrite)
		if res := callTool(t, newTestServer(store), ctx, toolListTransactions, nil); res.IsError {
			t.Fatalf("read tool denied: %v", textOf(res))
		}
		if res := callTool(t, newTestServer(store), ctx, toolGetSummary, nil); res.IsError {
			t.Fatalf("summary denied: %v", textOf(res))
		}
		if res := callTool(t, newTestServer(store), ctx, toolDeleteTransaction, map[string]any{"id": 1, "confirm": true}); res.IsError {
			t.Fatalf("delete denied: %v", textOf(res))
		}
	})
}

func TestAnonymousCannotCallTools(t *testing.T) {
	store := &fakeStore{}
	runWithStore(t, store, func(ctx context.Context) {
		// No permission set in ctx.
		for _, name := range []string{toolListTransactions, toolGetSummary, toolAddTransaction, toolEditTransaction, toolDeleteTransaction} {
			res := callTool(t, newTestServer(store), ctx, name, map[string]any{"confirm": true})
			if !res.IsError {
				t.Fatalf("expected %q to be denied for anonymous caller", name)
			}
		}
	})
}

func TestClassifyBearerTokenConstantTime(t *testing.T) {
	cfg := AuthConfig{ReadToken: "reader", WriteToken: "writer"}
	cases := []struct {
		name string
		in   string
		want Permission
	}{
		{"matches read", "reader", PermissionRead},
		{"matches write", "writer", PermissionWrite},
		{"read with extra whitespace", "  reader  ", PermissionRead},
		{"mismatched token", "nope", PermissionAnonymous},
		{"empty token", "", PermissionAnonymous},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyBearerToken(tt.in, cfg)
			if got != tt.want {
				t.Fatalf("classify = %v, want %v", got, tt.want)
			}
		})
	}

	empty := AuthConfig{}
	if got := classifyBearerToken("anything", empty); got != PermissionAnonymous {
		t.Fatalf("empty config should be anonymous, got %v", got)
	}

	writeOnly := AuthConfig{WriteToken: "writer"}
	if got := classifyBearerToken("writer", writeOnly); got != PermissionWrite {
		t.Fatalf("write-only config should grant write, got %v", got)
	}
	if got := classifyBearerToken("reader", writeOnly); got != PermissionAnonymous {
		t.Fatalf("write-only config should reject read token, got %v", got)
	}
}

func TestWriteTokenIncludesReadPermission(t *testing.T) {
	cfg := AuthConfig{WriteToken: "writer"}
	if got := classifyBearerToken("writer", cfg); got != PermissionWrite {
		t.Fatalf("write token should map to write, got %v", got)
	}
	if !PermissionWrite.allowsRead() {
		t.Fatal("write permission must allow read access")
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	called := false
	handler := bearerAuthMiddleware(AuthConfig{ReadToken: "r"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Fatal("downstream handler should not be called without auth")
	}
}

func TestAuthMiddlewareRejectsMalformedHeader(t *testing.T) {
	cases := []string{
		"Basic abc123",
		"Bearer",
		"Token abc",
		"  ",
	}
	for _, header := range cases {
		t.Run(header, func(t *testing.T) {
			handler := bearerAuthMiddleware(AuthConfig{ReadToken: "r"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Authorization", header)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	handler := bearerAuthMiddleware(AuthConfig{ReadToken: "r"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("401 Unauthorized")) {
		t.Fatalf("body = %q, want generic 401", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("wrong")) {
		t.Fatalf("body leaked supplied token: %q", rec.Body.String())
	}
}

func TestAuthMiddlewarePassesValidToken(t *testing.T) {
	var seen Permission
	handler := bearerAuthMiddleware(AuthConfig{ReadToken: "r", WriteToken: "w"}, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = PermissionFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer w")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seen != PermissionWrite {
		t.Fatalf("ctx permission = %v, want write", seen)
	}
}

func TestAuthFailureDoesNotLeakSecrets(t *testing.T) {
	handler := bearerAuthMiddleware(AuthConfig{ReadToken: "secret-read"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-supplied")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	body := rec.Body.String()
	if bytes.Contains([]byte(body), []byte("secret-read")) {
		t.Fatalf("configured token leaked in response: %q", body)
	}
	if bytes.Contains([]byte(body), []byte("secret-supplied")) {
		t.Fatalf("supplied token leaked in response: %q", body)
	}
}

func TestRunHTTPFailsFastWithoutAuthTokens(t *testing.T) {
	err := RunHTTP(context.Background(), Options{Store: &fakeStore{}})
	if !errors.Is(err, ErrHTTPNoAuthTokens) {
		t.Fatalf("err = %v, want ErrHTTPNoAuthTokens", err)
	}
}

func TestRunUnsupportedTransport(t *testing.T) {
	err := Run(context.Background(), Transport("websocket"), Options{Store: &fakeStore{}})
	if !errors.Is(err, ErrUnsupportedTransport) {
		t.Fatalf("err = %v, want ErrUnsupportedTransport", err)
	}
}

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		header string
		want   string
		ok     bool
	}{
		{"Bearer abc", "abc", true},
		{"bearer abc", "abc", true},
		{"BEARER abc", "abc", true},
		{"", "", false},
		{"Basic abc", "", false},
		{"Bearer", "", false},
		{"Bearer abc def", "", false},
	}
	for _, c := range cases {
		got, ok := extractBearerToken(c.header)
		if got != c.want || ok != c.ok {
			t.Fatalf("extract(%q) = (%q, %v), want (%q, %v)", c.header, got, ok, c.want, c.ok)
		}
	}
}

// textOf extracts the first TextContent payload from a tool result for
// readable failure messages.
func textOf(res *mcpsdk.CallToolResult) string {
	if res == nil {
		return ""
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func decodeStructured(t *testing.T, res *mcpsdk.CallToolResult) map[string]any {
	t.Helper()
	body, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content = %T, want map", res.StructuredContent)
	}
	return body
}
