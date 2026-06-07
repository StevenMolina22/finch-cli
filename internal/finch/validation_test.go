package finch

import "testing"

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{name: "whole dollars", input: "12", want: 1200},
		{name: "one fractional digit", input: "12.3", want: 1230},
		{name: "two fractional digits", input: "12.34", want: 1234},
		{name: "leading zeros", input: "001.05", want: 105},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAmount(tt.input)
			if err != nil {
				t.Fatalf("ParseAmount() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseAmount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseAmountRejectsInvalidValues(t *testing.T) {
	inputs := []string{"", "0", "-1", "+1", "1.999", "1.", ".50", "abc", "1.a"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseAmount(input); err == nil {
				t.Fatalf("ParseAmount(%q) expected error", input)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{cents: 0, want: "0.00"},
		{cents: 5, want: "0.05"},
		{cents: 1234, want: "12.34"},
		{cents: -987, want: "-9.87"},
	}

	for _, tt := range tests {
		if got := FormatAmount(tt.cents); got != tt.want {
			t.Fatalf("FormatAmount(%d) = %q, want %q", tt.cents, got, tt.want)
		}
	}
}

func TestValidateLimit(t *testing.T) {
	if err := ValidateLimit(5); err != nil {
		t.Fatalf("ValidateLimit(5) error = %v", err)
	}
	if err := ValidateLimit(0); err == nil {
		t.Fatal("ValidateLimit(0) expected error")
	}
	if err := ValidateLimit(-1); err == nil {
		t.Fatal("ValidateLimit(-1) expected error")
	}
}

func TestValidateRecurring(t *testing.T) {
	valid := []string{"", "monthly", "weekly", "yearly"}
	for _, v := range valid {
		t.Run("valid "+v, func(t *testing.T) {
			if err := ValidateRecurring(v); err != nil {
				t.Fatalf("ValidateRecurring(%q) error = %v", v, err)
			}
		})
	}

	invalid := []string{"daily", "annually", "biweekly"}
	for _, v := range invalid {
		t.Run("invalid "+v, func(t *testing.T) {
			if err := ValidateRecurring(v); err == nil {
				t.Fatalf("ValidateRecurring(%q) expected error", v)
			}
		})
	}
}

func TestValidateEditFields(t *testing.T) {
	if err := ValidateEditFields(EditInput{ID: 1, AmountCents: ptr[int64](1000)}); err != nil {
		t.Fatalf("ValidateEditFields with amount error = %v", err)
	}
	if err := ValidateEditFields(EditInput{ID: 1, Category: ptr("food")}); err != nil {
		t.Fatalf("ValidateEditFields with category error = %v", err)
	}
	if err := ValidateEditFields(EditInput{ID: 1}); err == nil {
		t.Fatal("ValidateEditFields with no changes expected error")
	}
}

func TestValidateImportRows(t *testing.T) {
	valid := []ImportRow{
		{Type: "income", Amount: "100.00", Category: "salary"},
		{Type: "expense", Amount: "25.50", Category: "food", Desc: "lunch", Date: "2026-05-01"},
		{Type: "income", Amount: "50", Category: "freelance", Tags: "work", Recurring: "monthly"},
	}
	if err := ValidateImportRows(valid); err != nil {
		t.Fatalf("ValidateImportRows valid rows error = %v", err)
	}

	invalid := []struct {
		name string
		rows []ImportRow
	}{
		{name: "invalid type", rows: []ImportRow{{Type: "transfer", Amount: "10", Category: "cat"}}},
		{name: "invalid amount", rows: []ImportRow{{Type: "income", Amount: "1.999", Category: "cat"}}},
		{name: "empty category", rows: []ImportRow{{Type: "income", Amount: "10", Category: ""}}},
		{name: "invalid date", rows: []ImportRow{{Type: "income", Amount: "10", Category: "cat", Date: "2026/05/01"}}},
		{name: "invalid recurring", rows: []ImportRow{{Type: "income", Amount: "10", Category: "cat", Recurring: "daily"}}},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateImportRows(tt.rows); err == nil {
				t.Fatal("ValidateImportRows expected error")
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestValidateMonth(t *testing.T) {
	valid := []string{"", "2026-05", "1999-12"}
	for _, month := range valid {
		t.Run("valid "+month, func(t *testing.T) {
			if err := ValidateMonth(month); err != nil {
				t.Fatalf("ValidateMonth(%q) error = %v", month, err)
			}
		})
	}

	invalid := []string{"2026-5", "2026-13", "May-2026", "202605", "2026-00"}
	for _, month := range invalid {
		t.Run("invalid "+month, func(t *testing.T) {
			if err := ValidateMonth(month); err == nil {
				t.Fatalf("ValidateMonth(%q) expected error", month)
			}
		})
	}
}
