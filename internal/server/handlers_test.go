package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"finch/internal/finch"

	"github.com/gofiber/fiber/v2"
)

// fakeStore is a hand-rolled test double that records calls and lets
// individual tests configure behavior. It does not persist data.
type fakeStore struct {
	mu sync.Mutex

	addCalls    int
	listCalls   int
	summaryCalls int
	updateCalls int
	deleteCalls  int

	addErr       error
	listErr      error
	summaryErr   error
	updateErr    error
	deleteErr    error

	lastAdd     finch.AddInput
	lastList    finch.ListFilter
	lastSummary string
	lastUpdate  finch.EditInput
	lastDelete  int64
}

func (f *fakeStore) Add(ctx context.Context, input finch.AddInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.addCalls++
	f.lastAdd = input
	return f.addErr
}

func (f *fakeStore) List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	f.lastList = filter
	return nil, f.listErr
}

func (f *fakeStore) Summary(ctx context.Context, month string) (finch.Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.summaryCalls++
	f.lastSummary = month
	return finch.NewSummary(month, 0, 0, nil), f.summaryErr
}

func (f *fakeStore) Update(ctx context.Context, input finch.EditInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateCalls++
	f.lastUpdate = input
	return f.updateErr
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls++
	f.lastDelete = id
	return f.deleteErr
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
}

func do(t *testing.T, app *fiber.App, method, path string, body []byte) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body error: %v", err)
	}
	return &httptest.ResponseRecorder{
		HeaderMap: resp.Header,
		Body:      bytes.NewBuffer(raw),
		Code:      resp.StatusCode,
	}, raw
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode JSON: %v (body=%s)", err, string(raw))
	}
	return v
}

func TestHealthReturnsJSONWithoutStore(t *testing.T) {
	// /health must not require or touch the store. We pass nil to ensure
	// the route is registered independently of the database.
	app := NewApp(nil, fixedNow)
	resp, raw := do(t, app, "GET", "/health", nil)
	if resp.Code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.Code, string(raw))
	}
	got := decode[map[string]any](t, raw)
	if got["status"] != "ok" {
		t.Fatalf("body = %v, want status=ok", got)
	}
	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}

func TestPostTransactionSuccess(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	body := []byte(`{"type":"expense","amount":"12.34","category":"groceries","desc":"weekly","date":"2026-05-31"}`)
	resp, raw := do(t, app, "POST", "/transactions", body)
	if resp.Code != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", resp.Code, string(raw))
	}
	if store.addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", store.addCalls)
	}
	if store.lastAdd.Type != "expense" || store.lastAdd.AmountCents != 1234 ||
		store.lastAdd.Category != "groceries" || store.lastAdd.Date != "2026-05-31" {
		t.Fatalf("last add = %+v", store.lastAdd)
	}
	got := decode[map[string]any](t, raw)
	if got["status"] != "created" || got["date"] != "2026-05-31" || got["type"] != "expense" {
		t.Fatalf("response = %v", got)
	}
}

func TestPostTransactionDefaultsDateToUTC(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	body := []byte(`{"type":"income","amount":"50.00","category":"salary"}`)
	resp, raw := do(t, app, "POST", "/transactions", body)
	if resp.Code != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", resp.Code, string(raw))
	}
	if store.lastAdd.Date != "2026-05-26" {
		t.Fatalf("last add date = %q, want 2026-05-26", store.lastAdd.Date)
	}
}

func TestPostTransactionValidationFailure(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid type", body: `{"type":"transfer","amount":"1.00","category":"x"}`},
		{name: "invalid amount", body: `{"type":"expense","amount":"0","category":"x"}`},
		{name: "empty category", body: `{"type":"expense","amount":"1.00","category":""}`},
		{name: "invalid date", body: `{"type":"expense","amount":"1.00","category":"x","date":"2026/05/31"}`},
		{name: "invalid recurring", body: `{"type":"expense","amount":"1.00","category":"x","recurring":"daily"}`},
		{name: "empty body", body: ``},
		{name: "invalid JSON", body: `not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			app := NewApp(store, fixedNow)
			resp, raw := do(t, app, "POST", "/transactions", []byte(tt.body))
			if resp.Code != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
			}
			if store.addCalls != 0 {
				t.Fatalf("add was called %d times, want 0", store.addCalls)
			}
			got := decode[map[string]any](t, raw)
			if _, ok := got["error"]; !ok {
				t.Fatalf("response missing error key: %v", got)
			}
		})
	}
}

func TestGetTransactions(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)

	resp, raw := do(t, app, "GET", "/transactions?month=2026-05&category=groceries&limit=5", nil)
	if resp.Code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.Code, string(raw))
	}
	if store.listCalls != 1 {
		t.Fatalf("list calls = %d, want 1", store.listCalls)
	}
	if store.lastList.Month != "2026-05" || store.lastList.Category != "groceries" || store.lastList.Limit != 5 {
		t.Fatalf("last list = %+v", store.lastList)
	}
	if !strings.Contains(string(raw), "[]") {
		t.Fatalf("expected empty JSON array, got %s", string(raw))
	}
}

func TestGetTransactionsInvalidFilters(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "invalid month", path: "/transactions?month=May-2026"},
		{name: "invalid limit", path: "/transactions?limit=0"},
		{name: "non numeric limit", path: "/transactions?limit=abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			app := NewApp(store, fixedNow)
			resp, raw := do(t, app, "GET", tt.path, nil)
			if resp.Code != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
			}
			if store.listCalls != 0 {
				t.Fatalf("list was called %d times, want 0", store.listCalls)
			}
		})
	}
}

func TestGetSummary(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)

	resp, raw := do(t, app, "GET", "/summary?month=2026-05", nil)
	if resp.Code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.Code, string(raw))
	}
	if store.summaryCalls != 1 || store.lastSummary != "2026-05" {
		t.Fatalf("summary calls = %d, last = %q", store.summaryCalls, store.lastSummary)
	}
	if !strings.Contains(string(raw), `"income"`) {
		t.Fatalf("summary response missing income field: %s", string(raw))
	}
}

func TestSummaryInvalidMonth(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "GET", "/summary?month=May-2026", nil)
	if resp.Code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
	}
}

func TestPatchTransactionSuccess(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	body := []byte(`{"amount":"5.00","category":"snacks"}`)
	resp, raw := do(t, app, "PATCH", "/transactions/42", body)
	if resp.Code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.Code, string(raw))
	}
	if store.updateCalls != 1 || store.lastUpdate.ID != 42 {
		t.Fatalf("update calls = %d, last = %+v", store.updateCalls, store.lastUpdate)
	}
	if store.lastUpdate.AmountCents == nil || *store.lastUpdate.AmountCents != 500 {
		t.Fatalf("amount = %v, want 500", store.lastUpdate.AmountCents)
	}
	if store.lastUpdate.Category == nil || *store.lastUpdate.Category != "snacks" {
		t.Fatalf("category = %v, want snacks", store.lastUpdate.Category)
	}
}

func TestPatchTransactionInvalidID(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "PATCH", "/transactions/not-an-id", []byte(`{"amount":"1.00"}`))
	if resp.Code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
	}
	if store.updateCalls != 0 {
		t.Fatalf("update was called %d times, want 0", store.updateCalls)
	}
}

func TestPatchTransactionWithoutFields(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "PATCH", "/transactions/1", []byte(`{}`))
	if resp.Code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
	}
	if store.updateCalls != 0 {
		t.Fatalf("update was called %d times, want 0", store.updateCalls)
	}
}

func TestPatchTransactionInvalidField(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "PATCH", "/transactions/1", []byte(`{"recurring":"daily"}`))
	if resp.Code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
	}
	if store.updateCalls != 0 {
		t.Fatalf("update was called %d times, want 0", store.updateCalls)
	}
}

func TestPatchTransactionMissingReturns404(t *testing.T) {
	store := &fakeStore{updateErr: finch.ErrTransactionNotFound}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "PATCH", "/transactions/99", []byte(`{"amount":"5.00"}`))
	if resp.Code != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.Code, string(raw))
	}
	got := decode[map[string]any](t, raw)
	if _, ok := got["error"]; !ok {
		t.Fatalf("response missing error key: %v", got)
	}
}

func TestDeleteTransactionSuccess(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "DELETE", "/transactions/7", nil)
	if resp.Code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.Code, string(raw))
	}
	if store.deleteCalls != 1 || store.lastDelete != 7 {
		t.Fatalf("delete calls = %d, last = %d", store.deleteCalls, store.lastDelete)
	}
}

func TestDeleteTransactionInvalidID(t *testing.T) {
	store := &fakeStore{}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "DELETE", "/transactions/abc", nil)
	if resp.Code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.Code, string(raw))
	}
	if store.deleteCalls != 0 {
		t.Fatalf("delete was called %d times, want 0", store.deleteCalls)
	}
}

func TestDeleteTransactionMissingReturns404(t *testing.T) {
	store := &fakeStore{deleteErr: finch.ErrTransactionNotFound}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "DELETE", "/transactions/99", nil)
	if resp.Code != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.Code, string(raw))
	}
}

func TestStoreErrorReturns500(t *testing.T) {
	store := &fakeStore{listErr: errors.New("database is down")}
	app := NewApp(store, fixedNow)
	resp, raw := do(t, app, "GET", "/transactions", nil)
	if resp.Code != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", resp.Code, string(raw))
	}
	got := decode[map[string]any](t, raw)
	if _, ok := got["error"]; !ok {
		t.Fatalf("response missing error key: %v", got)
	}
}

func TestStoreUnavailableReturns500(t *testing.T) {
	app := NewApp(nil, fixedNow)
	resp, raw := do(t, app, "GET", "/transactions", nil)
	if resp.Code != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", resp.Code, string(raw))
	}
}
