package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"finch/internal/finch"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const isoDateLayout = "2006-01-02"

// addInput mirrors the JSON schema declared for finch_add_transaction.
// Fields are pointers so callers can detect "field not provided" vs
// "field provided with zero value" (especially for confirm and the
// editable fields).
type addInput struct {
	Type      string `json:"type"`
	Amount    string `json:"amount"`
	Category  string `json:"category"`
	Desc      string `json:"desc"`
	Date      string `json:"date"`
	Tags      string `json:"tags"`
	Recurring string `json:"recurring"`
	Confirm   *bool  `json:"confirm,omitempty"`
}

type listInput struct {
	Month    string `json:"month"`
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

type summaryInput struct {
	Month string `json:"month"`
}

type editInput struct {
	ID        int64   `json:"id"`
	Amount    *string `json:"amount,omitempty"`
	Category  *string `json:"category,omitempty"`
	Desc      *string `json:"desc,omitempty"`
	Tags      *string `json:"tags,omitempty"`
	Recurring *string `json:"recurring,omitempty"`
	Confirm   *bool   `json:"confirm,omitempty"`
}

type deleteInput struct {
	ID      int64 `json:"id"`
	Confirm *bool `json:"confirm,omitempty"`
}

func makeAddHandler(store Store, now func() time.Time) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if denied := requireWrite(ctx, toolAddTransaction); denied != nil {
			return denied, nil
		}
		var in addInput
		if err := unmarshalArgs(req, &in); err != nil {
			return errorResult(err.Error()), nil
		}
		if err := finch.ValidateType(in.Type); err != nil {
			return errorResult(err.Error()), nil
		}
		amountCents, err := finch.ParseAmount(in.Amount)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		category, err := trimRequiredCategory(in.Category)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		date := trimOptional(in.Date)
		if date == "" {
			date = now().UTC().Format(isoDateLayout)
		}
		if err := finch.ValidateDate(date); err != nil {
			return errorResult(err.Error()), nil
		}
		desc := trimOptional(in.Desc)
		tags := trimOptional(in.Tags)
		recurring := trimOptional(in.Recurring)
		if err := finch.ValidateRecurring(recurring); err != nil {
			return errorResult(err.Error()), nil
		}
		input := finch.AddInput{
			Type:        in.Type,
			AmountCents: amountCents,
			Category:    category,
			Desc:        desc,
			Date:        date,
			Tags:        tags,
			Recurring:   recurring,
		}
		if err := store.Add(ctx, input); err != nil {
			return errorResult(err.Error()), nil
		}
		return jsonResult(map[string]any{
			"success":  true,
			"status":   "created",
			"date":     input.Date,
			"type":     input.Type,
			"amount":   finch.FormatAmount(input.AmountCents),
			"category": input.Category,
			"message":  "transaction added",
		})
	}
}

func makeListHandler(store Store) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if !PermissionFromContext(ctx).allowsRead() {
			return permissionDenied(toolListTransactions), nil
		}
		var in listInput
		if err := unmarshalArgs(req, &in); err != nil {
			return errorResult(err.Error()), nil
		}
		if err := finch.ValidateMonth(in.Month); err != nil {
			return errorResult(err.Error()), nil
		}
		filter := finch.ListFilter{
			Month:    trimOptional(in.Month),
			Category: trimOptional(in.Category),
		}
		if reqHasField(req, "limit") {
			if err := finch.ValidateLimit(in.Limit); err != nil {
				return errorResult(err.Error()), nil
			}
			filter.Limit = in.Limit
		}
		transactions, err := store.List(ctx, filter)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if transactions == nil {
			transactions = []finch.Transaction{}
		}
		return jsonResult(map[string]any{
			"transactions": transactions,
			"count":        len(transactions),
		})
	}
}

func makeSummaryHandler(store Store) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if !PermissionFromContext(ctx).allowsRead() {
			return permissionDenied(toolGetSummary), nil
		}
		var in summaryInput
		if err := unmarshalArgs(req, &in); err != nil {
			return errorResult(err.Error()), nil
		}
		if err := finch.ValidateMonth(in.Month); err != nil {
			return errorResult(err.Error()), nil
		}
		summary, err := store.Summary(ctx, trimOptional(in.Month))
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return jsonResult(summary)
	}
}

func makeEditHandler(store Store) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if denied := requireWrite(ctx, toolEditTransaction); denied != nil {
			return denied, nil
		}
		var in editInput
		if err := unmarshalArgs(req, &in); err != nil {
			return errorResult(err.Error()), nil
		}
		if in.ID <= 0 {
			return errorResult("id must be a positive integer"), nil
		}
		if !boolConfirmed(in.Confirm) {
			return errorResult("confirmation required: set confirm to true to edit a transaction"), nil
		}
		edit := finch.EditInput{ID: in.ID}
		if in.Amount != nil {
			cents, err := finch.ParseAmount(*in.Amount)
			if err != nil {
				return errorResult(err.Error()), nil
			}
			edit.AmountCents = &cents
		}
		if in.Category != nil {
			cat, err := trimRequiredCategory(*in.Category)
			if err != nil {
				return errorResult(err.Error()), nil
			}
			v := cat
			edit.Category = &v
		}
		if in.Desc != nil {
			v := trimOptional(*in.Desc)
			edit.Desc = &v
		}
		if in.Tags != nil {
			v := trimOptional(*in.Tags)
			edit.Tags = &v
		}
		if in.Recurring != nil {
			v := trimOptional(*in.Recurring)
			if err := finch.ValidateRecurring(v); err != nil {
				return errorResult(err.Error()), nil
			}
			edit.Recurring = &v
		}
		if err := finch.ValidateEditFields(edit); err != nil {
			return errorResult(err.Error()), nil
		}
		err := store.Update(ctx, edit)
		if errors.Is(err, finch.ErrTransactionNotFound) {
			return jsonResult(notFoundMessage(in.ID, "edit"))
		}
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return jsonResult(successMessage(in.ID, "updated", "transaction updated"))
	}
}

func makeDeleteHandler(store Store) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if denied := requireWrite(ctx, toolDeleteTransaction); denied != nil {
			return denied, nil
		}
		var in deleteInput
		if err := unmarshalArgs(req, &in); err != nil {
			return errorResult(err.Error()), nil
		}
		if in.ID <= 0 {
			return errorResult("id must be a positive integer"), nil
		}
		if !boolConfirmed(in.Confirm) {
			return errorResult("confirmation required: set confirm to true to delete a transaction"), nil
		}
		err := store.Delete(ctx, in.ID)
		if errors.Is(err, finch.ErrTransactionNotFound) {
			return jsonResult(notFoundMessage(in.ID, "delete"))
		}
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return jsonResult(successMessage(in.ID, "deleted", "transaction deleted"))
	}
}

// unmarshalArgs decodes the raw arguments from a tool call into the
// supplied target. It tolerates missing or empty arguments.
func unmarshalArgs(req *mcpsdk.CallToolRequest, target any) error {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return nil
	}
	return json.Unmarshal(req.Params.Arguments, target)
}

// reqHasField reports whether the named field was present in the original
// request arguments. We inspect the raw JSON to detect explicit presence,
// since the SDK decodes Arguments as json.RawMessage.
func reqHasField(req *mcpsdk.CallToolRequest, field string) bool {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(req.Params.Arguments, &probe); err != nil {
		return false
	}
	_, present := probe[field]
	return present
}

// boolConfirmed reports whether the supplied confirmation pointer is
// non-nil and points to true. nil, false, null, and non-boolean values
// are all treated as not confirmed.
func boolConfirmed(confirm *bool) bool {
	return confirm != nil && *confirm
}
