package pagination

import (
	"fmt"
	"net/url"
)

type Params struct {
	Page    int
	PerPage int
	Query   string
}

func DefaultParams() Params {
	return Params{Page: 1, PerPage: 25}
}

func (p Params) QueryString() string {
	v := url.Values{}
	v.Set("page", fmt.Sprintf("%d", p.Page))
	v.Set("per_page", fmt.Sprintf("%d", p.PerPage))
	if p.Query != "" {
		v.Set("q", p.Query)
	}
	return v.Encode()
}

type Result struct {
	TotalCount  int `json:"total_count"`
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
}
