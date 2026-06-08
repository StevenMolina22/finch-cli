package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"finch/internal/finch"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "finch-mcp"
const serverVersion = "0.1.0"

const (
	toolAddTransaction    = "finch_add_transaction"
	toolListTransactions  = "finch_list_transactions"
	toolGetSummary        = "finch_get_summary"
	toolEditTransaction   = "finch_edit_transaction"
	toolDeleteTransaction = "finch_delete_transaction"
)

// NewServer constructs an MCP server with all Finch transaction tools
// registered. The store is invoked directly from tool handlers; tests may
// substitute a fake implementation.
func NewServer(store Store) *mcpsdk.Server {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: serverName, Version: serverVersion}, nil)
	registerTools(s, store)
	return s
}

// toolSchema returns a JSON schema object that the SDK accepts as a
// Tool.InputSchema. The argument is a map description of the expected
// properties; it is encoded as a JSON object before being attached to the
// tool definition.
func toolSchema(properties map[string]any, required []string) json.RawMessage {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	data, err := json.Marshal(schema)
	if err != nil {
		// Falling back to a basic object schema keeps the server usable
		// even if a programming error produces an unencodable schema; the
		// tool would still register, but its input validation would be
		// degraded.
		data = []byte(`{"type":"object"}`)
	}
	return data
}

func schemaString(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func schemaStringWithDefault(description, def string) map[string]any {
	m := schemaString(description)
	if def != "" {
		m["default"] = def
	}
	return m
}

func schemaInt(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description, "minimum": 1}
}

func schemaBool(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func registerTools(s *mcpsdk.Server, store Store) {
	nowFn := time.Now

	s.AddTool(&mcpsdk.Tool{
		Name:        toolAddTransaction,
		Description: "Add an income or expense transaction. If date is omitted, the current UTC date is used.",
		InputSchema: toolSchema(map[string]any{
			"type":        schemaString(`Transaction type. Must be "income" or "expense".`),
			"amount":      schemaString(`Amount as a positive decimal string, e.g. "12.34".`),
			"category":    schemaString("Category label, e.g. \"groceries\"."),
			"desc":        schemaString("Optional description."),
			"date":        schemaStringWithDefault("Optional date in YYYY-MM-DD. Defaults to today (UTC).", ""),
			"tags":        schemaString("Optional comma-separated tags."),
			"recurring":   schemaString(`Optional recurrence: "monthly", "weekly", or "yearly".`),
			"confirm":     schemaBool("Set true to confirm. Required for destructive tools, optional for add."),
		}, []string{"type", "amount", "category"}),
	}, makeAddHandler(store, nowFn))

	s.AddTool(&mcpsdk.Tool{
		Name:        toolListTransactions,
		Description: "List transactions with optional month, category, and limit filters.",
		InputSchema: toolSchema(map[string]any{
			"month":    schemaString("Optional month filter in YYYY-MM."),
			"category": schemaString("Optional exact category filter."),
			"limit":    schemaInt("Optional maximum number of transactions to return."),
		}, nil),
	}, makeListHandler(store))

	s.AddTool(&mcpsdk.Tool{
		Name:        toolGetSummary,
		Description: "Return income, expense, net, and top categories. Optional month filter in YYYY-MM.",
		InputSchema: toolSchema(map[string]any{
			"month": schemaString("Optional month filter in YYYY-MM."),
		}, nil),
	}, makeSummaryHandler(store))

	s.AddTool(&mcpsdk.Tool{
		Name:        toolEditTransaction,
		Description: "Edit a transaction. Requires confirm=true and at least one editable field.",
		InputSchema: toolSchema(map[string]any{
			"id":        schemaInt("Transaction id to edit."),
			"amount":    schemaString("New amount as a positive decimal string."),
			"category":  schemaString("New category label."),
			"desc":      schemaString("New description."),
			"tags":      schemaString("New comma-separated tags."),
			"recurring": schemaString(`New recurrence: "monthly", "weekly", "yearly", or empty to clear.`),
			"confirm":   schemaBool("Must be true to apply the edit."),
		}, []string{"id", "confirm"}),
	}, makeEditHandler(store))

	s.AddTool(&mcpsdk.Tool{
		Name:        toolDeleteTransaction,
		Description: "Delete a transaction. Requires confirm=true.",
		InputSchema: toolSchema(map[string]any{
			"id":      schemaInt("Transaction id to delete."),
			"confirm": schemaBool("Must be true to delete."),
		}, []string{"id", "confirm"}),
	}, makeDeleteHandler(store))
}

// jsonResult builds a successful CallToolResult with both a text content
// block and structured content. The text block contains a JSON rendering
// of the value for clients that only inspect content.
func jsonResult(v any) (*mcpsdk.CallToolResult, error) {
	text, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode result: %w", err)
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(text)},
		},
		StructuredContent: v,
	}, nil
}

// errorResult returns a CallToolResult that signals a tool-level error
// without leaking storage internals. The supplied message is included in
// the text content.
func errorResult(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}

// permissionDenied returns a CallToolResult for an unauthorized caller.
// The message does not reference tokens or credentials.
func permissionDenied(tool string) *mcpsdk.CallToolResult {
	return errorResult(fmt.Sprintf("permission denied: tool %q is not permitted for the authenticated token class", tool))
}

// requireWrite returns a permission-denied result if the context does
// not carry write permission. nil means the caller may proceed.
func requireWrite(ctx context.Context, tool string) *mcpsdk.CallToolResult {
	if !PermissionFromContext(ctx).allowsWrite() {
		return permissionDenied(tool)
	}
	return nil
}

// finch.Store adapter sanity check during compile: confirm that the
// adapter satisfies the Store interface and that finch.ErrTransactionNotFound
// is the sentinel used by handlers.
var (
	_ Store              = StoreAdapter{}
	_ = finch.ErrTransactionNotFound
	_ = errors.Is
)
