package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Transport represents the wire protocol Finch will use to expose its MCP
// server. Only HTTP and stdio are supported in the first implementation.
type Transport string

const (
	// TransportHTTP exposes the MCP server over Streamable HTTP.
	TransportHTTP Transport = "http"
	// TransportStdio exposes the MCP server over standard input/output.
	// Support is optional and the command rejects it explicitly when not
	// implemented.
	TransportStdio Transport = "stdio"
)

// Options bundles the parameters required to start an MCP transport.
type Options struct {
	// Store is invoked by registered MCP tools. It must be safe for
	// concurrent use.
	Store Store
	// Auth holds the bearer token configuration used for HTTP transport.
	// For stdio transport the values are not consulted.
	Auth AuthConfig
	// Addr is the listen address used by HTTP transport, e.g. ":3333".
	Addr string
}

// ErrUnsupportedTransport is returned by Run when the supplied transport
// is not one of the supported values.
var ErrUnsupportedTransport = errors.New("unsupported MCP transport")

// ErrHTTPNoAPIKey is returned by RunHTTP when FINCH_API_KEY is not configured.
var ErrHTTPNoAPIKey = errors.New("HTTP MCP transport requires FINCH_API_KEY to be set")

// Run starts the MCP server using the given transport and blocks until
// the context is canceled or a fatal transport error occurs.
func Run(ctx context.Context, transport Transport, opts Options) error {
	switch transport {
	case TransportHTTP:
		return RunHTTP(ctx, opts)
	case TransportStdio:
		return RunStdio(ctx, opts)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedTransport, transport)
	}
}

// RunHTTP starts the MCP server using the Streamable HTTP transport. It
// fails fast before binding when no auth token is configured. It returns
// when the listener stops or the context is canceled.
func RunHTTP(ctx context.Context, opts Options) error {
	if opts.Auth.IsEmpty() {
		return ErrHTTPNoAPIKey
	}
	if opts.Addr == "" {
		opts.Addr = ":3333"
	}

	httpServer := &http.Server{
		Addr:              opts.Addr,
		Handler:           NewHTTPHandler(opts.Store, opts.Auth),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// NewHTTPHandler constructs the production HTTP routing layer for Finch MCP.
// Documentation routes are public, while the MCP transport is isolated at
// /mcp and protected by bearer authentication.
func NewHTTPHandler(store Store, auth AuthConfig) http.Handler {
	server := NewServer(store)
	streamable := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server },
		&mcpsdk.StreamableHTTPOptions{Stateless: true},
	)
	authenticatedMCP := bearerAuthMiddleware(auth, streamable)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleDocsLanding)
	mux.HandleFunc("/llms.txt", handleLLMsDocs)
	mux.HandleFunc("/.well-known/mcp.json", handleMCPMetadata)
	mux.Handle("/mcp", authenticatedMCP)
	mux.Handle("/mcp/", authenticatedMCP)
	return mux
}

func handleDocsLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Finch MCP</title></head>
<body>
<main>
<h1>Finch MCP</h1>
<p>Finch is an MCP server for agents, not a browser app.</p>
<p>MCP endpoint: <code>/mcp</code></p>
<p>Authentication: <code>Authorization: Bearer &lt;API_KEY&gt;</code></p>
<h2>Minimal MCP client config</h2>
<pre><code>{
  "mcpServers": {
    "finch": {
      "url": "https://your-finch-host.example.com/mcp",
      "headers": {
        "Authorization": "Bearer &lt;API_KEY&gt;"
      }
    }
  }
}</code></pre>
<h2>Tools</h2>
<ul>
%s
</ul>
<p>AI-readable docs: <a href="/llms.txt">/llms.txt</a></p>
</main>
</body>
</html>
`, htmlToolList())
}

func htmlToolList() string {
	var b strings.Builder
	for _, name := range ToolNames() {
		_, _ = fmt.Fprintf(&b, "<li><code>%s</code></li>\n", name)
	}
	return b.String()
}

func handleLLMsDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, `# Finch MCP

Base URL: use the deployed Finch MCP server origin.
MCP endpoint: /mcp
Transport: Streamable HTTP
Authentication: Authorization: Bearer <API_KEY>

Tools:
%s
Usage notes:
- Finch is an MCP server for agents, not a browser app.
- Send MCP JSON-RPC requests to /mcp.
- Documentation routes are public and do not require MCP initialization.
- Mutating tools may require confirm=true.
- Never expose real API keys in prompts, code, or docs.
`, plainToolList())
}

func plainToolList() string {
	var b strings.Builder
	for _, name := range ToolNames() {
		_, _ = fmt.Fprintf(&b, "- %s\n", name)
	}
	return b.String()
}

func handleMCPMetadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	metadata := map[string]any{
		"name":      serverName,
		"version":   serverVersion,
		"transport": "streamable-http",
		"endpoint":  "/mcp",
		"auth": map[string]string{
			"type": "bearer",
		},
		"docs": map[string]string{
			"human": "/",
			"llms":  "/llms.txt",
		},
		"tools": ToolNames(),
	}
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "encode metadata", http.StatusInternalServerError)
	}
}

// RunStdio starts the MCP server using standard input and output. It is
// reserved for future support and is not yet implemented; the command
// returns ErrUnsupportedTransport when stdio is requested without HTTP
// being available.
func RunStdio(ctx context.Context, _ Options) error {
	_ = ctx
	return fmt.Errorf("%w: %q is not implemented in this release", ErrUnsupportedTransport, TransportStdio)
}

// bearerAuthMiddleware wraps next with bearer token authentication. It
// rejects requests that are missing, malformed, or do not match a
// configured token, and propagates the resulting permission through the
// request context so that tool handlers can authorize per-tool access.
func bearerAuthMiddleware(cfg AuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthFailure(w)
			return
		}
		perm := classifyBearerToken(token, cfg)
		if perm == PermissionAnonymous {
			writeAuthFailure(w)
			return
		}
		ctx := WithPermission(r.Context(), perm)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
