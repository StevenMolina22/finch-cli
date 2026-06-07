package server

import (
	"context"
	"time"

	"finch/internal/finch"

	"github.com/gofiber/fiber/v2"
)

// Store is the minimal subset of finch.Store used by the HTTP server.
// It is defined here so handlers can be tested with a fake implementation
// without depending on a live Turso database.
type Store interface {
	Add(ctx context.Context, input finch.AddInput) error
	List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error)
	Summary(ctx context.Context, month string) (finch.Summary, error)
	Update(ctx context.Context, input finch.EditInput) error
	Delete(ctx context.Context, id int64) error
}

// ListenFunc blocks the calling goroutine and serves HTTP traffic on addr
// until the app shuts down or returns an error.
type ListenFunc func(app *fiber.App, addr string) error

// Listen is the production ListenFunc that calls Fiber's blocking Listen.
func Listen(app *fiber.App, addr string) error {
	return app.Listen(addr)
}

// NewApp constructs a *fiber.App with all HTTP routes registered.
// If store is nil, only routes that do not touch the store (such as
// GET /health) will succeed; the other handlers will return 500.
func NewApp(store Store, now func() time.Time) *fiber.App {
	return NewAppWithOpener(func() (Store, error) { return store, nil }, now)
}

// NewAppWithOpener constructs a *fiber.App whose store is resolved lazily
// via open. The opener is invoked at the start of each handler that needs
// the store, allowing command code to defer database connection until
// after configuration validation.
func NewAppWithOpener(open func() (Store, error), now func() time.Time) *fiber.App {
	if now == nil {
		now = time.Now
	}
	if open == nil {
		open = func() (Store, error) { return nil, errStoreUnavailable }
	}
	s := &server{now: now, open: open}

	app := fiber.New(fiber.Config{
		AppName:               "finch",
		DisableStartupMessage: true,
		ErrorHandler:          s.errorHandler,
	})
	s.registerRoutes(app)
	return app
}
