package pagination_test

import (
	"testing"

	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

func TestParams_QueryString_Defaults(t *testing.T) {
	p := pagination.Params{Page: 1, PerPage: 25}
	got := p.QueryString()
	if got != "page=1&per_page=25" {
		t.Errorf("QueryString() = %q, want %q", got, "page=1&per_page=25")
	}
}

func TestParams_QueryString_CustomValues(t *testing.T) {
	p := pagination.Params{Page: 3, PerPage: 50}
	got := p.QueryString()
	if got != "page=3&per_page=50" {
		t.Errorf("QueryString() = %q, want %q", got, "page=3&per_page=50")
	}
}

func TestParams_QueryString_WithQuery(t *testing.T) {
	p := pagination.Params{Page: 1, PerPage: 25, Query: "test corp"}
	got := p.QueryString()
	if got != "page=1&per_page=25&q=test+corp" {
		t.Errorf("QueryString() = %q, want %q", got, "page=1&per_page=25&q=test+corp")
	}
}

func TestParams_QueryString_EmptyQuery(t *testing.T) {
	p := pagination.Params{Page: 1, PerPage: 25, Query: ""}
	got := p.QueryString()
	if got != "page=1&per_page=25" {
		t.Errorf("QueryString() = %q, want %q", got, "page=1&per_page=25")
	}
}

func TestDefaultParams(t *testing.T) {
	p := pagination.DefaultParams()
	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("PerPage = %d, want 25", p.PerPage)
	}
}
