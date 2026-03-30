package invoice

var exciseAliases = map[string]string{
	"10":     "ten_percent",
	"8":      "eight_percent",
	"8r":     "eight_percent_as_reduced_tax_rate",
	"5":      "five_percent",
	"0":      "untaxable",
	"exempt": "tax_exemption",
	"non":    "non_taxable",
}

// resolveExcise maps short aliases to full excise type names.
// Unknown values pass through unchanged.
func resolveExcise(input string) string {
	if full, ok := exciseAliases[input]; ok {
		return full
	}
	return input
}
