package invoice

import (
	"testing"
)

func TestResolveExcise(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"10", "ten_percent"},
		{"8", "eight_percent"},
		{"8r", "eight_percent_as_reduced_tax_rate"},
		{"5", "five_percent"},
		{"0", "untaxable"},
		{"exempt", "tax_exemption"},
		{"non", "non_taxable"},
		// Full names pass through unchanged.
		{"ten_percent", "ten_percent"},
		{"untaxable", "untaxable"},
		// Unknown value passes through unchanged.
		{"custom_value", "custom_value"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveExcise(tt.input)
			if got != tt.want {
				t.Errorf("resolveExcise(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
