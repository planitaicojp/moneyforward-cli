package cmdutil

import (
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

// rateLimitDelay is the sleep duration between paginated requests for rate-limit compliance.
const rateLimitDelay = 400 * time.Millisecond

// FetchAll fetches all pages from a paginated API endpoint.
// It calls fetchPage for each page, sleeping between requests for rate-limit compliance.
func FetchAll[T any](fetchPage func(page int) ([]T, *pagination.Result, error)) ([]T, error) {
	var all []T
	page := 1
	for {
		items, pg, err := fetchPage(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if pg == nil || page >= pg.TotalPages {
			break
		}
		page++
		time.Sleep(rateLimitDelay)
	}
	return all, nil
}
