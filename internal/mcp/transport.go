package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// ErrHTTPNoAuthTokens is returned by RunHTTP when neither FINCH_MCP_READ_TOKEN
// nor FINCH_MCP_WRITE_TOKEN is configured.
var ErrHTTPNoAuthTokens = errors.New("HTTP MCP transport requires FINCH_MCP_READ_TOKEN and/or FINCH_MCP_WRITE_TOKEN to be set")

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
		return ErrHTTPNoAuthTokens
	}
	if opts.Addr == "" {
		opts.Addr = ":3333"
	}

	server := NewServer(opts.Store)
	handler := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server },
		&mcpsdk.StreamableHTTPOptions{Stateless: true},
	)

	mux := http.NewServeMux()
	mux.Handle("/", bearerAuthMiddleware(opts.Auth, handler))

	httpServer := &http.Server{
		Addr:              opts.Addr,
		Handler:           mux,
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
