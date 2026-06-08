package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"finch/internal/finch"
)

// Store is the narrow subset of finch.Store used by MCP tool handlers. It
// allows tests to substitute a fake implementation without depending on
// libSQL or Turso.
type Store interface {
	Add(ctx context.Context, input finch.AddInput) error
	List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error)
	Summary(ctx context.Context, month string) (finch.Summary, error)
	Update(ctx context.Context, input finch.EditInput) error
	Delete(ctx context.Context, id int64) error
}

// StoreAdapter exposes a finch.Store as a Store, satisfying the smaller
// interface the MCP handlers require.
type StoreAdapter struct {
	Store *finch.Store
}

func (a StoreAdapter) Add(ctx context.Context, input finch.AddInput) error {
	return a.Store.Add(ctx, input)
}

func (a StoreAdapter) List(ctx context.Context, filter finch.ListFilter) ([]finch.Transaction, error) {
	return a.Store.List(ctx, filter)
}

func (a StoreAdapter) Summary(ctx context.Context, month string) (finch.Summary, error) {
	return a.Store.Summary(ctx, month)
}

func (a StoreAdapter) Update(ctx context.Context, input finch.EditInput) error {
	return a.Store.Update(ctx, input)
}

func (a StoreAdapter) Delete(ctx context.Context, id int64) error {
	return a.Store.Delete(ctx, id)
}

// errMissingCategory is returned when a category is empty after trimming.
var errMissingCategory = errors.New("category is required")

func trimRequiredCategory(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errMissingCategory
	}
	return trimmed, nil
}

func trimOptional(value string) string {
	return strings.TrimSpace(value)
}

// notFoundMessage formats a structured not-found response for an MCP
// destructive tool. It does not include any error wrapping from the storage
// layer to avoid leaking implementation details.
func notFoundMessage(id int64, op string) map[string]any {
	return map[string]any{
		"success": false,
		"id":      id,
		"status":  "not_found",
		"message": fmt.Sprintf("transaction %d not found for %s", id, op),
	}
}

func successMessage(id int64, op, message string) map[string]any {
	result := map[string]any{
		"success": true,
		"status":  op,
		"message": message,
	}
	if id > 0 {
		result["id"] = id
	}
	return result
}
