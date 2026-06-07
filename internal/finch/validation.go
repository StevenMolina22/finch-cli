package finch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrTransactionNotFound is returned by store operations (Delete, Update)
// when the targeted transaction does not exist. Callers can use errors.Is
// to detect this without relying on error message text.
var ErrTransactionNotFound = errors.New("transaction not found")

func ValidateType(typ string) error {
	if typ != TypeIncome && typ != TypeExpense {
		return fmt.Errorf("type must be %q or %q", TypeIncome, TypeExpense)
	}
	return nil
}

func ParseAmount(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("amount is required")
	}
	if strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, fmt.Errorf("amount must be positive")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("amount must be a positive decimal value")
	}
	if !allDigits(parts[0]) {
		return 0, fmt.Errorf("amount must be a positive decimal value")
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 2 || !allDigits(fraction) {
			return 0, fmt.Errorf("amount must have no more than two fractional digits")
		}
	}
	for len(fraction) < 2 {
		fraction += "0"
	}

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount is too large")
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount must be a positive decimal value")
	}
	amount := dollars*100 + cents
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	return amount, nil
}

func FormatAmount(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func ValidateMonth(month string) error {
	if month == "" {
		return nil
	}
	if len(month) != len("2006-01") {
		return fmt.Errorf("month must use YYYY-MM format")
	}
	parsed, err := time.Parse("2006-01", month)
	if err != nil || parsed.Format("2006-01") != month {
		return fmt.Errorf("month must use YYYY-MM format")
	}
	return nil
}

// ValidateDate verifies that value is a calendar date formatted as YYYY-MM-DD.
// An empty string is considered valid (callers default to the current date).
func ValidateDate(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return fmt.Errorf("date must use YYYY-MM-DD format")
	}
	return nil
}

func ValidateLimit(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("limit must be a positive integer")
	}
	return nil
}

func ValidateRecurring(value string) error {
	if value == "" {
		return nil
	}
	if value != "monthly" && value != "weekly" && value != "yearly" {
		return fmt.Errorf("recurring must be %q, %q, or %q", "monthly", "weekly", "yearly")
	}
	return nil
}

func ValidateEditFields(input EditInput) error {
	if input.AmountCents == nil && input.Category == nil && input.Desc == nil && input.Tags == nil && input.Recurring == nil {
		return fmt.Errorf("at least one field must be changed")
	}
	return nil
}

func ValidateImportRows(rows []ImportRow) error {
	for i, row := range rows {
		if err := ValidateType(row.Type); err != nil {
			return fmt.Errorf("row %d: invalid type: %w", i+1, err)
		}
		if _, err := ParseAmount(row.Amount); err != nil {
			return fmt.Errorf("row %d: invalid amount: %w", i+1, err)
		}
		if strings.TrimSpace(row.Category) == "" {
			return fmt.Errorf("row %d: category is required", i+1)
		}
		if row.Date != "" {
			if _, err := time.Parse("2006-01-02", row.Date); err != nil {
				return fmt.Errorf("row %d: invalid date %q, expected YYYY-MM-DD", i+1, row.Date)
			}
		}
		if err := ValidateRecurring(strings.TrimSpace(row.Recurring)); err != nil {
			return fmt.Errorf("row %d: %w", i+1, err)
		}
	}
	return nil
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
