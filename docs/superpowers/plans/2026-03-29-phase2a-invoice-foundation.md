# Phase 2a: Invoice Foundation + Office + Partners — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the Invoice API client pattern, pagination helper, and data models, then implement office and partners commands as the first CRUD surface.

**Architecture:** `InvoiceService` wraps the existing Phase 1 `Client` with typed methods per resource. A shared `pagination` package provides page-based query helpers. Commands follow cobra subcommand nesting: `mf invoice partners list`.

**Tech Stack:** Go, cobra, httptest for API tests

**Design Spec:** `docs/superpowers/specs/2026-03-29-phase2-invoice-api-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/pagination/pagination.go` | Create | Pagination params + result structs, query string builder |
| `internal/pagination/pagination_test.go` | Create | Unit tests for pagination |
| `internal/model/invoice.go` | Create | Office, Partner, PartnerDepartment structs + CreatePartnerParams, UpdatePartnerParams |
| `internal/api/invoice.go` | Create | InvoiceService with office + partners methods |
| `internal/api/invoice_test.go` | Create | httptest-based tests for InvoiceService |
| `cmd/invoice/invoice.go` | Create | `mf invoice` root command, register subcommands |
| `cmd/invoice/office.go` | Create | `mf invoice office show` command |
| `cmd/invoice/partners.go` | Create | `mf invoice partners list|show|create|update|delete` commands |
| `cmd/invoice/partners_departments.go` | Create | `mf invoice partners departments list` command |
| `cmd/root.go` | Modify | Register `invoice` command |

---

### Task 1: Pagination Package

**Files:**
- Create: `internal/pagination/pagination.go`
- Create: `internal/pagination/pagination_test.go`

- [ ] **Step 1: Write failing tests for pagination**

Create `internal/pagination/pagination_test.go`:

```go
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
	// url.Values sorts keys alphabetically
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/pagination/ -v`
Expected: compilation error — package does not exist

- [ ] **Step 3: Write pagination implementation**

Create `internal/pagination/pagination.go`:

```go
package pagination

import (
	"fmt"
	"net/url"
)

// Params holds pagination query parameters.
type Params struct {
	Page    int
	PerPage int
	Query   string
}

// DefaultParams returns default pagination parameters.
func DefaultParams() Params {
	return Params{Page: 1, PerPage: 25}
}

// QueryString returns URL-encoded query parameters.
func (p Params) QueryString() string {
	v := url.Values{}
	v.Set("page", fmt.Sprintf("%d", p.Page))
	v.Set("per_page", fmt.Sprintf("%d", p.PerPage))
	if p.Query != "" {
		v.Set("q", p.Query)
	}
	return v.Encode()
}

// Result holds pagination metadata from the API response.
type Result struct {
	TotalCount  int `json:"total_count"`
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/pagination/ -v`
Expected: all 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pagination/
git commit -m "Add pagination package for page-based API queries"
```

---

### Task 2: Invoice Data Models

**Files:**
- Create: `internal/model/invoice.go`

- [ ] **Step 1: Create model file with all PR 2a structs**

Create `internal/model/invoice.go`:

```go
package model

// Office represents the organization info from GET /office.
type Office struct {
	Name       string `json:"name"`
	Zip        string `json:"zip"`
	Prefecture string `json:"prefecture"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Tel        string `json:"tel"`
	Fax        string `json:"fax"`
}

// Partner represents a business partner.
type Partner struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	NameKana    string              `json:"name_kana,omitempty"`
	NameSuffix  string              `json:"name_suffix,omitempty"`
	Code        string              `json:"code,omitempty"`
	Memo        string              `json:"memo,omitempty"`
	Departments []PartnerDepartment `json:"departments"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// PartnerDepartment represents a department within a partner.
type PartnerDepartment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Zip         string `json:"zip,omitempty"`
	Tel         string `json:"tel,omitempty"`
	Prefecture  string `json:"prefecture,omitempty"`
	Address1    string `json:"address1,omitempty"`
	Address2    string `json:"address2,omitempty"`
	PersonName  string `json:"person_name,omitempty"`
	PersonTitle string `json:"person_title,omitempty"`
	Email       string `json:"email,omitempty"`
	CCEmails    string `json:"cc_emails,omitempty"`
}

// CreatePartnerParams holds parameters for creating a partner.
type CreatePartnerParams struct {
	Name       string `json:"name"`
	NameKana   string `json:"name_kana,omitempty"`
	NameSuffix string `json:"name_suffix,omitempty"`
	Code       string `json:"code,omitempty"`
	Memo       string `json:"memo,omitempty"`
}

// UpdatePartnerParams holds parameters for updating a partner.
// Pointer fields allow distinguishing "not set" from "set to empty".
type UpdatePartnerParams struct {
	Name       *string `json:"name,omitempty"`
	NameKana   *string `json:"name_kana,omitempty"`
	NameSuffix *string `json:"name_suffix,omitempty"`
	Code       *string `json:"code,omitempty"`
	Memo       *string `json:"memo,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/model/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/model/invoice.go
git commit -m "Add Invoice data models: Office, Partner, PartnerDepartment"
```

---

### Task 3: InvoiceService — GetOffice

**Files:**
- Create: `internal/api/invoice.go`
- Create: `internal/api/invoice_test.go`

- [ ] **Step 1: Write failing test for GetOffice**

Create `internal/api/invoice_test.go`:

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

func TestInvoiceService_GetOffice(t *testing.T) {
	office := model.Office{
		Name:       "Test Corp",
		Zip:        "100-0001",
		Prefecture: "Tokyo",
		Address1:   "Chiyoda-ku",
		Address2:   "1-1-1",
		Tel:        "03-1234-5678",
		Fax:        "03-1234-5679",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/office" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(office)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	got, err := svc.GetOffice()
	if err != nil {
		t.Fatalf("GetOffice: %v", err)
	}
	if got.Name != "Test Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Corp")
	}
	if got.Zip != "100-0001" {
		t.Errorf("Zip = %q, want %q", got.Zip, "100-0001")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run TestInvoiceService_GetOffice -v`
Expected: compilation error — `NewInvoiceService` not defined

- [ ] **Step 3: Write InvoiceService with GetOffice**

Create `internal/api/invoice.go`:

```go
package api

import (
	"fmt"
	"net/http"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

const invoiceBaseURL = "https://invoice.moneyforward.com/api/v3"

// InvoiceService provides methods for the Invoice API.
type InvoiceService struct {
	client *Client
	base   string
}

// NewInvoiceService creates an InvoiceService.
// For production use, pass invoiceBaseURL as base.
// For testing, pass the httptest server URL.
func NewInvoiceService(client *Client, base string) *InvoiceService {
	return &InvoiceService{client: client, base: base}
}

// NewInvoiceServiceDefault creates an InvoiceService with the production base URL.
func NewInvoiceServiceDefault(client *Client) *InvoiceService {
	return NewInvoiceService(client, invoiceBaseURL)
}

// GetOffice returns the office (organization) information.
func (s *InvoiceService) GetOffice() (*model.Office, error) {
	var office model.Office
	err := s.client.DoJSON(http.MethodGet, s.base+"/office", nil, &office)
	if err != nil {
		return nil, fmt.Errorf("getting office: %w", err)
	}
	return &office, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestInvoiceService_GetOffice -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add InvoiceService with GetOffice method"
```

---

### Task 4: InvoiceService — ListPartners and GetPartner

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write failing tests for ListPartners and GetPartner**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_ListPartners(t *testing.T) {
	response := map[string]any{
		"data": []model.Partner{
			{ID: "p1", Name: "Partner A"},
			{ID: "p2", Name: "Partner B"},
		},
		"pagination": pagination.Result{
			TotalCount: 2, TotalPages: 1, CurrentPage: 1, PerPage: 25,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %q, want %q", r.URL.Query().Get("page"), "1")
		}
		if r.URL.Query().Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", r.URL.Query().Get("per_page"), "25")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	partners, pg, err := svc.ListPartners(pagination.DefaultParams())
	if err != nil {
		t.Fatalf("ListPartners: %v", err)
	}
	if len(partners) != 2 {
		t.Fatalf("len(partners) = %d, want 2", len(partners))
	}
	if partners[0].Name != "Partner A" {
		t.Errorf("partners[0].Name = %q, want %q", partners[0].Name, "Partner A")
	}
	if pg.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", pg.TotalCount)
	}
}

func TestInvoiceService_ListPartners_WithQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "abc" {
			t.Errorf("q = %q, want %q", r.URL.Query().Get("q"), "abc")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data":       []model.Partner{},
			"pagination": pagination.Result{TotalCount: 0, TotalPages: 0, CurrentPage: 1, PerPage: 25},
		})
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	params := pagination.Params{Page: 1, PerPage: 25, Query: "abc"}
	partners, _, err := svc.ListPartners(params)
	if err != nil {
		t.Fatalf("ListPartners: %v", err)
	}
	if len(partners) != 0 {
		t.Fatalf("len(partners) = %d, want 0", len(partners))
	}
}

func TestInvoiceService_GetPartner(t *testing.T) {
	partner := model.Partner{
		ID:   "p1",
		Name: "Partner A",
		Code: "PA001",
		Departments: []model.PartnerDepartment{
			{ID: "d1", Name: "Sales"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(partner)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	got, err := svc.GetPartner("p1")
	if err != nil {
		t.Fatalf("GetPartner: %v", err)
	}
	if got.Name != "Partner A" {
		t.Errorf("Name = %q, want %q", got.Name, "Partner A")
	}
	if len(got.Departments) != 1 {
		t.Fatalf("len(Departments) = %d, want 1", len(got.Departments))
	}
}
```

Add the import for `pagination` at the top of `invoice_test.go`:

```go
import (
	...
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/ -run TestInvoiceService_List -v`
Expected: compilation error — `ListPartners` not defined

- [ ] **Step 3: Add ListPartners and GetPartner to InvoiceService**

Append to `internal/api/invoice.go`:

```go
// listResponse is a generic wrapper for paginated API responses.
type listResponse[T any] struct {
	Data       []T               `json:"data"`
	Pagination pagination.Result `json:"pagination"`
}

// ListPartners returns a paginated list of partners.
func (s *InvoiceService) ListPartners(params pagination.Params) ([]model.Partner, *pagination.Result, error) {
	url := fmt.Sprintf("%s/partners?%s", s.base, params.QueryString())
	var resp listResponse[model.Partner]
	if err := s.client.DoJSON(http.MethodGet, url, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing partners: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

// GetPartner returns a partner by ID.
func (s *InvoiceService) GetPartner(id string) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/partners/%s", s.base, id), nil, &partner)
	if err != nil {
		return nil, fmt.Errorf("getting partner: %w", err)
	}
	return &partner, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/api/ -run TestInvoiceService -v`
Expected: all 4 tests PASS (GetOffice + ListPartners + ListPartners_WithQuery + GetPartner)

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add ListPartners and GetPartner to InvoiceService"
```

---

### Task 5: InvoiceService — CreatePartner, UpdatePartner, DeletePartner

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write failing tests for Create, Update, Delete**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_CreatePartner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/v3/partners" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var params model.CreatePartnerParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if params.Name != "New Corp" {
			t.Errorf("Name = %q, want %q", params.Name, "New Corp")
		}
		created := model.Partner{ID: "p-new", Name: params.Name, Code: params.Code}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	got, err := svc.CreatePartner(model.CreatePartnerParams{Name: "New Corp", Code: "NC001"})
	if err != nil {
		t.Fatalf("CreatePartner: %v", err)
	}
	if got.ID != "p-new" {
		t.Errorf("ID = %q, want %q", got.ID, "p-new")
	}
}

func TestInvoiceService_UpdatePartner(t *testing.T) {
	newName := "Updated Corp"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v3/partners/p1" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var params model.UpdatePartnerParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if params.Name == nil || *params.Name != "Updated Corp" {
			t.Errorf("Name = %v, want %q", params.Name, "Updated Corp")
		}
		updated := model.Partner{ID: "p1", Name: *params.Name}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	got, err := svc.UpdatePartner("p1", model.UpdatePartnerParams{Name: &newName})
	if err != nil {
		t.Fatalf("UpdatePartner: %v", err)
	}
	if got.Name != "Updated Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Corp")
	}
}

func TestInvoiceService_DeletePartner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v3/partners/p1" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	err := svc.DeletePartner("p1")
	if err != nil {
		t.Fatalf("DeletePartner: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/ -run TestInvoiceService_CreatePartner -v`
Expected: compilation error — `CreatePartner` not defined

- [ ] **Step 3: Add Create, Update, Delete methods**

Append to `internal/api/invoice.go`:

```go
// CreatePartner creates a new partner.
func (s *InvoiceService) CreatePartner(params model.CreatePartnerParams) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodPost, s.base+"/partners", params, &partner)
	if err != nil {
		return nil, fmt.Errorf("creating partner: %w", err)
	}
	return &partner, nil
}

// UpdatePartner updates an existing partner.
func (s *InvoiceService) UpdatePartner(id string, params model.UpdatePartnerParams) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/partners/%s", s.base, id), params, &partner)
	if err != nil {
		return nil, fmt.Errorf("updating partner: %w", err)
	}
	return &partner, nil
}

// DeletePartner deletes a partner. Returns nil on success (HTTP 204).
func (s *InvoiceService) DeletePartner(id string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/partners/%s", s.base, id), nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("deleting partner: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("deleting partner: HTTP %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/api/ -run TestInvoiceService -v`
Expected: all 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add CreatePartner, UpdatePartner, DeletePartner to InvoiceService"
```

---

### Task 6: InvoiceService — ListPartnerDepartments

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write failing test**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_ListPartnerDepartments(t *testing.T) {
	departments := []model.PartnerDepartment{
		{ID: "d1", Name: "Sales", PersonName: "Taro Yamada"},
		{ID: "d2", Name: "Engineering"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1/departments" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": departments})
	}))
	defer srv.Close()

	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")

	got, err := svc.ListPartnerDepartments("p1")
	if err != nil {
		t.Fatalf("ListPartnerDepartments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].PersonName != "Taro Yamada" {
		t.Errorf("PersonName = %q, want %q", got[0].PersonName, "Taro Yamada")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run TestInvoiceService_ListPartnerDepartments -v`
Expected: compilation error — `ListPartnerDepartments` not defined

- [ ] **Step 3: Add ListPartnerDepartments**

Append to `internal/api/invoice.go`:

```go
// departmentsResponse wraps the departments list endpoint response.
type departmentsResponse struct {
	Data []model.PartnerDepartment `json:"data"`
}

// ListPartnerDepartments returns departments for a partner.
func (s *InvoiceService) ListPartnerDepartments(partnerID string) ([]model.PartnerDepartment, error) {
	var resp departmentsResponse
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/partners/%s/departments", s.base, partnerID), nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("listing partner departments: %w", err)
	}
	return resp.Data, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestInvoiceService -v`
Expected: all 8 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add ListPartnerDepartments to InvoiceService"
```

---

### Task 7: `mf invoice` Root Command + `office show`

**Files:**
- Create: `cmd/invoice/invoice.go`
- Create: `cmd/invoice/office.go`
- Modify: `cmd/root.go` (add invoice import + AddCommand)

- [ ] **Step 1: Create invoice root command**

Create `cmd/invoice/invoice.go`:

```go
package invoice

import "github.com/spf13/cobra"

// InvoiceCmd is the root command for Invoice API operations.
var InvoiceCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Money Forward Cloud Invoice",
	Long:  "Commands for Money Forward Cloud Invoice API",
}

func init() {
	InvoiceCmd.AddCommand(officeCmd)
}
```

- [ ] **Step 2: Create office show command**

Create `cmd/invoice/office.go`:

```go
package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var officeCmd = &cobra.Command{
	Use:   "office",
	Short: "Office operations",
}

var officeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show office information",
	RunE:  runOfficeShow,
}

func init() {
	officeCmd.AddCommand(officeShowCmd)
}

func runOfficeShow(cmd *cobra.Command, args []string) error {
	profile := cmdutil.GetProfile(cmd)
	token, err := cmdutil.EnsureValidToken(profile, api.Services["invoice"])
	if err != nil {
		return err
	}

	client := api.NewWithToken(token, "dev", false)
	svc := api.NewInvoiceServiceDefault(client)

	office, err := svc.GetOffice()
	if err != nil {
		return err
	}

	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "" {
		format = "table"
	}
	f := output.New(format)
	return f.Format(os.Stdout, []any{office})
}
```

- [ ] **Step 3: Register invoice command in root.go**

Add import and `AddCommand` in `cmd/root.go`:

Add to imports:
```go
"github.com/planitaicojp/moneyforward-cli/cmd/invoice"
```

Add to `init()`:
```go
rootCmd.AddCommand(invoice.InvoiceCmd)
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add cmd/invoice/invoice.go cmd/invoice/office.go cmd/root.go
git commit -m "Add mf invoice command with office show subcommand"
```

---

### Task 8: `mf invoice partners list` and `show` Commands

**Files:**
- Create: `cmd/invoice/partners.go`
- Modify: `cmd/invoice/invoice.go` (register partners command)

- [ ] **Step 1: Create partners command file**

Create `cmd/invoice/partners.go`:

```go
package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

var partnersCmd = &cobra.Command{
	Use:   "partners",
	Short: "Partner operations",
}

// --- list ---

var (
	partnersListPage    int
	partnersListPerPage int
	partnersListQuery   string
	partnersListAll     bool
)

var partnersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List partners",
	RunE:  runPartnersList,
}

func init() {
	partnersListCmd.Flags().IntVar(&partnersListPage, "page", 1, "page number")
	partnersListCmd.Flags().IntVar(&partnersListPerPage, "per-page", 25, "items per page (max 100)")
	partnersListCmd.Flags().StringVar(&partnersListQuery, "query", "", "search query")
	partnersListCmd.Flags().BoolVar(&partnersListAll, "all", false, "fetch all pages")

	partnersCmd.AddCommand(partnersListCmd)
	partnersCmd.AddCommand(partnersShowCmd)
}

func newInvoiceService(cmd *cobra.Command) (*api.InvoiceService, error) {
	profile := cmdutil.GetProfile(cmd)
	token, err := cmdutil.EnsureValidToken(profile, api.Services["invoice"])
	if err != nil {
		return nil, err
	}
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	client := api.NewWithToken(token, "dev", verbose)
	return api.NewInvoiceServiceDefault(client), nil
}

func getFormat(cmd *cobra.Command) string {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "" {
		format = "table"
	}
	return format
}

func runPartnersList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := pagination.Params{
		Page:    partnersListPage,
		PerPage: partnersListPerPage,
		Query:   partnersListQuery,
	}

	partners, pg, err := svc.ListPartners(params)
	if err != nil {
		return err
	}

	format := getFormat(cmd)
	f := output.New(format)

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": partners, "pagination": pg})
	}
	return f.Format(os.Stdout, partners)
}

// --- show ---

var partnersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show partner details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersShow,
}

func runPartnersShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	partner, err := svc.GetPartner(args[0])
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}
```

- [ ] **Step 2: Register partners in invoice.go**

Add to `cmd/invoice/invoice.go` init():

```go
InvoiceCmd.AddCommand(partnersCmd)
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/partners.go cmd/invoice/invoice.go
git commit -m "Add mf invoice partners list and show commands"
```

---

### Task 9: `mf invoice partners create`, `update`, `delete` Commands

**Files:**
- Modify: `cmd/invoice/partners.go`

- [ ] **Step 1: Add create, update, delete commands**

Append to `cmd/invoice/partners.go`:

```go
// --- create ---

var (
	partnersCreateName       string
	partnersCreateNameKana   string
	partnersCreateNameSuffix string
	partnersCreateCode       string
	partnersCreateMemo       string
)

var partnersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a partner",
	RunE:  runPartnersCreate,
}

// --- update ---

var (
	partnersUpdateName       string
	partnersUpdateNameKana   string
	partnersUpdateNameSuffix string
	partnersUpdateCode       string
	partnersUpdateMemo       string
)

var partnersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersUpdate,
}

// --- delete ---

var partnersDeleteYes bool

var partnersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersDelete,
}
```

Add to the existing `init()` in partners.go (where `partnersListCmd` flags are registered):

```go
	partnersCreateCmd.Flags().StringVar(&partnersCreateName, "name", "", "partner name (required)")
	partnersCreateCmd.Flags().StringVar(&partnersCreateNameKana, "name-kana", "", "partner name in kana")
	partnersCreateCmd.Flags().StringVar(&partnersCreateNameSuffix, "name-suffix", "", "name suffix (e.g. 様, 御中)")
	partnersCreateCmd.Flags().StringVar(&partnersCreateCode, "code", "", "partner code")
	partnersCreateCmd.Flags().StringVar(&partnersCreateMemo, "memo", "", "memo")
	_ = partnersCreateCmd.MarkFlagRequired("name")

	partnersUpdateCmd.Flags().StringVar(&partnersUpdateName, "name", "", "partner name")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateNameKana, "name-kana", "", "partner name in kana")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateNameSuffix, "name-suffix", "", "name suffix")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateCode, "code", "", "partner code")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateMemo, "memo", "", "memo")

	partnersDeleteCmd.Flags().BoolVar(&partnersDeleteYes, "yes", false, "skip confirmation prompt")

	partnersCmd.AddCommand(partnersCreateCmd)
	partnersCmd.AddCommand(partnersUpdateCmd)
	partnersCmd.AddCommand(partnersDeleteCmd)
```

Add the RunE implementations:

```go
func runPartnersCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := model.CreatePartnerParams{
		Name:       partnersCreateName,
		NameKana:   partnersCreateNameKana,
		NameSuffix: partnersCreateNameSuffix,
		Code:       partnersCreateCode,
		Memo:       partnersCreateMemo,
	}

	partner, err := svc.CreatePartner(params)
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}

func runPartnersUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdatePartnerParams
	if cmd.Flags().Changed("name") {
		params.Name = &partnersUpdateName
	}
	if cmd.Flags().Changed("name-kana") {
		params.NameKana = &partnersUpdateNameKana
	}
	if cmd.Flags().Changed("name-suffix") {
		params.NameSuffix = &partnersUpdateNameSuffix
	}
	if cmd.Flags().Changed("code") {
		params.Code = &partnersUpdateCode
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &partnersUpdateMemo
	}

	partner, err := svc.UpdatePartner(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}

func runPartnersDelete(cmd *cobra.Command, args []string) error {
	if !partnersDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete partner %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	return svc.DeletePartner(args[0])
}
```

Add the missing imports at the top of `partners.go`:

```go
import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/invoice/partners.go
git commit -m "Add mf invoice partners create, update, delete commands"
```

---

### Task 10: `mf invoice partners departments list` Command

**Files:**
- Create: `cmd/invoice/partners_departments.go`
- Modify: `cmd/invoice/invoice.go` (register in partners)

- [ ] **Step 1: Create departments command**

Create `cmd/invoice/partners_departments.go`:

```go
package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var partnersDepartmentsCmd = &cobra.Command{
	Use:   "departments",
	Short: "Partner department operations",
}

var partnersDepartmentsListCmd = &cobra.Command{
	Use:   "list <partner-id>",
	Short: "List departments for a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersDepartmentsList,
}

func init() {
	partnersDepartmentsCmd.AddCommand(partnersDepartmentsListCmd)
}

func runPartnersDepartmentsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	departments, err := svc.ListPartnerDepartments(args[0])
	if err != nil {
		return err
	}

	format := getFormat(cmd)
	f := output.New(format)

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": departments})
	}
	return f.Format(os.Stdout, departments)
}
```

- [ ] **Step 2: Register departments under partners**

Add to `cmd/invoice/invoice.go` init() (after `InvoiceCmd.AddCommand(partnersCmd)`):

```go
partnersCmd.AddCommand(partnersDepartmentsCmd)
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/partners_departments.go cmd/invoice/invoice.go
git commit -m "Add mf invoice partners departments list command"
```

---

### Task 11: Refactor office.go to use shared helpers + final wiring

**Files:**
- Modify: `cmd/invoice/office.go` (use `newInvoiceService` and `getFormat` from partners.go)

- [ ] **Step 1: Refactor office.go to use shared helpers**

Update `cmd/invoice/office.go` to use the `newInvoiceService` and `getFormat` helpers defined in partners.go:

```go
package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var officeCmd = &cobra.Command{
	Use:   "office",
	Short: "Office operations",
}

var officeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show office information",
	RunE:  runOfficeShow,
}

func init() {
	officeCmd.AddCommand(officeShowCmd)
}

func runOfficeShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	office, err := svc.GetOffice()
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, office)
}
```

- [ ] **Step 2: Run full build and all tests**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: all pass, no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/invoice/office.go
git commit -m "Refactor office.go to use shared invoice helpers"
```

---

### Task 12: Final Verification and PR Branch

- [ ] **Step 1: Run full test suite**

Run: `go build ./... && go test ./... -v && go vet ./...`
Expected: all existing Phase 1 tests pass, all new pagination + invoice tests pass

- [ ] **Step 2: Verify command tree**

Run: `go run . invoice --help`
Expected output should show: `office`, `partners` subcommands

Run: `go run . invoice partners --help`
Expected output should show: `list`, `show`, `create`, `update`, `delete`, `departments` subcommands

- [ ] **Step 3: Create PR branch and push**

```bash
git checkout -b phase2a-invoice-foundation
git push -u origin phase2a-invoice-foundation
```

- [ ] **Step 4: Create PR**

Title: `Phase 2a: Invoice foundation + office + partners`

Body should include:
- Summary of new packages (pagination, model, api/invoice)
- Command list (7 commands)
- Test count
- Link to design spec
