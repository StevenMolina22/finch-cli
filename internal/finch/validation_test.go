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
