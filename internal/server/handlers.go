package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"finch/internal/finch"

	"github.com/gofiber/fiber/v2"
)

// server holds request-scoped dependencies for the handlers.
type server struct {
	now  func() time.Time
	open func() (Store, error)
}

func (s *server) registerRoutes(app *fiber.App) {
	app.Get("/health", s.handleHealth)
	app.Post("/transactions", s.handleCreateTransaction)
	app.Get("/transactions", s.handleListTransactions)
	app.Get("/summary", s.handleSummary)
	app.Patch("/transactions/:id", s.handleUpdateTransaction)
	app.Delete("/transactions/:id", s.handleDeleteTransaction)
}

func (s *server) errorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	var fe *fiber.Error
	if errors.As(err, &fe) {
		status = fe.Code
	}
	return writeError(c, status, err.Error())
}

// resolveStore returns the configured store or an error suitable for a
// 500 response when no store is available.
func (s *server) resolveStore() (Store, error) {
	if s.open == nil {
		return nil, errStoreUnavailable
	}
	store, err := s.open()
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, errStoreUnavailable
	}
	return store, nil
}

func (s *server) handleHealth(c *fiber.Ctx) error {
	return writeJSON(c, fiber.Map{
		"status": "ok",
	})
}

func (s *server) handleCreateTransaction(c *fiber.Ctx) error {
	input, err := parseCreateInput(c.Body(), s.now)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	store, err := s.resolveStore()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "store unavailable")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	if err := store.Add(ctx, input); err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}

	return writeCreated(c, createResponse{
		Status: "created",
		Date:   input.Date,
		Type:   input.Type,
	})
}

func (s *server) handleListTransactions(c *fiber.Ctx) error {
	filter, err := parseListFilter(c)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	store, err := s.resolveStore()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "store unavailable")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	transactions, err := store.List(ctx, filter)
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	if transactions == nil {
		transactions = []finch.Transaction{}
	}
	return writeJSON(c, transactions)
}

func (s *server) handleSummary(c *fiber.Ctx) error {
	month, err := parseSummaryMonth(c)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	store, err := s.resolveStore()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "store unavailable")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	summary, err := store.Summary(ctx, month)
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, summary)
}

func (s *server) handleUpdateTransaction(c *fiber.Ctx) error {
	id, err := parseTransactionID(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	input, err := parseUpdateInput(c.Body(), id)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	store, err := s.resolveStore()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "store unavailable")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	if err := store.Update(ctx, input); err != nil {
		if errors.Is(err, finch.ErrTransactionNotFound) {
			return writeError(c, fiber.StatusNotFound, fmt.Sprintf("transaction %d not found", id))
		}
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}

	return writeJSON(c, updateResponse{
		ID:      id,
		Status:  "updated",
		Message: "transaction updated",
	})
}

func (s *server) handleDeleteTransaction(c *fiber.Ctx) error {
	id, err := parseTransactionID(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	store, err := s.resolveStore()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "store unavailable")
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	if err := store.Delete(ctx, id); err != nil {
		if errors.Is(err, finch.ErrTransactionNotFound) {
			return writeError(c, fiber.StatusNotFound, fmt.Sprintf("transaction %d not found", id))
		}
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}

	return writeJSON(c, deleteResponse{
		ID:      id,
		Status:  "deleted",
		Message: "transaction deleted",
	})
}
