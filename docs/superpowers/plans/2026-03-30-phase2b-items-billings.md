# Phase 2b: Items + Billings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Item CRUD and Billing CRUD commands with line item input (`--item`, `--items-file`, `--items-stdin`), excise aliases, department_id auto-resolution, `--dry-run`, `--all` auto-pagination, `--partner` name resolution, and PDF download.

**Architecture:** Extends the existing `InvoiceService` with Item and Billing methods. A new `cmd/invoice/itemparse` helper package handles line-item parsing and excise alias resolution. The `--all` auto-pagination loop is implemented in commands that need it (items list, billings list), with `time.Sleep(400ms)` between pages for rate-limit compliance.

**Tech Stack:** Go, cobra, httptest for API tests

**Design Spec:** `docs/superpowers/specs/2026-03-29-phase2-invoice-api-design.md` (PR 2b section)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/model/invoice.go` | Modify | Add Item, Billing, BillingItem structs + enums + Create/Update params |
| `internal/api/invoice.go` | Modify | Add Item and Billing API methods to InvoiceService |
| `internal/api/invoice_test.go` | Modify | Add tests for new API methods |
| `cmd/invoice/excise.go` | Create | Excise alias resolution (`"10"` → `"ten_percent"`) |
| `cmd/invoice/excise_test.go` | Create | Tests for excise alias resolution |
| `cmd/invoice/itemparse.go` | Create | Parse `--item` flag, `--items-file`, `--items-stdin` into line items |
| `cmd/invoice/itemparse_test.go` | Create | Tests for item parsing |
| `cmd/invoice/items.go` | Create | `mf invoice items list\|show\|create\|update\|delete` commands |
| `cmd/invoice/billings.go` | Create | `mf invoice billings list\|show\|create\|update\|delete\|set-payment-status\|pdf` commands |
| `cmd/invoice/invoice.go` | Modify | Register items and billings subcommands |
| `cmd/invoice/partners.go` | Modify | Add `--all` auto-pagination to partners list |

---

### Task 1: Data Models — Item, Billing, Enums

**Files:**
- Modify: `internal/model/invoice.go`

- [ ] **Step 1: Add Item, Billing, BillingItem, enums, and request param structs**

Append to `internal/model/invoice.go`:

```go
// --- Enums ---

type PaymentStatus string

const (
	PaymentStatusUnsettled PaymentStatus = "unsettled"
	PaymentStatusSettled   PaymentStatus = "settled"
)

type ExciseType string

const (
	ExciseTenPercent                 ExciseType = "ten_percent"
	ExciseEightPercent               ExciseType = "eight_percent"
	ExciseEightPercentReducedTaxRate ExciseType = "eight_percent_as_reduced_tax_rate"
	ExciseFivePercent                ExciseType = "five_percent"
	ExciseUntaxable                  ExciseType = "untaxable"
	ExciseTaxExemption               ExciseType = "tax_exemption"
	ExciseNonTaxable                 ExciseType = "non_taxable"
)

// --- Item ---

type Item struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  *int   `json:"price,omitempty"`
	Quantity               *int   `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

type CreateItemParams struct {
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  *int   `json:"price,omitempty"`
	Quantity               *int   `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
}

type UpdateItemParams struct {
	Name                   *string `json:"name,omitempty"`
	Code                   *string `json:"code,omitempty"`
	Detail                 *string `json:"detail,omitempty"`
	Unit                   *string `json:"unit,omitempty"`
	Price                  *int    `json:"price,omitempty"`
	Quantity               *int    `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool   `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 *string `json:"excise,omitempty"`
}

// --- Billing ---

type Billing struct {
	ID               string        `json:"id"`
	PDFURL           string        `json:"pdf_url,omitempty"`
	OperatorID       string        `json:"operator_id,omitempty"`
	DepartmentID     string        `json:"department_id,omitempty"`
	PartnerID        string        `json:"partner_id,omitempty"`
	PartnerName      string        `json:"partner_name,omitempty"`
	PartnerDetail    string        `json:"partner_detail,omitempty"`
	MemberID         string        `json:"member_id,omitempty"`
	MemberName       string        `json:"member_name,omitempty"`
	Title            string        `json:"title,omitempty"`
	Memo             string        `json:"memo,omitempty"`
	PaymentCondition string        `json:"payment_condition,omitempty"`
	BillingNumber    string        `json:"billing_number,omitempty"`
	BillingDate      string        `json:"billing_date,omitempty"`
	DueDate          string        `json:"due_date,omitempty"`
	SalesDate        string        `json:"sales_date,omitempty"`
	PaymentStatus    PaymentStatus `json:"payment_status"`
	Subtotal         *int          `json:"subtotal,omitempty"`
	TotalPrice       *int          `json:"total_price,omitempty"`
	Tax              *int          `json:"tax,omitempty"`
	Items            []BillingItem `json:"items"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
}

type BillingItem struct {
	ID                     string `json:"id,omitempty"`
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  int    `json:"price"`
	Quantity               int    `json:"quantity"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
}

// InvoiceTemplateLine is a line item for billing/quote creation (Invoice Act compliant).
type InvoiceTemplateLine struct {
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  int    `json:"price"`
	Quantity               int    `json:"quantity"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise"`
}

// CreateBillingParams is sent to POST /invoice_template_billings (Invoice Act).
type CreateBillingParams struct {
	DepartmentID     string                `json:"department_id"`
	BillingDate      string                `json:"billing_date"`
	Title            string                `json:"title,omitempty"`
	Memo             string                `json:"memo,omitempty"`
	PaymentCondition string                `json:"payment_condition,omitempty"`
	DueDate          string                `json:"due_date,omitempty"`
	SalesDate        string                `json:"sales_date,omitempty"`
	BillingNumber    string                `json:"billing_number,omitempty"`
	Items            []InvoiceTemplateLine `json:"items,omitempty"`
}

// UpdateBillingParams is wrapped as {"billing": {...}} for PATCH /billings/{id}.
type UpdateBillingParams struct {
	Title            *string `json:"title,omitempty"`
	Memo             *string `json:"memo,omitempty"`
	PaymentCondition *string `json:"payment_condition,omitempty"`
	BillingDate      *string `json:"billing_date,omitempty"`
	DueDate          *string `json:"due_date,omitempty"`
	SalesDate        *string `json:"sales_date,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/model/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/model/invoice.go
git commit -m "Add Item, Billing, BillingItem models with enums and request params"
```

---

### Task 2: InvoiceService — Item API Methods

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Add Item methods to InvoiceService**

Append to `internal/api/invoice.go`:

```go
// --- Items ---

func (s *InvoiceService) ListItems(params pagination.Params, query string) ([]model.Item, *pagination.Result, error) {
	u := fmt.Sprintf("%s/items?%s", s.base, params.QueryString())
	if query != "" {
		u += "&q=" + query
	}
	var resp listResponse[model.Item]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing items: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetItem(id string) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/items/%s", s.base, id), nil, &item)
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) CreateItem(params model.CreateItemParams) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodPost, s.base+"/items", params, &item)
	if err != nil {
		return nil, fmt.Errorf("creating item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) UpdateItem(id string, params model.UpdateItemParams) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/items/%s", s.base, id), params, &item)
	if err != nil {
		return nil, fmt.Errorf("updating item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) DeleteItem(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/items/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting item: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Write tests for Item methods**

Add to `internal/api/invoice_test.go` tests: `TestInvoiceService_ListItems`, `TestInvoiceService_GetItem`, `TestInvoiceService_CreateItem`, `TestInvoiceService_UpdateItem`, `TestInvoiceService_DeleteItem`. Follow the same httptest pattern as existing partner tests — verify HTTP method, path, request body, and response parsing.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/api/ -run TestInvoiceService -v`
Expected: all tests pass (8 existing + 5 new = 13)

- [ ] **Step 4: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add Item CRUD methods to InvoiceService"
```

---

### Task 3: InvoiceService — Billing API Methods

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Add Billing methods to InvoiceService**

Append to `internal/api/invoice.go`. Note the critical wrapping rules:

```go
// --- Billings ---

// BillingListOptions holds filter parameters for listing billings.
type BillingListOptions struct {
	pagination.Params
	PartnerID     string
	PaymentStatus string
	From          string
	To            string
	Query         string
}

func (o BillingListOptions) queryString() string {
	qs := o.Params.QueryString()
	if o.PartnerID != "" {
		qs += "&partner_id=" + o.PartnerID
	}
	if o.PaymentStatus != "" {
		qs += "&payment_status=" + o.PaymentStatus
	}
	if o.From != "" {
		qs += "&from=" + o.From
	}
	if o.To != "" {
		qs += "&to=" + o.To
	}
	if o.Query != "" {
		qs += "&q=" + o.Query
	}
	return qs
}

func (s *InvoiceService) ListBillings(opts BillingListOptions) ([]model.Billing, *pagination.Result, error) {
	u := fmt.Sprintf("%s/billings?%s", s.base, opts.queryString())
	var resp listResponse[model.Billing]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing billings: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetBilling(id string) (*model.Billing, error) {
	var billing model.Billing
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/billings/%s", s.base, id), nil, &billing)
	if err != nil {
		return nil, fmt.Errorf("getting billing: %w", err)
	}
	return &billing, nil
}

// CreateBilling uses POST /invoice_template_billings (Invoice Act compliant).
// Body is sent directly (no wrapping).
func (s *InvoiceService) CreateBilling(params model.CreateBillingParams) (*model.Billing, error) {
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPost, s.base+"/invoice_template_billings", params, &billing)
	if err != nil {
		return nil, fmt.Errorf("creating billing: %w", err)
	}
	return &billing, nil
}

// UpdateBilling uses PATCH /billings/{id} with body wrapped as {"billing": {...}}.
func (s *InvoiceService) UpdateBilling(id string, params model.UpdateBillingParams) (*model.Billing, error) {
	wrapped := map[string]any{"billing": params}
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/billings/%s", s.base, id), wrapped, &billing)
	if err != nil {
		return nil, fmt.Errorf("updating billing: %w", err)
	}
	return &billing, nil
}

func (s *InvoiceService) DeleteBilling(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/billings/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting billing: %w", err)
	}
	return nil
}

// SetPaymentStatus uses PATCH /billings/{id} with {"billing": {"payment_status": status}}.
func (s *InvoiceService) SetPaymentStatus(id string, status model.PaymentStatus) (*model.Billing, error) {
	wrapped := map[string]any{"billing": map[string]any{"payment_status": status}}
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/billings/%s", s.base, id), wrapped, &billing)
	if err != nil {
		return nil, fmt.Errorf("setting payment status: %w", err)
	}
	return &billing, nil
}

// GetBillingPDF returns the PDF URL for a billing.
func (s *InvoiceService) GetBillingPDF(id string) (string, error) {
	billing, err := s.GetBilling(id)
	if err != nil {
		return "", err
	}
	return billing.PDFURL, nil
}
```

- [ ] **Step 2: Write tests for Billing methods**

Add to `internal/api/invoice_test.go` tests: `TestInvoiceService_ListBillings`, `TestInvoiceService_GetBilling`, `TestInvoiceService_CreateBilling` (verify path is `/invoice_template_billings` and body is direct), `TestInvoiceService_UpdateBilling` (verify wrapping `{"billing": {...}}`), `TestInvoiceService_DeleteBilling`, `TestInvoiceService_SetPaymentStatus` (verify wrapping).

Critical test for CreateBilling: verify the request goes to `/api/v3/invoice_template_billings` (NOT `/billings`).
Critical test for UpdateBilling: verify the request body is `{"billing": {"title": "..."}}` (wrapped).

- [ ] **Step 3: Run tests**

Run: `go test ./internal/api/ -run TestInvoiceService -v`
Expected: all tests pass (13 existing + 6 new = 19)

- [ ] **Step 4: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add Billing API methods with Invoice Act endpoint and wrapping rules"
```

---

### Task 4: Excise Alias Resolution

**Files:**
- Create: `cmd/invoice/excise.go`
- Create: `cmd/invoice/excise_test.go`

- [ ] **Step 1: Write failing tests for excise alias resolution**

Create `cmd/invoice/excise_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/invoice/ -run TestResolveExcise -v`
Expected: compilation error — `resolveExcise` not defined

- [ ] **Step 3: Implement excise alias resolution**

Create `cmd/invoice/excise.go`:

```go
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/invoice/ -run TestResolveExcise -v`
Expected: all 10 subtests pass

- [ ] **Step 5: Commit**

```bash
git add cmd/invoice/excise.go cmd/invoice/excise_test.go
git commit -m "Add excise alias resolution (10, 8, 8r, 5, 0, exempt, non)"
```

---

### Task 5: Line Item Parser (`--item`, `--items-file`, `--items-stdin`)

**Files:**
- Create: `cmd/invoice/itemparse.go`
- Create: `cmd/invoice/itemparse_test.go`

- [ ] **Step 1: Write failing tests for item parsing**

Create `cmd/invoice/itemparse_test.go`:

```go
package invoice

import (
	"strings"
	"testing"
)

func TestParseItemFlag(t *testing.T) {
	input := "name=Consulting,price=100000,quantity=1,excise=10"
	item, err := parseItemFlag(input)
	if err != nil {
		t.Fatalf("parseItemFlag: %v", err)
	}
	if item.Name != "Consulting" {
		t.Errorf("Name = %q, want %q", item.Name, "Consulting")
	}
	if item.Price != 100000 {
		t.Errorf("Price = %d, want 100000", item.Price)
	}
	if item.Quantity != 1 {
		t.Errorf("Quantity = %d, want 1", item.Quantity)
	}
	if item.Excise != "ten_percent" {
		t.Errorf("Excise = %q, want %q", item.Excise, "ten_percent")
	}
}

func TestParseItemFlag_AllFields(t *testing.T) {
	input := "name=Test,code=T001,detail=Desc,unit=hours,price=5000,quantity=3,excise=8r"
	item, err := parseItemFlag(input)
	if err != nil {
		t.Fatalf("parseItemFlag: %v", err)
	}
	if item.Code != "T001" {
		t.Errorf("Code = %q, want %q", item.Code, "T001")
	}
	if item.Detail != "Desc" {
		t.Errorf("Detail = %q, want %q", item.Detail, "Desc")
	}
	if item.Unit != "hours" {
		t.Errorf("Unit = %q, want %q", item.Unit, "hours")
	}
	if item.Excise != "eight_percent_as_reduced_tax_rate" {
		t.Errorf("Excise = %q, want %q", item.Excise, "eight_percent_as_reduced_tax_rate")
	}
}

func TestParseItemFlag_MissingName(t *testing.T) {
	_, err := parseItemFlag("price=100,quantity=1,excise=10")
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseItemsFromJSON(t *testing.T) {
	jsonStr := `[{"name":"A","price":100,"quantity":1,"excise":"ten_percent"},{"name":"B","price":200,"quantity":2,"excise":"untaxable"}]`
	items, err := parseItemsFromReader(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("parseItemsFromReader: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].Name != "A" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "A")
	}
	if items[1].Price != 200 {
		t.Errorf("items[1].Price = %d, want 200", items[1].Price)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/invoice/ -run TestParseItem -v`
Expected: compilation error

- [ ] **Step 3: Implement item parser**

Create `cmd/invoice/itemparse.go`:

```go
package invoice

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
)

// parseItemFlag parses a single --item flag value like "name=Consulting,price=100000,quantity=1,excise=10".
func parseItemFlag(s string) (model.InvoiceTemplateLine, error) {
	var item model.InvoiceTemplateLine
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return item, fmt.Errorf("invalid key=value pair: %q", pair)
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "name":
			item.Name = val
		case "code":
			item.Code = val
		case "detail":
			item.Detail = val
		case "unit":
			item.Unit = val
		case "price":
			n, err := strconv.Atoi(val)
			if err != nil {
				return item, fmt.Errorf("invalid price %q: %w", val, err)
			}
			item.Price = n
		case "quantity":
			n, err := strconv.Atoi(val)
			if err != nil {
				return item, fmt.Errorf("invalid quantity %q: %w", val, err)
			}
			item.Quantity = n
		case "excise":
			item.Excise = resolveExcise(val)
		default:
			return item, fmt.Errorf("unknown item field: %q", key)
		}
	}
	if item.Name == "" {
		return item, fmt.Errorf("item name is required")
	}
	return item, nil
}

// parseItemsFromReader reads JSON array of InvoiceTemplateLine from a reader.
func parseItemsFromReader(r io.Reader) ([]model.InvoiceTemplateLine, error) {
	var items []model.InvoiceTemplateLine
	if err := json.NewDecoder(r).Decode(&items); err != nil {
		return nil, fmt.Errorf("parsing items JSON: %w", err)
	}
	return items, nil
}

// resolveLineItems resolves line items from --items-stdin, --items-file, or --item flags.
// Priority: stdin > file > flags.
func resolveLineItems(itemFlags []string, itemsFile string, itemsStdin bool) ([]model.InvoiceTemplateLine, error) {
	if itemsStdin {
		return parseItemsFromReader(os.Stdin)
	}
	if itemsFile != "" {
		f, err := os.Open(itemsFile)
		if err != nil {
			return nil, fmt.Errorf("opening items file: %w", err)
		}
		defer f.Close()
		return parseItemsFromReader(f)
	}
	if len(itemFlags) > 0 {
		items := make([]model.InvoiceTemplateLine, 0, len(itemFlags))
		for _, flag := range itemFlags {
			item, err := parseItemFlag(flag)
			if err != nil {
				return nil, fmt.Errorf("parsing --item %q: %w", flag, err)
			}
			items = append(items, item)
		}
		return items, nil
	}
	return nil, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/invoice/ -run TestParseItem -v`
Expected: all 4 tests pass

- [ ] **Step 5: Commit**

```bash
git add cmd/invoice/itemparse.go cmd/invoice/itemparse_test.go
git commit -m "Add line item parser for --item, --items-file, --items-stdin"
```

---

### Task 6: Items Commands

**Files:**
- Create: `cmd/invoice/items.go`
- Modify: `cmd/invoice/invoice.go`

- [ ] **Step 1: Create items command file**

Create `cmd/invoice/items.go` with list, show, create, update, delete commands. Include `--all` auto-pagination in list:

```go
package invoice

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var itemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Item operations",
}

// --- list ---

var (
	itemsListPage    int
	itemsListPerPage int
	itemsListQuery   string
	itemsListAll     bool
)

var itemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items",
	RunE:  runItemsList,
}

// --- show ---

var itemsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show item details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsShow,
}

// --- create ---

var (
	itemsCreateName     string
	itemsCreateCode     string
	itemsCreateDetail   string
	itemsCreateUnit     string
	itemsCreatePrice    int
	itemsCreateQuantity int
	itemsCreateExcise   string
)

var itemsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an item",
	RunE:  runItemsCreate,
}

// --- update ---

var (
	itemsUpdateName     string
	itemsUpdateCode     string
	itemsUpdateDetail   string
	itemsUpdateUnit     string
	itemsUpdatePrice    int
	itemsUpdateQuantity int
	itemsUpdateExcise   string
)

var itemsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an item",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsUpdate,
}

// --- delete ---

var itemsDeleteYes bool

var itemsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an item",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsDelete,
}

func init() {
	itemsListCmd.Flags().IntVar(&itemsListPage, "page", 1, "page number")
	itemsListCmd.Flags().IntVar(&itemsListPerPage, "per-page", 25, "items per page (max 100)")
	itemsListCmd.Flags().StringVar(&itemsListQuery, "query", "", "search query")
	itemsListCmd.Flags().BoolVar(&itemsListAll, "all", false, "fetch all pages")

	itemsCreateCmd.Flags().StringVar(&itemsCreateName, "name", "", "item name (required)")
	itemsCreateCmd.Flags().StringVar(&itemsCreateCode, "code", "", "item code")
	itemsCreateCmd.Flags().StringVar(&itemsCreateDetail, "detail", "", "item detail")
	itemsCreateCmd.Flags().StringVar(&itemsCreateUnit, "unit", "", "unit (e.g. hours, pcs)")
	itemsCreateCmd.Flags().IntVar(&itemsCreatePrice, "price", 0, "unit price")
	itemsCreateCmd.Flags().IntVar(&itemsCreateQuantity, "quantity", 0, "quantity")
	itemsCreateCmd.Flags().StringVar(&itemsCreateExcise, "excise", "", "excise type (10, 8, 8r, 5, 0, exempt, non)")
	_ = itemsCreateCmd.MarkFlagRequired("name")

	itemsUpdateCmd.Flags().StringVar(&itemsUpdateName, "name", "", "item name")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateCode, "code", "", "item code")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateDetail, "detail", "", "item detail")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateUnit, "unit", "", "unit")
	itemsUpdateCmd.Flags().IntVar(&itemsUpdatePrice, "price", 0, "unit price")
	itemsUpdateCmd.Flags().IntVar(&itemsUpdateQuantity, "quantity", 0, "quantity")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateExcise, "excise", "", "excise type")

	itemsDeleteCmd.Flags().BoolVar(&itemsDeleteYes, "yes", false, "skip confirmation prompt")

	itemsCmd.AddCommand(itemsListCmd)
	itemsCmd.AddCommand(itemsShowCmd)
	itemsCmd.AddCommand(itemsCreateCmd)
	itemsCmd.AddCommand(itemsUpdateCmd)
	itemsCmd.AddCommand(itemsDeleteCmd)
}

func runItemsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if itemsListAll {
		var allItems []model.Item
		page := 1
		for {
			params := pagination.Params{Page: page, PerPage: 100}
			items, pg, err := svc.ListItems(params, itemsListQuery)
			if err != nil {
				return err
			}
			allItems = append(allItems, items...)
			if page >= pg.TotalPages {
				break
			}
			page++
			time.Sleep(400 * time.Millisecond)
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allItems})
		}
		return f.Format(os.Stdout, allItems)
	}

	params := pagination.Params{Page: itemsListPage, PerPage: itemsListPerPage}
	items, pg, err := svc.ListItems(params, itemsListQuery)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": items, "pagination": pg})
	}
	return f.Format(os.Stdout, items)
}

func runItemsShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	item, err := svc.GetItem(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := model.CreateItemParams{
		Name:   itemsCreateName,
		Code:   itemsCreateCode,
		Detail: itemsCreateDetail,
		Unit:   itemsCreateUnit,
	}
	if cmd.Flags().Changed("price") {
		params.Price = &itemsCreatePrice
	}
	if cmd.Flags().Changed("quantity") {
		params.Quantity = &itemsCreateQuantity
	}
	if itemsCreateExcise != "" {
		excise := resolveExcise(itemsCreateExcise)
		params.Excise = excise
	}

	item, err := svc.CreateItem(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateItemParams
	if cmd.Flags().Changed("name") {
		params.Name = &itemsUpdateName
	}
	if cmd.Flags().Changed("code") {
		params.Code = &itemsUpdateCode
	}
	if cmd.Flags().Changed("detail") {
		params.Detail = &itemsUpdateDetail
	}
	if cmd.Flags().Changed("unit") {
		params.Unit = &itemsUpdateUnit
	}
	if cmd.Flags().Changed("price") {
		params.Price = &itemsUpdatePrice
	}
	if cmd.Flags().Changed("quantity") {
		params.Quantity = &itemsUpdateQuantity
	}
	if cmd.Flags().Changed("excise") {
		excise := resolveExcise(itemsUpdateExcise)
		params.Excise = &excise
	}

	item, err := svc.UpdateItem(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsDelete(cmd *cobra.Command, args []string) error {
	if !itemsDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete item %s?", args[0]))
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

	return svc.DeleteItem(args[0])
}
```

- [ ] **Step 2: Register items in invoice.go**

Add to `cmd/invoice/invoice.go` init():

```go
InvoiceCmd.AddCommand(itemsCmd)
```

- [ ] **Step 3: Verify it compiles and test**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: all pass

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/items.go cmd/invoice/invoice.go
git commit -m "Add mf invoice items list|show|create|update|delete with --all pagination"
```

---

### Task 7: Billings Commands

**Files:**
- Create: `cmd/invoice/billings.go`
- Modify: `cmd/invoice/invoice.go`

- [ ] **Step 1: Create billings command file**

Create `cmd/invoice/billings.go` with all 7 subcommands: list (with `--all`, `--partner`), show, create (with `--item`, `--items-file`, `--items-stdin`, `--dry-run`, department_id auto-resolution), update (with `--dry-run`), delete, set-payment-status, pdf (with `--download`, `--output`).

This is a large file. Key implementation details:

- `billings list` uses `BillingListOptions` with all filter flags + `--all` auto-pagination + `--partner <name>` resolution (search partners, error on 0 or >1 match)
- `billings create` resolves line items via `resolveLineItems()`, resolves department_id if not provided via `svc.ListPartnerDepartments()`, outputs request body on `--dry-run`
- `billings update` sends `UpdateBillingParams` (wrapping handled by service layer)
- `billings set-payment-status` calls `svc.SetPaymentStatus()`
- `billings pdf` defaults to printing URL, `--download` fetches to file

```go
package invoice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var billingsCmd = &cobra.Command{
	Use:   "billings",
	Short: "Billing operations",
}

// --- list ---

var (
	billingsListPage          int
	billingsListPerPage       int
	billingsListPartnerID     string
	billingsListPartner       string
	billingsListPaymentStatus string
	billingsListFrom          string
	billingsListTo            string
	billingsListQuery         string
	billingsListAll           bool
)

var billingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List billings",
	RunE:  runBillingsList,
}

// --- show ---

var billingsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show billing details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsShow,
}

// --- create ---

var (
	billingsCreatePartnerID   string
	billingsCreateBillingDate string
	billingsCreateDepartment  string
	billingsCreateTitle       string
	billingsCreateMemo        string
	billingsCreatePaymentCond string
	billingsCreateDueDate     string
	billingsCreateSalesDate   string
	billingsCreateItemFlags   []string
	billingsCreateItemsFile   string
	billingsCreateItemsStdin  bool
	billingsCreateDryRun      bool
)

var billingsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a billing",
	RunE:  runBillingsCreate,
}

// --- update ---

var (
	billingsUpdateTitle       string
	billingsUpdateMemo        string
	billingsUpdatePaymentCond string
	billingsUpdateBillingDate string
	billingsUpdateDueDate     string
	billingsUpdateSalesDate   string
	billingsUpdateDryRun      bool
)

var billingsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsUpdate,
}

// --- delete ---

var billingsDeleteYes bool

var billingsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsDelete,
}

// --- set-payment-status ---

var billingsSetStatusValue string

var billingsSetPaymentStatusCmd = &cobra.Command{
	Use:   "set-payment-status <id>",
	Short: "Set billing payment status",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsSetPaymentStatus,
}

// --- pdf ---

var (
	billingsPDFDownload bool
	billingsPDFOutput   string
)

var billingsPDFCmd = &cobra.Command{
	Use:   "pdf <id>",
	Short: "Get billing PDF",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsPDF,
}

func init() {
	billingsListCmd.Flags().IntVar(&billingsListPage, "page", 1, "page number")
	billingsListCmd.Flags().IntVar(&billingsListPerPage, "per-page", 25, "items per page (max 100)")
	billingsListCmd.Flags().StringVar(&billingsListPartnerID, "partner-id", "", "filter by partner ID")
	billingsListCmd.Flags().StringVar(&billingsListPartner, "partner", "", "filter by partner name (resolved to ID)")
	billingsListCmd.Flags().StringVar(&billingsListPaymentStatus, "payment-status", "", "filter by payment status (unsettled|settled)")
	billingsListCmd.Flags().StringVar(&billingsListFrom, "from", "", "from date (YYYY-MM-DD)")
	billingsListCmd.Flags().StringVar(&billingsListTo, "to", "", "to date (YYYY-MM-DD)")
	billingsListCmd.Flags().StringVar(&billingsListQuery, "query", "", "search query")
	billingsListCmd.Flags().BoolVar(&billingsListAll, "all", false, "fetch all pages")

	billingsCreateCmd.Flags().StringVar(&billingsCreatePartnerID, "partner-id", "", "partner ID (required)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateBillingDate, "billing-date", "", "billing date YYYY-MM-DD (required)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateDepartment, "department-id", "", "department ID (auto-resolved if omitted)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateTitle, "title", "", "billing title")
	billingsCreateCmd.Flags().StringVar(&billingsCreateMemo, "memo", "", "memo")
	billingsCreateCmd.Flags().StringVar(&billingsCreatePaymentCond, "payment-condition", "", "payment condition")
	billingsCreateCmd.Flags().StringVar(&billingsCreateDueDate, "due-date", "", "due date YYYY-MM-DD")
	billingsCreateCmd.Flags().StringVar(&billingsCreateSalesDate, "sales-date", "", "sales date YYYY-MM-DD")
	billingsCreateCmd.Flags().StringArrayVar(&billingsCreateItemFlags, "item", nil, `line item: "name=X,price=N,quantity=N,excise=10"`)
	billingsCreateCmd.Flags().StringVar(&billingsCreateItemsFile, "items-file", "", "JSON file with line items")
	billingsCreateCmd.Flags().BoolVar(&billingsCreateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	billingsCreateCmd.Flags().BoolVar(&billingsCreateDryRun, "dry-run", false, "print request body without sending")
	_ = billingsCreateCmd.MarkFlagRequired("partner-id")
	_ = billingsCreateCmd.MarkFlagRequired("billing-date")

	billingsUpdateCmd.Flags().StringVar(&billingsUpdateTitle, "title", "", "billing title")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateMemo, "memo", "", "memo")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdatePaymentCond, "payment-condition", "", "payment condition")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateBillingDate, "billing-date", "", "billing date YYYY-MM-DD")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateDueDate, "due-date", "", "due date YYYY-MM-DD")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateSalesDate, "sales-date", "", "sales date YYYY-MM-DD")
	billingsUpdateCmd.Flags().BoolVar(&billingsUpdateDryRun, "dry-run", false, "print request body without sending")

	billingsDeleteCmd.Flags().BoolVar(&billingsDeleteYes, "yes", false, "skip confirmation prompt")

	billingsSetPaymentStatusCmd.Flags().StringVar(&billingsSetStatusValue, "status", "", "payment status (unsettled|settled) (required)")
	_ = billingsSetPaymentStatusCmd.MarkFlagRequired("status")

	billingsPDFCmd.Flags().BoolVar(&billingsPDFDownload, "download", false, "download PDF file")
	billingsPDFCmd.Flags().StringVar(&billingsPDFOutput, "output", "", "output file path")

	billingsCmd.AddCommand(billingsListCmd)
	billingsCmd.AddCommand(billingsShowCmd)
	billingsCmd.AddCommand(billingsCreateCmd)
	billingsCmd.AddCommand(billingsUpdateCmd)
	billingsCmd.AddCommand(billingsDeleteCmd)
	billingsCmd.AddCommand(billingsSetPaymentStatusCmd)
	billingsCmd.AddCommand(billingsPDFCmd)
}

// resolvePartnerID resolves --partner name to partner_id via search.
func resolvePartnerID(svc *api.InvoiceService, name string) (string, error) {
	partners, _, err := svc.ListPartners(pagination.Params{Page: 1, PerPage: 25}, name)
	if err != nil {
		return "", fmt.Errorf("searching partner %q: %w", name, err)
	}
	if len(partners) == 0 {
		return "", fmt.Errorf("no partner found matching %q", name)
	}
	if len(partners) > 1 {
		return "", fmt.Errorf("multiple partners found matching %q (%d results); use --partner-id instead", name, len(partners))
	}
	return partners[0].ID, nil
}

func runBillingsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve --partner name to partner_id.
	partnerID := billingsListPartnerID
	if billingsListPartner != "" {
		partnerID, err = resolvePartnerID(svc, billingsListPartner)
		if err != nil {
			return err
		}
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if billingsListAll {
		var allBillings []model.Billing
		page := 1
		for {
			opts := api.BillingListOptions{
				Params:        pagination.Params{Page: page, PerPage: 100},
				PartnerID:     partnerID,
				PaymentStatus: billingsListPaymentStatus,
				From:          billingsListFrom,
				To:            billingsListTo,
				Query:         billingsListQuery,
			}
			billings, pg, err := svc.ListBillings(opts)
			if err != nil {
				return err
			}
			allBillings = append(allBillings, billings...)
			if page >= pg.TotalPages {
				break
			}
			page++
			time.Sleep(400 * time.Millisecond)
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allBillings})
		}
		return f.Format(os.Stdout, allBillings)
	}

	opts := api.BillingListOptions{
		Params:        pagination.Params{Page: billingsListPage, PerPage: billingsListPerPage},
		PartnerID:     partnerID,
		PaymentStatus: billingsListPaymentStatus,
		From:          billingsListFrom,
		To:            billingsListTo,
		Query:         billingsListQuery,
	}
	billings, pg, err := svc.ListBillings(opts)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": billings, "pagination": pg})
	}
	return f.Format(os.Stdout, billings)
}

func runBillingsShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.GetBilling(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve line items.
	items, err := resolveLineItems(billingsCreateItemFlags, billingsCreateItemsFile, billingsCreateItemsStdin)
	if err != nil {
		return err
	}

	// Auto-resolve department_id if not provided.
	departmentID := billingsCreateDepartment
	if departmentID == "" {
		depts, err := svc.ListPartnerDepartments(billingsCreatePartnerID)
		if err != nil {
			return fmt.Errorf("resolving department: %w", err)
		}
		if len(depts) == 0 {
			return fmt.Errorf("partner has no departments registered; use --department-id")
		}
		departmentID = depts[0].ID
	}

	params := model.CreateBillingParams{
		DepartmentID:     departmentID,
		BillingDate:      billingsCreateBillingDate,
		Title:            billingsCreateTitle,
		Memo:             billingsCreateMemo,
		PaymentCondition: billingsCreatePaymentCond,
		DueDate:          billingsCreateDueDate,
		SalesDate:        billingsCreateSalesDate,
		Items:            items,
	}

	if billingsCreateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	billing, err := svc.CreateBilling(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateBillingParams
	if cmd.Flags().Changed("title") {
		params.Title = &billingsUpdateTitle
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &billingsUpdateMemo
	}
	if cmd.Flags().Changed("payment-condition") {
		params.PaymentCondition = &billingsUpdatePaymentCond
	}
	if cmd.Flags().Changed("billing-date") {
		params.BillingDate = &billingsUpdateBillingDate
	}
	if cmd.Flags().Changed("due-date") {
		params.DueDate = &billingsUpdateDueDate
	}
	if cmd.Flags().Changed("sales-date") {
		params.SalesDate = &billingsUpdateSalesDate
	}

	if billingsUpdateDryRun {
		wrapped := map[string]any{"billing": params}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(wrapped)
	}

	billing, err := svc.UpdateBilling(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsDelete(cmd *cobra.Command, args []string) error {
	if !billingsDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete billing %s?", args[0]))
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

	return svc.DeleteBilling(args[0])
}

func runBillingsSetPaymentStatus(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.SetPaymentStatus(args[0], model.PaymentStatus(billingsSetStatusValue))
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsPDF(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	pdfURL, err := svc.GetBillingPDF(args[0])
	if err != nil {
		return err
	}

	if !billingsPDFDownload && billingsPDFOutput == "" {
		fmt.Println(pdfURL)
		return nil
	}

	// Download PDF.
	resp, err := http.Get(pdfURL)
	if err != nil {
		return fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	outPath := billingsPDFOutput
	if outPath == "" {
		// Use billing ID as fallback filename.
		outPath = args[0] + ".pdf"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Downloaded to %s\n", outPath)
	return nil
}
```

- [ ] **Step 2: Register billings in invoice.go**

Add to `cmd/invoice/invoice.go` init():

```go
InvoiceCmd.AddCommand(billingsCmd)
```

- [ ] **Step 3: Verify it compiles and test**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: all pass

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/billings.go cmd/invoice/invoice.go
git commit -m "Add mf invoice billings commands with line items, dry-run, PDF, partner resolution"
```

---

### Task 8: Add `--all` to Partners List (Retroactive)

**Files:**
- Modify: `cmd/invoice/partners.go`

- [ ] **Step 1: Add --all flag and auto-pagination loop to partners list**

In `cmd/invoice/partners.go`, add the flag variable and registration:

```go
// Add to var block:
partnersListAll bool

// Add to init():
partnersListCmd.Flags().BoolVar(&partnersListAll, "all", false, "fetch all pages")
```

Update `runPartnersList` to handle `--all`:

```go
func runPartnersList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if partnersListAll {
		var allPartners []model.Partner
		page := 1
		for {
			params := pagination.Params{Page: page, PerPage: 100}
			partners, pg, err := svc.ListPartners(params, partnersListQuery)
			if err != nil {
				return err
			}
			allPartners = append(allPartners, partners...)
			if page >= pg.TotalPages {
				break
			}
			page++
			time.Sleep(400 * time.Millisecond)
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allPartners})
		}
		return f.Format(os.Stdout, allPartners)
	}

	params := pagination.Params{Page: partnersListPage, PerPage: partnersListPerPage}
	partners, pg, err := svc.ListPartners(params, partnersListQuery)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": partners, "pagination": pg})
	}
	return f.Format(os.Stdout, partners)
}
```

Add `"time"` to imports.

- [ ] **Step 2: Verify build and tests**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: all pass

- [ ] **Step 3: Commit**

```bash
git add cmd/invoice/partners.go
git commit -m "Add --all auto-pagination to partners list"
```

---

### Task 9: Final Verification + PR

- [ ] **Step 1: Run full test suite**

Run: `go build ./... && go test ./... -v && go vet ./...`
Expected: all tests pass

- [ ] **Step 2: Verify command tree**

Run: `go run . invoice items --help`
Expected: list, show, create, update, delete subcommands

Run: `go run . invoice billings --help`
Expected: list, show, create, update, delete, set-payment-status, pdf subcommands

- [ ] **Step 3: Push and create PR**

```bash
git push -u origin phase2b-items-billings
gh pr create --title "Phase 2b: Items + Billings" --body "..."
```
