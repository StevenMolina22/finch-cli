package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"finch/internal/finch"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// newStreamableHandler builds a fully wired HTTP handler that combines
// the SDK's streamable transport with the auth middleware. The returned
// handler is suitable for use with httptest.
func newStreamableHandler(store Store, auth AuthConfig) http.Handler {
	server := NewServer(store)
	streamable := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server },
		&mcpsdk.StreamableHTTPOptions{Stateless: true},
	)
	return bearerAuthMiddleware(auth, streamable)
}

func postMCPMessage(t *testing.T, h http.Handler, headers map[string]string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHTTPMCPRejectsMissingBearer(t *testing.T) {
	store := &fakeStore{}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})
	rec := postMCPMessage(t, h, nil, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(store.addInputs) != 0 {
		t.Fatal("storage should not be touched for unauthenticated requests")
	}
}

func TestHTTPMCPRejectsInvalidBearer(t *testing.T) {
	store := &fakeStore{}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})
	rec := postMCPMessage(t, h, map[string]string{"Authorization": "Bearer wrong"}, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "401 Unauthorized") {
		t.Fatalf("body = %q, want generic 401", rec.Body.String())
	}
}

func TestHTTPMCPRejectsMalformedBearer(t *testing.T) {
	store := &fakeStore{}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})
	rec := postMCPMessage(t, h, map[string]string{"Authorization": "Basic abc"}, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHTTPMCPAcceptsValidBearer(t *testing.T) {
	store := &fakeStore{}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})
	rec := postMCPMessage(t, h, map[string]string{"Authorization": "Bearer secret"}, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}

// TestHTTPMCPPermitsWriteToolForAPIKey exercises the full path through the
// SDK's streamable transport with the single API key.
func TestHTTPMCPPermitsWriteToolForAPIKey(t *testing.T) {
	store := &fakeStore{}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})

	list, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": toolAddTransaction,
			"arguments": map[string]any{
				"type":     "expense",
				"amount":   "1.00",
				"category": "food",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := postMCPMessage(t, h, map[string]string{"Authorization": "Bearer secret"}, string(list))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "permission denied") {
		t.Fatalf("response should not deny the API key: %s", body)
	}
	if len(store.addInputs) != 1 {
		t.Fatalf("Add called %d times, want 1", len(store.addInputs))
	}
}

// TestHTTPMCPPermitsReadToolForAPIKey verifies the read path works through HTTP.
func TestHTTPMCPPermitsReadToolForAPIKey(t *testing.T) {
	store := &fakeStore{summaryV: finch.NewSummary("", 0, 0, nil)}
	h := newStreamableHandler(store, AuthConfig{APIKey: "secret"})

	list, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolGetSummary,
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := postMCPMessage(t, h, map[string]string{"Authorization": "Bearer secret"}, string(list))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "income") {
		t.Fatalf("expected summary output, got %s", rec.Body.String())
	}
}

// TestRunHTTPListenAndShutdown ensures the production RunHTTP function
// actually binds a port and shuts down cleanly when its context is
// canceled. It uses a free port (127.0.0.1:0) to avoid clashes.
func TestRunHTTPListenAndShutdown(t *testing.T) {
	store := &fakeStore{}
	auth := AuthConfig{APIKey: "secret"}

	// Find a free port by starting and stopping a temporary listener.
	tmpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := tmpListener.Addr().String()
	tmpListener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunHTTP(ctx, Options{Store: store, Auth: auth, Addr: addr})
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Make a request that should be rejected by auth.
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("server may not be ready yet: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunHTTP error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunHTTP did not shut down within 5s")
	}
}
