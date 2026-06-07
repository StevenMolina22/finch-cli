package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"finch/internal/finch"

	"github.com/gofiber/fiber/v2"
)

// errStoreUnavailable is returned when a handler cannot access a store.
var errStoreUnavailable = errors.New("store is not available")

// createTransactionRequest is the JSON shape accepted by POST /transactions.
type createTransactionRequest struct {
	Type      string `json:"type"`
	Amount    string `json:"amount"`
	Category  string `json:"category"`
	Desc      string `json:"desc"`
	Date      string `json:"date"`
	Tags      string `json:"tags"`
	Recurring string `json:"recurring"`
}

// updateTransactionRequest is the JSON shape accepted by PATCH /transactions/:id.
// Pointer fields let the handler distinguish "not provided" from "explicit zero value".
type updateTransactionRequest struct {
	Amount    *string `json:"amount,omitempty"`
	Category  *string `json:"category,omitempty"`
	Desc      *string `json:"desc,omitempty"`
	Tags      *string `json:"tags,omitempty"`
	Recurring *string `json:"recurring,omitempty"`
}

// parseTransactionID extracts and validates a positive int64 id from a path parameter.
func parseTransactionID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid transaction id: %q", raw)
	}
	return id, nil
}

// parseCreateInput decodes and validates a POST /transactions body.
func parseCreateInput(body []byte, now func() time.Time) (finch.AddInput, error) {
	if len(body) == 0 {
		return finch.AddInput{}, fmt.Errorf("request body is required")
	}
	var req createTransactionRequest
	if err := jsonUnmarshal(body, &req); err != nil {
		return finch.AddInput{}, fmt.Errorf("invalid JSON body: %w", err)
	}

	if err := finch.ValidateType(req.Type); err != nil {
		return finch.AddInput{}, err
	}
	cents, err := finch.ParseAmount(req.Amount)
	if err != nil {
		return finch.AddInput{}, err
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return finch.AddInput{}, fmt.Errorf("category is required")
	}
	if err := finch.ValidateDate(req.Date); err != nil {
		return finch.AddInput{}, err
	}
	if err := finch.ValidateRecurring(strings.TrimSpace(req.Recurring)); err != nil {
		return finch.AddInput{}, err
	}

	date := strings.TrimSpace(req.Date)
	if date == "" {
		date = now().UTC().Format("2006-01-02")
	}

	return finch.AddInput{
		Type:        req.Type,
		AmountCents: cents,
		Category:    category,
		Desc:        strings.TrimSpace(req.Desc),
		Date:        date,
		Tags:        strings.TrimSpace(req.Tags),
		Recurring:   strings.TrimSpace(req.Recurring),
	}, nil
}

// parseUpdateInput decodes and validates a PATCH /transactions/:id body and
// converts it into a finch.EditInput. Returns an error when no editable
// field is supplied or any provided field fails validation.
func parseUpdateInput(body []byte, id int64) (finch.EditInput, error) {
	if len(body) == 0 {
		return finch.EditInput{}, fmt.Errorf("request body is required")
	}
	var req updateTransactionRequest
	if err := jsonUnmarshal(body, &req); err != nil {
		return finch.EditInput{}, fmt.Errorf("invalid JSON body: %w", err)
	}

	input := finch.EditInput{ID: id}

	if req.Amount != nil {
		cents, err := finch.ParseAmount(*req.Amount)
		if err != nil {
			return finch.EditInput{}, err
		}
		input.AmountCents = &cents
	}
	if req.Category != nil {
		v := strings.TrimSpace(*req.Category)
		if v == "" {
			return finch.EditInput{}, fmt.Errorf("category is required")
		}
		input.Category = &v
	}
	if req.Desc != nil {
		v := strings.TrimSpace(*req.Desc)
		input.Desc = &v
	}
	if req.Tags != nil {
		v := strings.TrimSpace(*req.Tags)
		input.Tags = &v
	}
	if req.Recurring != nil {
		v := strings.TrimSpace(*req.Recurring)
		if err := finch.ValidateRecurring(v); err != nil {
			return finch.EditInput{}, err
		}
		input.Recurring = &v
	}

	if err := finch.ValidateEditFields(input); err != nil {
		return finch.EditInput{}, err
	}
	return input, nil
}

// parseListFilter reads and validates month, category, and limit query
// parameters. Empty category and zero limit are treated as unset.
func parseListFilter(c *fiber.Ctx) (finch.ListFilter, error) {
	month := strings.TrimSpace(c.Query("month"))
	if err := finch.ValidateMonth(month); err != nil {
		return finch.ListFilter{}, err
	}
	category := strings.TrimSpace(c.Query("category"))

	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return finch.ListFilter{}, fmt.Errorf("limit must be a positive integer")
		}
		if err := finch.ValidateLimit(n); err != nil {
			return finch.ListFilter{}, err
		}
		limit = n
	}

	return finch.ListFilter{Month: month, Category: category, Limit: limit}, nil
}

// parseSummaryMonth reads and validates the optional month query parameter.
func parseSummaryMonth(c *fiber.Ctx) (string, error) {
	month := strings.TrimSpace(c.Query("month"))
	if err := finch.ValidateMonth(month); err != nil {
		return "", err
	}
	return month, nil
}
