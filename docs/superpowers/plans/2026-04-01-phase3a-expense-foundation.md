# Phase 3a: Expense Foundation + Master Data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add ExpenseService with offices and master data (departments, projects, categories, taxes, positions) list/show commands.

**Architecture:** ExpenseService holds v1/v2 base URLs. All Expense endpoints require `{office_id}` in the path, resolved via `resolveOfficeID()` (flag → auto-detect). Expense API uses cursor-based pagination (`next`/`prev` links) with resource-named list keys (e.g., `{"depts": [...], "next": "...", "prev": "..."}`), unlike Invoice's `{"data": [...], "pagination": {...}}`.

**Tech Stack:** Go 1.26, cobra, httptest

---

### Task 1: Extract fetchAll to cmd/cmdutil/pagination.go

**Files:**
- Create: `cmd/cmdutil/pagination.go`
- Modify: `cmd/invoice/pagination_helper.go` (delete)
- Modify: `cmd/invoice/partners.go` (update import)
- Modify: `cmd/invoice/items.go` (update import)
- Modify: `cmd/invoice/billings.go` (update import)
- Modify: `cmd/invoice/quotes.go` (update import)

- [ ] **Step 1: Create `cmd/cmdutil/pagination.go`**

```go
package cmdutil

import (
	"time"

	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

// FetchAll fetches all pages from a paginated API endpoint.
// It calls fetchPage for each page, sleeping 400ms between requests for rate-limit compliance.
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
		time.Sleep(400 * time.Millisecond)
	}
	return all, nil
}
```

- [ ] **Step 2: Delete `cmd/invoice/pagination_helper.go`**

```bash
rm cmd/invoice/pagination_helper.go
```

- [ ] **Step 3: Update all `fetchAll` calls in `cmd/invoice/` to `cmdutil.FetchAll`**

In each file (`partners.go`, `items.go`, `billings.go`, `quotes.go`), replace:
- `fetchAll(func(page int)` → `cmdutil.FetchAll(func(page int)`
- Add `"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"` import if not present

Example change in `partners.go` line 147:
```go
		allPartners, err := cmdutil.FetchAll(func(page int) ([]model.Partner, *pagination.Result, error) {
```

- [ ] **Step 4: Run tests to verify nothing broke**

```bash
go build ./...
go test ./cmd/invoice/... ./cmd/cmdutil/...
```

Expected: all pass, no compilation errors.

- [ ] **Step 5: Commit**

```bash
git add cmd/cmdutil/pagination.go cmd/invoice/
git commit -m "Extract fetchAll to cmd/cmdutil for reuse across services"
```

---

### Task 2: Expense data models

**Files:**
- Create: `internal/model/expense.go`

- [ ] **Step 1: Create `internal/model/expense.go` with all Phase 3a types**

```go
package model

// --- Expense Office ---

// ExpenseOffice is an office (事業者) from the Expense API.
// Named ExpenseOffice to avoid collision with Invoice's Office type.
type ExpenseOffice struct {
	ID                 string `json:"id"`
	IdentificationCode string `json:"identification_code,omitempty"`
	OfficeTypeID       int    `json:"office_type_id,omitempty"` // 1:個人, 2:法人
	Name               string `json:"name"`
}

// --- Dept (部門) ---

type Dept struct {
	ID              string `json:"id"`
	Code            string `json:"code,omitempty"`
	Name            string `json:"name"`
	DispOrder       int    `json:"disp_order,omitempty"`
	IsActive        bool   `json:"is_active"`
	ParentID        string `json:"parent_id,omitempty"`
	AncestryDepth   int    `json:"ancestry_depth,omitempty"`
	ExJournalDeptID string `json:"ex_journal_dept_id,omitempty"`
	RootID          string `json:"root_id,omitempty"`
}

// --- Project (プロジェクト) ---

type ExpenseProject struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	DispOrder int    `json:"disp_order,omitempty"`
	ParentID  string `json:"parent_id,omitempty"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
}

// --- ExItem (経費科目) ---

type ExItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Code            string `json:"code,omitempty"`
	IsActive        bool   `json:"is_active"`
	ItemID          string `json:"item_id,omitempty"`
	SubItemID       string `json:"sub_item_id,omitempty"`
	DefaultExciseID string `json:"default_excise_id,omitempty"`
}

// --- Excise (税区分) ---

type ExpenseExcise struct {
	ID       string  `json:"id"`
	LongName string  `json:"long_name"`
	Code     string  `json:"code,omitempty"`
	Rate     float64 `json:"rate"`
}

// --- Position (役職) ---

type Position struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsAuthorizer bool   `json:"is_authorizer"`
	Priority     int    `json:"priority"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/model/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/model/expense.go
git commit -m "Add Expense API data models for Phase 3a"
```

---

### Task 3: ExpenseService foundation + ListOffices

**Files:**
- Create: `internal/api/expense.go`
- Create: `internal/api/expense_test.go`

The Expense API uses a different pagination format than Invoice:
- List key varies per resource (e.g., `"offices"`, `"depts"`, `"ex_items"`)
- Pagination uses `next`/`prev` URL links (cursor-based), not `total_count`/`total_pages`

We need a different response wrapper and a modified `fetchAll` strategy for Expense.

- [ ] **Step 1: Write test for ExpenseService.ListOffices**

Create `internal/api/expense_test.go`:

```go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
)

func newTestExpenseService(t *testing.T, handler http.HandlerFunc) *api.ExpenseService {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := api.NewWithToken("test-token", "1.0.0", false)
	return api.NewExpenseService(client, srv.URL+"/api/external/v1", srv.URL+"/api/external/v2")
}

func TestExpenseService_ListOffices(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"offices": []model.ExpenseOffice{
				{ID: "o1", Name: "Test Office", OfficeTypeID: 2},
			},
			"next": nil,
			"prev": nil,
		})
	})

	offices, hasNext, err := svc.ListOffices(1)
	if err != nil {
		t.Fatalf("ListOffices() error: %v", err)
	}
	if len(offices) != 1 {
		t.Fatalf("got %d offices, want 1", len(offices))
	}
	if offices[0].ID != "o1" {
		t.Errorf("offices[0].ID = %q, want %q", offices[0].ID, "o1")
	}
	if offices[0].Name != "Test Office" {
		t.Errorf("offices[0].Name = %q, want %q", offices[0].Name, "Test Office")
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_ListOffices_Pagination(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "" || page == "1" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offices": []model.ExpenseOffice{{ID: "o1", Name: "Office 1"}},
				"next":    "/api/external/v1/offices?page=2",
				"prev":    nil,
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offices": []model.ExpenseOffice{{ID: "o2", Name: "Office 2"}},
				"next":    nil,
				"prev":    "/api/external/v1/offices?page=1",
			})
		}
	})

	offices1, hasNext1, err := svc.ListOffices(1)
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if !hasNext1 {
		t.Error("page 1: hasNext = false, want true")
	}
	if offices1[0].ID != "o1" {
		t.Errorf("page 1: ID = %q, want %q", offices1[0].ID, "o1")
	}

	offices2, hasNext2, err := svc.ListOffices(2)
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if hasNext2 {
		t.Error("page 2: hasNext = true, want false")
	}
	if offices2[0].ID != "o2" {
		t.Errorf("page 2: ID = %q, want %q", offices2[0].ID, "o2")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/api/ -run TestExpenseService -v
```

Expected: FAIL — `NewExpenseService` not defined.

- [ ] **Step 3: Implement ExpenseService with ListOffices**

Create `internal/api/expense.go`:

```go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
)

const (
	expenseBaseURL   = "https://expense.moneyforward.com/api/external/v1"
	expenseBaseURLV2 = "https://expense.moneyforward.com/api/external/v2"
)

// ExpenseService is an API client for the Money Forward Cloud Expense API.
type ExpenseService struct {
	client *Client
	base   string // v1 base URL
	baseV2 string // v2 base URL (for office_members, /me)
}

// NewExpenseService creates an ExpenseService with explicit base URLs.
func NewExpenseService(client *Client, base, baseV2 string) *ExpenseService {
	return &ExpenseService{client: client, base: base, baseV2: baseV2}
}

// NewExpenseServiceDefault creates an ExpenseService with production base URLs.
func NewExpenseServiceDefault(client *Client) *ExpenseService {
	return NewExpenseService(client, expenseBaseURL, expenseBaseURLV2)
}

// expenseListResponse is a generic helper for decoding Expense API list responses.
// The Expense API returns lists under a resource-specific key (e.g. "offices", "depts")
// with cursor-based pagination via "next"/"prev" URL strings.
type expenseListResponse[T any] struct {
	Items []T
	Next  *string
	Prev  *string
}

// decodeExpenseList decodes a list response where the array is under the given key.
func decodeExpenseList[T any](data []byte, key string) (*expenseListResponse[T], error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var items []T
	if itemsRaw, ok := raw[key]; ok {
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil, fmt.Errorf("decoding %s: %w", key, err)
		}
	}
	resp := &expenseListResponse[T]{Items: items}
	if nextRaw, ok := raw["next"]; ok {
		var next *string
		if err := json.Unmarshal(nextRaw, &next); err == nil {
			resp.Next = next
		}
	}
	if prevRaw, ok := raw["prev"]; ok {
		var prev *string
		if err := json.Unmarshal(prevRaw, &prev); err == nil {
			resp.Prev = prev
		}
	}
	return resp, nil
}

// doExpenseList performs a GET request and decodes the list response.
func doExpenseList[T any](s *ExpenseService, url, key string) ([]T, bool, error) {
	body, err := s.client.DoRaw(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := decodeExpenseList[T](body, key)
	if err != nil {
		return nil, false, err
	}
	hasNext := resp.Next != nil && *resp.Next != ""
	return resp.Items, hasNext, nil
}

// --- Offices ---

func (s *ExpenseService) ListOffices(page int) ([]model.ExpenseOffice, bool, error) {
	u := fmt.Sprintf("%s/offices?page=%s", s.base, strconv.Itoa(page))
	items, hasNext, err := doExpenseList[model.ExpenseOffice](s, u, "offices")
	if err != nil {
		return nil, false, fmt.Errorf("listing offices: %w", err)
	}
	return items, hasNext, nil
}
```

- [ ] **Step 4: Add `DoRaw` method to Client**

The existing `Client` has `DoJSON` (which decodes into a target struct) but we need `DoRaw` to get raw bytes for our custom decoding. Add to `internal/api/client.go`:

```go
// DoRaw executes a JSON API request and returns the raw response body.
func (c *Client) DoRaw(method, url string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}
```

Check if `parseAPIError` already exists in client.go. If not, check how `DoJSON` handles errors and reuse the same pattern. The implementation should match what `DoJSON` does for error responses.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/api/ -run TestExpenseService -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/expense.go internal/api/expense_test.go internal/api/client.go
git commit -m "Add ExpenseService with ListOffices and cursor-based pagination"
```

---

### Task 4: Master data API methods (depts, projects, ex_items, excises, positions)

**Files:**
- Modify: `internal/api/expense.go`
- Modify: `internal/api/expense_test.go`

- [ ] **Step 1: Write tests for ListDepts and GetDept**

Add to `internal/api/expense_test.go`:

```go
func TestExpenseService_ListDepts(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/depts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"depts": []model.Dept{
				{ID: "d1", Name: "Engineering", Code: "ENG", IsActive: true},
			},
			"next": nil,
			"prev": nil,
		})
	})

	depts, hasNext, err := svc.ListDepts("office1", 1, "")
	if err != nil {
		t.Fatalf("ListDepts() error: %v", err)
	}
	if len(depts) != 1 {
		t.Fatalf("got %d depts, want 1", len(depts))
	}
	if depts[0].Name != "Engineering" {
		t.Errorf("depts[0].Name = %q, want %q", depts[0].Name, "Engineering")
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_GetDept(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/depts/d1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(model.Dept{ID: "d1", Name: "Engineering", IsActive: true})
	})

	dept, err := svc.GetDept("office1", "d1")
	if err != nil {
		t.Fatalf("GetDept() error: %v", err)
	}
	if dept.Name != "Engineering" {
		t.Errorf("dept.Name = %q, want %q", dept.Name, "Engineering")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/api/ -run TestExpenseService_ListDepts -v
go test ./internal/api/ -run TestExpenseService_GetDept -v
```

Expected: FAIL — methods not defined.

- [ ] **Step 3: Implement all master data API methods**

Add to `internal/api/expense.go`:

```go
// --- Depts (部門) ---

func (s *ExpenseService) ListDepts(officeID string, page int, keyword string) ([]model.Dept, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/depts?page=%d", s.base, officeID, page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.Dept](s, u, "depts")
	if err != nil {
		return nil, false, fmt.Errorf("listing depts: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetDept(officeID, id string) (*model.Dept, error) {
	var dept model.Dept
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/depts/%s", s.base, officeID, id), nil, &dept)
	if err != nil {
		return nil, fmt.Errorf("getting dept: %w", err)
	}
	return &dept, nil
}

// --- Projects ---

func (s *ExpenseService) ListProjects(officeID string, page int, keyword string) ([]model.ExpenseProject, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/projects?page=%d", s.base, officeID, page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.ExpenseProject](s, u, "projects")
	if err != nil {
		return nil, false, fmt.Errorf("listing projects: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetProject(officeID, id string) (*model.ExpenseProject, error) {
	var project model.ExpenseProject
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/projects/%s", s.base, officeID, id), nil, &project)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return &project, nil
}

// --- ExItems (経費科目) ---

func (s *ExpenseService) ListExItems(officeID string, page int, keyword string) ([]model.ExItem, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/ex_items?page=%d", s.base, officeID, page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.ExItem](s, u, "ex_items")
	if err != nil {
		return nil, false, fmt.Errorf("listing ex_items: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetExItem(officeID, id string) (*model.ExItem, error) {
	var item model.ExItem
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/ex_items/%s", s.base, officeID, id), nil, &item)
	if err != nil {
		return nil, fmt.Errorf("getting ex_item: %w", err)
	}
	return &item, nil
}

// --- Excises (税区分) ---

func (s *ExpenseService) ListExcises(officeID string, page int) ([]model.ExpenseExcise, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/excises?page=%d", s.base, officeID, page)
	items, hasNext, err := doExpenseList[model.ExpenseExcise](s, u, "excises")
	if err != nil {
		return nil, false, fmt.Errorf("listing excises: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetExcise(officeID, id string) (*model.ExpenseExcise, error) {
	var excise model.ExpenseExcise
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/excises/%s", s.base, officeID, id), nil, &excise)
	if err != nil {
		return nil, fmt.Errorf("getting excise: %w", err)
	}
	return &excise, nil
}

// --- Positions (役職) ---

func (s *ExpenseService) ListPositions(officeID string, page int) ([]model.Position, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/positions?page=%d", s.base, officeID, page)
	items, hasNext, err := doExpenseList[model.Position](s, u, "positions")
	if err != nil {
		return nil, false, fmt.Errorf("listing positions: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetPosition(officeID, id string) (*model.Position, error) {
	var pos model.Position
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/positions/%s", s.base, officeID, id), nil, &pos)
	if err != nil {
		return nil, fmt.Errorf("getting position: %w", err)
	}
	return &pos, nil
}
```

Add `"net/url"` to the import block in `expense.go`.

- [ ] **Step 4: Write remaining tests for projects, ex_items, excises, positions**

Add to `expense_test.go` — follow the same pattern as ListDepts/GetDept tests above:

```go
func TestExpenseService_ListProjects(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if kw := r.URL.Query().Get("search_keyword"); kw != "test" {
			t.Errorf("search_keyword = %q, want %q", kw, "test")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"projects": []model.ExpenseProject{
				{ID: "p1", Name: "Project Alpha", Code: "PA", IsActive: true},
			},
			"next": nil,
		})
	})

	projects, _, err := svc.ListProjects("office1", 1, "test")
	if err != nil {
		t.Fatalf("ListProjects() error: %v", err)
	}
	if len(projects) != 1 || projects[0].Code != "PA" {
		t.Errorf("unexpected projects: %+v", projects)
	}
}

func TestExpenseService_ListExItems(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/ex_items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ex_items": []model.ExItem{
				{ID: "ei1", Name: "Travel", Code: "TRV", IsActive: true},
			},
			"next": nil,
		})
	})

	items, _, err := svc.ListExItems("office1", 1, "")
	if err != nil {
		t.Fatalf("ListExItems() error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Travel" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestExpenseService_ListExcises(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/excises" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"excises": []model.ExpenseExcise{
				{ID: "ex1", LongName: "10%", Rate: 0.1},
			},
			"next": nil,
		})
	})

	excises, _, err := svc.ListExcises("office1", 1)
	if err != nil {
		t.Fatalf("ListExcises() error: %v", err)
	}
	if len(excises) != 1 || excises[0].LongName != "10%" {
		t.Errorf("unexpected excises: %+v", excises)
	}
}

func TestExpenseService_ListPositions(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/office1/positions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"positions": []model.Position{
				{ID: "pos1", Name: "Manager", IsAuthorizer: true, Priority: 1},
			},
			"next": nil,
		})
	})

	positions, _, err := svc.ListPositions("office1", 1)
	if err != nil {
		t.Fatalf("ListPositions() error: %v", err)
	}
	if len(positions) != 1 || positions[0].Name != "Manager" {
		t.Errorf("unexpected positions: %+v", positions)
	}
}
```

- [ ] **Step 5: Run all Expense tests**

```bash
go test ./internal/api/ -run TestExpenseService -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/expense.go internal/api/expense_test.go
git commit -m "Add master data API methods: depts, projects, ex_items, excises, positions"
```

---

### Task 5: FetchAllExpense helper for cursor-based pagination

**Files:**
- Create: `cmd/cmdutil/pagination_expense.go`

The Expense API uses `next`/`prev` links, not `total_pages`. We need a separate `FetchAllExpense` that keeps incrementing page until `hasNext` is false.

- [ ] **Step 1: Create `cmd/cmdutil/pagination_expense.go`**

```go
package cmdutil

import "time"

// FetchAllExpense fetches all pages from an Expense API endpoint using cursor-based pagination.
// fetchPage returns the items for a page and whether there is a next page.
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/cmdutil/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/cmdutil/pagination_expense.go
git commit -m "Add FetchAllExpense for cursor-based Expense API pagination"
```

---

### Task 6: Root expense command + newExpenseService + resolveOfficeID

**Files:**
- Create: `cmd/expense/expense.go`
- Modify: `cmd/root.go` (register expense command)

- [ ] **Step 1: Create `cmd/expense/expense.go`**

```go
package expense

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
)

var officeIDFlag string

// ExpenseCmd is the root command for expense subcommands.
var ExpenseCmd = &cobra.Command{
	Use:   "expense",
	Short: "Money Forward Cloud Expense",
	Long:  "Commands for Money Forward Cloud Expense API",
}

func init() {
	ExpenseCmd.PersistentFlags().StringVar(&officeIDFlag, "office-id", "", "office ID (auto-detected if only one office)")

	ExpenseCmd.AddCommand(officesCmd)
	ExpenseCmd.AddCommand(departmentsCmd)
	ExpenseCmd.AddCommand(projectsCmd)
	ExpenseCmd.AddCommand(categoriesCmd)
	ExpenseCmd.AddCommand(taxesCmd)
	ExpenseCmd.AddCommand(positionsCmd)
}

func newExpenseService(cmd *cobra.Command) (*api.ExpenseService, error) {
	profile := cmdutil.GetProfile(cmd)
	token, err := cmdutil.EnsureValidToken(profile, api.Services["expense"])
	if err != nil {
		return nil, err
	}
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	version := cmd.Root().Version
	if version == "" {
		version = "dev"
	}
	client := api.NewWithToken(token, version, verbose)
	return api.NewExpenseServiceDefault(client), nil
}

// resolveOfficeID resolves the office ID from:
// 1. --office-id flag
// 2. Auto-detect: GET /offices, use if exactly 1, error if multiple.
func resolveOfficeID(cmd *cobra.Command, svc *api.ExpenseService) (string, error) {
	if officeIDFlag != "" {
		return officeIDFlag, nil
	}

	offices, _, err := svc.ListOffices(1)
	if err != nil {
		return "", fmt.Errorf("auto-detecting office: %w", err)
	}

	switch len(offices) {
	case 0:
		return "", fmt.Errorf("no offices found for this account")
	case 1:
		return offices[0].ID, nil
	default:
		msg := "multiple offices found; specify --office-id:\n"
		for _, o := range offices {
			msg += fmt.Sprintf("  %s  %s\n", o.ID, o.Name)
		}
		return "", fmt.Errorf(msg)
	}
}
```

- [ ] **Step 2: Register in `cmd/root.go`**

Add import and AddCommand:

```go
import (
	// ... existing imports ...
	"github.com/planitaicojp/moneyforward-cli/cmd/expense"
)

func init() {
	// ... existing commands ...
	rootCmd.AddCommand(expense.ExpenseCmd)
}
```

- [ ] **Step 3: Verify it compiles (will fail until subcommand files exist)**

We'll create stub files in the next tasks. For now, comment out the AddCommand lines for subcommands that don't exist yet in `expense.go`'s `init()` — or proceed to create all command files before compiling.

---

### Task 7: offices list command

**Files:**
- Create: `cmd/expense/offices.go`

- [ ] **Step 1: Create `cmd/expense/offices.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	officesListPage int
	officesListAll  bool
)

var officesCmd = &cobra.Command{
	Use:   "offices",
	Short: "Office operations",
}

var officesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List offices",
	RunE:  runOfficesList,
}

func init() {
	officesListCmd.Flags().IntVar(&officesListPage, "page", 1, "page number")
	officesListCmd.Flags().BoolVar(&officesListAll, "all", false, "fetch all pages")

	officesCmd.AddCommand(officesListCmd)
}

func runOfficesList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if officesListAll {
		allOffices, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseOffice, bool, error) {
			return svc.ListOffices(page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"offices": allOffices})
		}
		return f.Format(os.Stdout, allOffices)
	}

	offices, _, err := svc.ListOffices(officesListPage)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"offices": offices})
	}
	return f.Format(os.Stdout, offices)
}
```

Add the missing import for model:
```go
import (
	// ...
	"github.com/planitaicojp/moneyforward-cli/internal/model"
)
```

Note: `offices` doesn't need `resolveOfficeID()` since it's the command used to discover office IDs.

---

### Task 8: departments list/show command

**Files:**
- Create: `cmd/expense/departments.go`

- [ ] **Step 1: Create `cmd/expense/departments.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	deptsListPage    int
	deptsListAll     bool
	deptsListKeyword string
)

var departmentsCmd = &cobra.Command{
	Use:   "departments",
	Short: "Department operations (API: depts)",
}

var deptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List departments",
	RunE:  runDeptsList,
}

var deptsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show department details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runDeptsShow,
}

func init() {
	deptsListCmd.Flags().IntVar(&deptsListPage, "page", 1, "page number")
	deptsListCmd.Flags().BoolVar(&deptsListAll, "all", false, "fetch all pages")
	deptsListCmd.Flags().StringVar(&deptsListKeyword, "keyword", "", "search keyword (max 50 chars)")

	departmentsCmd.AddCommand(deptsListCmd)
	departmentsCmd.AddCommand(deptsShowCmd)
}

func runDeptsList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if deptsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.Dept, bool, error) {
			return svc.ListDepts(oid, page, deptsListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"depts": all})
		}
		return f.Format(os.Stdout, all)
	}

	depts, _, err := svc.ListDepts(oid, deptsListPage, deptsListKeyword)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"depts": depts})
	}
	return f.Format(os.Stdout, depts)
}

func runDeptsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	dept, err := svc.GetDept(oid, args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, dept)
}
```

---

### Task 9: projects list/show command

**Files:**
- Create: `cmd/expense/projects.go`

- [ ] **Step 1: Create `cmd/expense/projects.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	projectsListPage    int
	projectsListAll     bool
	projectsListKeyword string
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Project operations",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE:  runProjectsList,
}

var projectsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show project details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runProjectsShow,
}

func init() {
	projectsListCmd.Flags().IntVar(&projectsListPage, "page", 1, "page number")
	projectsListCmd.Flags().BoolVar(&projectsListAll, "all", false, "fetch all pages")
	projectsListCmd.Flags().StringVar(&projectsListKeyword, "keyword", "", "search keyword")

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsShowCmd)
}

func runProjectsList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if projectsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseProject, bool, error) {
			return svc.ListProjects(oid, page, projectsListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"projects": all})
		}
		return f.Format(os.Stdout, all)
	}

	projects, _, err := svc.ListProjects(oid, projectsListPage, projectsListKeyword)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"projects": projects})
	}
	return f.Format(os.Stdout, projects)
}

func runProjectsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	project, err := svc.GetProject(oid, args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, project)
}
```

---

### Task 10: categories list/show command

**Files:**
- Create: `cmd/expense/categories.go`

- [ ] **Step 1: Create `cmd/expense/categories.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	categoriesListPage    int
	categoriesListAll     bool
	categoriesListKeyword string
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Expense category operations (API: ex_items)",
}

var categoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List expense categories",
	RunE:  runCategoriesList,
}

var categoriesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show expense category details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runCategoriesShow,
}

func init() {
	categoriesListCmd.Flags().IntVar(&categoriesListPage, "page", 1, "page number")
	categoriesListCmd.Flags().BoolVar(&categoriesListAll, "all", false, "fetch all pages")
	categoriesListCmd.Flags().StringVar(&categoriesListKeyword, "keyword", "", "search keyword")

	categoriesCmd.AddCommand(categoriesListCmd)
	categoriesCmd.AddCommand(categoriesShowCmd)
}

func runCategoriesList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if categoriesListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExItem, bool, error) {
			return svc.ListExItems(oid, page, categoriesListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"ex_items": all})
		}
		return f.Format(os.Stdout, all)
	}

	items, _, err := svc.ListExItems(oid, categoriesListPage, categoriesListKeyword)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"ex_items": items})
	}
	return f.Format(os.Stdout, items)
}

func runCategoriesShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	item, err := svc.GetExItem(oid, args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}
```

---

### Task 11: taxes list/show command

**Files:**
- Create: `cmd/expense/taxes.go`

- [ ] **Step 1: Create `cmd/expense/taxes.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	taxesListPage int
	taxesListAll  bool
)

var taxesCmd = &cobra.Command{
	Use:   "taxes",
	Short: "Tax classification operations (API: excises)",
}

var taxesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tax classifications",
	RunE:  runTaxesList,
}

var taxesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show tax classification details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runTaxesShow,
}

func init() {
	taxesListCmd.Flags().IntVar(&taxesListPage, "page", 1, "page number")
	taxesListCmd.Flags().BoolVar(&taxesListAll, "all", false, "fetch all pages")

	taxesCmd.AddCommand(taxesListCmd)
	taxesCmd.AddCommand(taxesShowCmd)
}

func runTaxesList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if taxesListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseExcise, bool, error) {
			return svc.ListExcises(oid, page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"excises": all})
		}
		return f.Format(os.Stdout, all)
	}

	excises, _, err := svc.ListExcises(oid, taxesListPage)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"excises": excises})
	}
	return f.Format(os.Stdout, excises)
}

func runTaxesShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	excise, err := svc.GetExcise(oid, args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, excise)
}
```

---

### Task 12: positions list/show command

**Files:**
- Create: `cmd/expense/positions.go`

- [ ] **Step 1: Create `cmd/expense/positions.go`**

```go
package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	positionsListPage int
	positionsListAll  bool
)

var positionsCmd = &cobra.Command{
	Use:   "positions",
	Short: "Position operations",
}

var positionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List positions",
	RunE:  runPositionsList,
}

var positionsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show position details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPositionsShow,
}

func init() {
	positionsListCmd.Flags().IntVar(&positionsListPage, "page", 1, "page number")
	positionsListCmd.Flags().BoolVar(&positionsListAll, "all", false, "fetch all pages")

	positionsCmd.AddCommand(positionsListCmd)
	positionsCmd.AddCommand(positionsShowCmd)
}

func runPositionsList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if positionsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.Position, bool, error) {
			return svc.ListPositions(oid, page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"positions": all})
		}
		return f.Format(os.Stdout, all)
	}

	positions, _, err := svc.ListPositions(oid, positionsListPage)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"positions": positions})
	}
	return f.Format(os.Stdout, positions)
}

func runPositionsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	pos, err := svc.GetPosition(oid, args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, pos)
}
```

---

### Task 13: Build, test, and final commit

**Files:**
- All files from Tasks 6-12

- [ ] **Step 1: Register expense command in root.go**

Add to `cmd/root.go`:

```go
import (
	// existing imports...
	"github.com/planitaicojp/moneyforward-cli/cmd/expense"
)
```

In `init()`:
```go
	rootCmd.AddCommand(expense.ExpenseCmd)
```

- [ ] **Step 2: Build entire project**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 4: Verify CLI help output**

```bash
go run . expense --help
go run . expense offices --help
go run . expense departments --help
go run . expense projects --help
go run . expense categories --help
go run . expense taxes --help
go run . expense positions --help
```

Expected: each shows correct subcommands and flags.

- [ ] **Step 5: Commit all cmd/expense/ files and root.go change**

```bash
git add cmd/expense/ cmd/root.go
git commit -m "Phase 3a: Expense offices + master data commands (departments, projects, categories, taxes, positions)"
```

---

## Implementation Notes

### Expense vs Invoice API differences

| Aspect | Invoice API | Expense API |
|--------|------------|-------------|
| Pagination | `{"data": [...], "pagination": {total_count, total_pages}}` | `{"<resource>": [...], "next": "url", "prev": "url"}` |
| List key | Always `"data"` | Resource-specific (e.g., `"offices"`, `"depts"`) |
| Base URL | Single | v1 + v2 |
| office_id | Not in path | Required in all paths |
| fetchAll | `FetchAll` (page-count based) | `FetchAllExpense` (cursor/hasNext based) |

### DoRaw vs DoJSON

`DoJSON` decodes directly into a struct, but Expense API responses have variable top-level keys. `DoRaw` returns raw bytes so `decodeExpenseList` can extract the correct key dynamically.
