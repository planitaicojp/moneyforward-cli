package cmdutil

import "time"

// FetchAllExpense fetches all pages from an Expense API endpoint using cursor-based pagination.
func FetchAllExpense[T any](fetchPage func(page int) ([]T, bool, error)) ([]T, error) {
	var all []T
	page := 1
	for {
		items, hasNext, err := fetchPage(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !hasNext {
			break
		}
		page++
		time.Sleep(400 * time.Millisecond)
	}
	return all, nil
}
