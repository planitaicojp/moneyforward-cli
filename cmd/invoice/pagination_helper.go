package invoice

import (
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

// fetchAll fetches all pages from a paginated API endpoint.
// It calls fetchPage for each page, sleeping 400ms between requests for rate-limit compliance.
func fetchAll[T any](fetchPage func(page int) ([]T, *pagination.Result, error)) ([]T, error) {
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
		time.Sleep(400 * time.Millisecond)
	}
	return all, nil
}
