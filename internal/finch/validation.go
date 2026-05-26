package finch

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
