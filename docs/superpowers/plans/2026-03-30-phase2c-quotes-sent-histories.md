# Phase 2c: Quotes + Sent-Histories Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Quote CRUD commands (8) and Sent-Histories list (1) to the `mf invoice` CLI, reusing the billings pattern from Phase 2b.

**Architecture:** Follows the established InvoiceService pattern — models in `internal/model/invoice.go`, service methods in `internal/api/invoice.go`, cobra commands in `cmd/invoice/`. Quotes mirror billings with different wrapping rules (direct, no wrapping) and a new `to-billing` conversion command.

**Tech Stack:** Go, cobra, httptest, `net/url` for query building

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/model/invoice.go` | Modify | Add QuoteStatus enum, Quote, QuoteItem, CreateQuoteParams, UpdateQuoteParams, SentHistory |
| `internal/api/invoice.go` | Modify | Add QuoteListOptions, 9 service methods (ListQuotes, GetQuote, CreateQuote, UpdateQuote, DeleteQuote, SetQuoteStatus, ConvertQuoteToBilling, GetQuotePDF, ListSentHistories) |
| `internal/api/invoice_test.go` | Modify | Add httptest tests for all new service methods |
| `cmd/invoice/quotes.go` | Create | 8 quote cobra commands |
| `cmd/invoice/sent_histories.go` | Create | 1 sent-histories cobra command |
| `cmd/invoice/invoice.go` | Modify | Register quotesCmd and sentHistoriesCmd |

---

### Task 1: Add Quote and SentHistory models

**Files:**
- Modify: `internal/model/invoice.go`

- [ ] **Step 1: Add QuoteStatus enum, Quote, QuoteItem, CreateQuoteParams, UpdateQuoteParams, SentHistory**

Append to the end of `internal/model/invoice.go`:

```go
// --- Quote ---

type QuoteStatus string

const (
	QuoteStatusDraft     QuoteStatus = "draft"
	QuoteStatusSent      QuoteStatus = "sent"
	QuoteStatusAccepted  QuoteStatus = "accepted"
	QuoteStatusRejected  QuoteStatus = "rejected"
	QuoteStatusCancelled QuoteStatus = "cancelled"
)

type Quote struct {
	ID            string      `json:"id"`
	PDFURL        string      `json:"pdf_url,omitempty"`
	OperatorID    string      `json:"operator_id,omitempty"`
	DepartmentID  string      `json:"department_id,omitempty"`
	PartnerID     string      `json:"partner_id,omitempty"`
	PartnerName   string      `json:"partner_name,omitempty"`
	PartnerDetail string      `json:"partner_detail,omitempty"`
	Title         string      `json:"title,omitempty"`
	Memo          string      `json:"memo,omitempty"`
	QuoteNumber   string      `json:"quote_number,omitempty"`
	QuoteDate     string      `json:"quote_date,omitempty"`
	ExpiredDate   string      `json:"expired_date,omitempty"`
	Status        QuoteStatus `json:"status"`
	Subtotal      *int        `json:"subtotal,omitempty"`
	TotalPrice    *int        `json:"total_price,omitempty"`
	Tax           *int        `json:"tax,omitempty"`
	Items         []QuoteItem `json:"items"`
	CreatedAt     string      `json:"created_at"`
	UpdatedAt     string      `json:"updated_at"`
}

type QuoteItem struct {
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

// CreateQuoteParams is sent to POST /quotes (direct, no wrapping).
type CreateQuoteParams struct {
	DepartmentID string                `json:"department_id"`
	QuoteDate    string                `json:"quote_date"`
	ExpiredDate  string                `json:"expired_date"`
	Title        string                `json:"title,omitempty"`
	Memo         string                `json:"memo,omitempty"`
	Items        []InvoiceTemplateLine `json:"items,omitempty"`
}

// UpdateQuoteParams is sent to PATCH /quotes/{id} (direct, no wrapping).
type UpdateQuoteParams struct {
	Title       *string               `json:"title,omitempty"`
	Memo        *string               `json:"memo,omitempty"`
	QuoteDate   *string               `json:"quote_date,omitempty"`
	ExpiredDate *string               `json:"expired_date,omitempty"`
	Items       []InvoiceTemplateLine  `json:"items,omitempty"`
}

// --- Sent History ---

type SentHistory struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	Operator   string `json:"operator,omitempty"`
	SentAt     string `json:"sent_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go build ./internal/model/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/model/invoice.go
git commit -m "Add Quote, SentHistory models and QuoteStatus enum"
```

---

### Task 2: Add Quote service methods and tests (ListQuotes, GetQuote)

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write the failing tests for ListQuotes and GetQuote**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_ListQuotes(t *testing.T) {
	subtotal, total := 3000, 3300
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}
		if q.Get("partner_id") != "p1" {
			t.Errorf("partner_id = %q, want %q", q.Get("partner_id"), "p1")
		}
		if q.Get("status") != "draft" {
			t.Errorf("status = %q, want %q", q.Get("status"), "draft")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Quote{
				{ID: "q1", Title: "Quote 1", Status: model.QuoteStatusDraft, Subtotal: &subtotal, TotalPrice: &total, CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
			},
			"pagination": map[string]int{
				"total_count":  1,
				"total_pages":  1,
				"current_page": 1,
				"per_page":     25,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	opts := api.QuoteListOptions{
		Params:    pagination.Params{Page: 1, PerPage: 25},
		PartnerID: "p1",
		Status:    "draft",
	}
	quotes, pag, err := svc.ListQuotes(opts)
	if err != nil {
		t.Fatalf("ListQuotes() error: %v", err)
	}
	if len(quotes) != 1 {
		t.Errorf("len(quotes) = %d, want 1", len(quotes))
	}
	if quotes[0].ID != "q1" {
		t.Errorf("quotes[0].ID = %q, want %q", quotes[0].ID, "q1")
	}
	if quotes[0].Status != model.QuoteStatusDraft {
		t.Errorf("quotes[0].Status = %q, want %q", quotes[0].Status, model.QuoteStatusDraft)
	}
	if pag.TotalCount != 1 {
		t.Errorf("pag.TotalCount = %d, want 1", pag.TotalCount)
	}
}

func TestInvoiceService_GetQuote(t *testing.T) {
	subtotal, total := 5000, 5500
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:          "q1",
			Title:       "Test Quote",
			Status:      model.QuoteStatusSent,
			PDFURL:      "https://example.com/q1.pdf",
			QuoteDate:   "2024-06-01",
			ExpiredDate: "2024-07-01",
			Subtotal:    &subtotal,
			TotalPrice:  &total,
			CreatedAt:   "2024-01-01",
			UpdatedAt:   "2024-01-01",
		})
	})

	quote, err := svc.GetQuote("q1")
	if err != nil {
		t.Fatalf("GetQuote() error: %v", err)
	}
	if quote.ID != "q1" {
		t.Errorf("quote.ID = %q, want %q", quote.ID, "q1")
	}
	if quote.Title != "Test Quote" {
		t.Errorf("quote.Title = %q, want %q", quote.Title, "Test Quote")
	}
	if quote.PDFURL != "https://example.com/q1.pdf" {
		t.Errorf("quote.PDFURL = %q, want %q", quote.PDFURL, "https://example.com/q1.pdf")
	}
	if quote.Status != model.QuoteStatusSent {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusSent)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_ListQuotes|TestInvoiceService_GetQuote" -v`
Expected: FAIL — `QuoteListOptions` undefined, `ListQuotes`/`GetQuote` methods not found

- [ ] **Step 3: Implement QuoteListOptions, ListQuotes, and GetQuote**

Append to `internal/api/invoice.go`:

```go
// --- Quotes ---

// QuoteListOptions holds filter parameters for listing quotes.
type QuoteListOptions struct {
	pagination.Params
	PartnerID string
	Status    string
	From      string
	To        string
	Query     string
}

func (o QuoteListOptions) queryString() string {
	v := buildListQuery(o.Params, o.Query)
	if o.PartnerID != "" {
		v.Set("partner_id", o.PartnerID)
	}
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	if o.From != "" {
		v.Set("from", o.From)
	}
	if o.To != "" {
		v.Set("to", o.To)
	}
	return v.Encode()
}

func (s *InvoiceService) ListQuotes(opts QuoteListOptions) ([]model.Quote, *pagination.Result, error) {
	u := fmt.Sprintf("%s/quotes?%s", s.base, opts.queryString())
	var resp listResponse[model.Quote]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing quotes: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetQuote(id string) (*model.Quote, error) {
	var quote model.Quote
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/quotes/%s", s.base, id), nil, &quote)
	if err != nil {
		return nil, fmt.Errorf("getting quote: %w", err)
	}
	return &quote, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_ListQuotes|TestInvoiceService_GetQuote" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add ListQuotes and GetQuote service methods with tests"
```

---

### Task 3: Add CreateQuote, UpdateQuote, DeleteQuote service methods and tests

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_CreateQuote(t *testing.T) {
	subtotal, total := 10000, 11000
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		// POST /quotes (direct, no wrapping)
		if r.URL.Path != "/api/v3/quotes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		// Verify body is NOT wrapped — decode directly as CreateQuoteParams
		var body model.CreateQuoteParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.DepartmentID != "d1" {
			t.Errorf("body.DepartmentID = %q, want %q", body.DepartmentID, "d1")
		}
		if body.QuoteDate != "2024-06-01" {
			t.Errorf("body.QuoteDate = %q, want %q", body.QuoteDate, "2024-06-01")
		}
		if body.ExpiredDate != "2024-07-01" {
			t.Errorf("body.ExpiredDate = %q, want %q", body.ExpiredDate, "2024-07-01")
		}
		if len(body.Items) != 1 {
			t.Fatalf("len(body.Items) = %d, want 1", len(body.Items))
		}
		if body.Items[0].Name != "Consulting" {
			t.Errorf("body.Items[0].Name = %q, want %q", body.Items[0].Name, "Consulting")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:           "q-new",
			DepartmentID: body.DepartmentID,
			QuoteDate:    body.QuoteDate,
			ExpiredDate:  body.ExpiredDate,
			Status:       model.QuoteStatusDraft,
			Subtotal:     &subtotal,
			TotalPrice:   &total,
			CreatedAt:    "2024-06-01",
			UpdatedAt:    "2024-06-01",
		})
	})

	params := model.CreateQuoteParams{
		DepartmentID: "d1",
		QuoteDate:    "2024-06-01",
		ExpiredDate:  "2024-07-01",
		Items: []model.InvoiceTemplateLine{
			{Name: "Consulting", Price: 10000, Quantity: 1, Excise: "ten_percent"},
		},
	}
	quote, err := svc.CreateQuote(params)
	if err != nil {
		t.Fatalf("CreateQuote() error: %v", err)
	}
	if quote.ID != "q-new" {
		t.Errorf("quote.ID = %q, want %q", quote.ID, "q-new")
	}
	if quote.Status != model.QuoteStatusDraft {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusDraft)
	}
}

func TestInvoiceService_UpdateQuote(t *testing.T) {
	newTitle := "Updated Quote"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		// PATCH /quotes/{id} — direct (NO wrapping, unlike billings)
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify body is direct (not wrapped)
		var body model.UpdateQuoteParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Title == nil || *body.Title != "Updated Quote" {
			t.Errorf("body.Title = %v, want pointer to %q", body.Title, "Updated Quote")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			Title:     *body.Title,
			Status:    model.QuoteStatusDraft,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	params := model.UpdateQuoteParams{Title: &newTitle}
	quote, err := svc.UpdateQuote("q1", params)
	if err != nil {
		t.Fatalf("UpdateQuote() error: %v", err)
	}
	if quote.Title != "Updated Quote" {
		t.Errorf("quote.Title = %q, want %q", quote.Title, "Updated Quote")
	}
}

func TestInvoiceService_DeleteQuote(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := svc.DeleteQuote("q1")
	if err != nil {
		t.Fatalf("DeleteQuote() error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_(Create|Update|Delete)Quote" -v`
Expected: FAIL — methods not found

- [ ] **Step 3: Implement CreateQuote, UpdateQuote, DeleteQuote**

Append to `internal/api/invoice.go` (after GetQuote):

```go
// CreateQuote uses POST /quotes (direct, no wrapping).
func (s *InvoiceService) CreateQuote(params model.CreateQuoteParams) (*model.Quote, error) {
	var quote model.Quote
	err := s.client.DoJSON(http.MethodPost, s.base+"/quotes", params, &quote)
	if err != nil {
		return nil, fmt.Errorf("creating quote: %w", err)
	}
	return &quote, nil
}

// UpdateQuote uses PATCH /quotes/{id} (direct, no wrapping — unlike billings).
func (s *InvoiceService) UpdateQuote(id string, params model.UpdateQuoteParams) (*model.Quote, error) {
	var quote model.Quote
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/quotes/%s", s.base, id), params, &quote)
	if err != nil {
		return nil, fmt.Errorf("updating quote: %w", err)
	}
	return &quote, nil
}

func (s *InvoiceService) DeleteQuote(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/quotes/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting quote: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_(Create|Update|Delete)Quote" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add CreateQuote, UpdateQuote, DeleteQuote service methods with tests"
```

---

### Task 4: Add SetQuoteStatus, ConvertQuoteToBilling, GetQuotePDF service methods and tests

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_SetQuoteStatus(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify body sends status directly (no wrapping)
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["status"] != "sent" {
			t.Errorf("body[status] = %v, want %q", body["status"], "sent")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			Status:    model.QuoteStatusSent,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	quote, err := svc.SetQuoteStatus("q1", model.QuoteStatusSent)
	if err != nil {
		t.Fatalf("SetQuoteStatus() error: %v", err)
	}
	if quote.Status != model.QuoteStatusSent {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusSent)
	}
}

func TestInvoiceService_ConvertQuoteToBilling(t *testing.T) {
	subtotal, total := 10000, 11000
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1/convert_to_billing" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify body is empty object {}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("body should be empty, got %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b-converted",
			PaymentStatus: model.PaymentStatusUnsettled,
			Subtotal:      &subtotal,
			TotalPrice:    &total,
			CreatedAt:     "2024-06-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	billing, err := svc.ConvertQuoteToBilling("q1")
	if err != nil {
		t.Fatalf("ConvertQuoteToBilling() error: %v", err)
	}
	if billing.ID != "b-converted" {
		t.Errorf("billing.ID = %q, want %q", billing.ID, "b-converted")
	}
	if billing.PaymentStatus != model.PaymentStatusUnsettled {
		t.Errorf("billing.PaymentStatus = %q, want %q", billing.PaymentStatus, model.PaymentStatusUnsettled)
	}
}

func TestInvoiceService_GetQuotePDF(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			PDFURL:    "https://example.com/q1.pdf",
			Status:    model.QuoteStatusDraft,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-01",
		})
	})

	pdfURL, err := svc.GetQuotePDF("q1")
	if err != nil {
		t.Fatalf("GetQuotePDF() error: %v", err)
	}
	if pdfURL != "https://example.com/q1.pdf" {
		t.Errorf("pdfURL = %q, want %q", pdfURL, "https://example.com/q1.pdf")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_(SetQuoteStatus|ConvertQuoteToBilling|GetQuotePDF)" -v`
Expected: FAIL — methods not found

- [ ] **Step 3: Implement SetQuoteStatus, ConvertQuoteToBilling, GetQuotePDF**

Append to `internal/api/invoice.go` (after DeleteQuote):

```go
// SetQuoteStatus updates quote status via PATCH /quotes/{id} (direct, no wrapping).
func (s *InvoiceService) SetQuoteStatus(id string, status model.QuoteStatus) (*model.Quote, error) {
	body := map[string]any{"status": status}
	var quote model.Quote
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/quotes/%s", s.base, id), body, &quote)
	if err != nil {
		return nil, fmt.Errorf("setting quote status: %w", err)
	}
	return &quote, nil
}

// ConvertQuoteToBilling converts a quote to a billing via POST /quotes/{id}/convert_to_billing.
// Sends empty {} body.
func (s *InvoiceService) ConvertQuoteToBilling(id string) (*model.Billing, error) {
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPost, fmt.Sprintf("%s/quotes/%s/convert_to_billing", s.base, id), map[string]any{}, &billing)
	if err != nil {
		return nil, fmt.Errorf("converting quote to billing: %w", err)
	}
	return &billing, nil
}

// GetQuotePDF returns the PDF URL for a quote.
func (s *InvoiceService) GetQuotePDF(id string) (string, error) {
	quote, err := s.GetQuote(id)
	if err != nil {
		return "", err
	}
	return quote.PDFURL, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_(SetQuoteStatus|ConvertQuoteToBilling|GetQuotePDF)" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add SetQuoteStatus, ConvertQuoteToBilling, GetQuotePDF with tests"
```

---

### Task 5: Add ListSentHistories service method and test

**Files:**
- Modify: `internal/api/invoice.go`
- Modify: `internal/api/invoice_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/api/invoice_test.go`:

```go
func TestInvoiceService_ListSentHistories(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/sent_histories" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.SentHistory{
				{ID: "sh1", Type: "billing", DocumentID: "b1", Operator: "user@example.com", SentAt: "2024-06-01T10:00:00Z", CreatedAt: "2024-06-01", UpdatedAt: "2024-06-01"},
				{ID: "sh2", Type: "quote", DocumentID: "q1", Operator: "user@example.com", SentAt: "2024-06-02T10:00:00Z", CreatedAt: "2024-06-02", UpdatedAt: "2024-06-02"},
			},
			"pagination": map[string]int{
				"total_count":  2,
				"total_pages":  1,
				"current_page": 1,
				"per_page":     25,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	params := pagination.Params{Page: 1, PerPage: 25}
	histories, pag, err := svc.ListSentHistories(params)
	if err != nil {
		t.Fatalf("ListSentHistories() error: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("len(histories) = %d, want 2", len(histories))
	}
	if histories[0].ID != "sh1" {
		t.Errorf("histories[0].ID = %q, want %q", histories[0].ID, "sh1")
	}
	if histories[0].Type != "billing" {
		t.Errorf("histories[0].Type = %q, want %q", histories[0].Type, "billing")
	}
	if histories[1].DocumentID != "q1" {
		t.Errorf("histories[1].DocumentID = %q, want %q", histories[1].DocumentID, "q1")
	}
	if pag.TotalCount != 2 {
		t.Errorf("pag.TotalCount = %d, want 2", pag.TotalCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_ListSentHistories" -v`
Expected: FAIL — `ListSentHistories` method not found

- [ ] **Step 3: Implement ListSentHistories**

Append to `internal/api/invoice.go`:

```go
// --- Sent Histories ---

func (s *InvoiceService) ListSentHistories(params pagination.Params) ([]model.SentHistory, *pagination.Result, error) {
	u := s.base + "/sent_histories?" + buildListQuery(params, "").Encode()
	var resp listResponse[model.SentHistory]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing sent histories: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -run "TestInvoiceService_ListSentHistories" -v`
Expected: PASS

- [ ] **Step 5: Run all API tests to verify nothing broke**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./internal/api/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/api/invoice.go internal/api/invoice_test.go
git commit -m "Add ListSentHistories service method with test"
```

---

### Task 6: Add quotes cobra commands

**Files:**
- Create: `cmd/invoice/quotes.go`
- Modify: `cmd/invoice/invoice.go`

- [ ] **Step 1: Create `cmd/invoice/quotes.go`**

Create the file with all 8 quote commands:

```go
package invoice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var quotesCmd = &cobra.Command{
	Use:   "quotes",
	Short: "Quote operations",
}

// --- list ---

var (
	quotesListPage      int
	quotesListPerPage   int
	quotesListPartnerID string
	quotesListPartner   string
	quotesListStatus    string
	quotesListFrom      string
	quotesListTo        string
	quotesListQuery     string
	quotesListAll       bool
)

var quotesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quotes",
	RunE:  runQuotesList,
}

// --- show ---

var quotesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show quote details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesShow,
}

// --- create ---

var (
	quotesCreatePartnerID   string
	quotesCreateQuoteDate   string
	quotesCreateExpiredDate string
	quotesCreateDepartment  string
	quotesCreateTitle       string
	quotesCreateMemo        string
	quotesCreateItemFlags   []string
	quotesCreateItemsFile   string
	quotesCreateItemsStdin  bool
	quotesCreateDryRun      bool
)

var quotesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a quote",
	RunE:  runQuotesCreate,
}

// --- update ---

var (
	quotesUpdateTitle       string
	quotesUpdateMemo        string
	quotesUpdateQuoteDate   string
	quotesUpdateExpiredDate string
	quotesUpdateItemsFile   string
	quotesUpdateItemsStdin  bool
	quotesUpdateDryRun      bool
)

var quotesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a quote",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesUpdate,
}

// --- delete ---

var quotesDeleteYes bool

var quotesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a quote",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesDelete,
}

// --- set-status ---

var quotesSetStatusValue string

var quotesSetStatusCmd = &cobra.Command{
	Use:   "set-status <id>",
	Short: "Set quote status",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesSetStatus,
}

// --- to-billing ---

var quotesToBillingCmd = &cobra.Command{
	Use:   "to-billing <id>",
	Short: "Convert quote to billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesToBilling,
}

// --- pdf ---

var (
	quotesPDFDownload bool
	quotesPDFOutput   string
)

var quotesPDFCmd = &cobra.Command{
	Use:   "pdf <id>",
	Short: "Get quote PDF",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesPDF,
}

func init() {
	quotesListCmd.Flags().IntVar(&quotesListPage, "page", 1, "page number")
	quotesListCmd.Flags().IntVar(&quotesListPerPage, "per-page", 25, "items per page (max 100)")
	quotesListCmd.Flags().StringVar(&quotesListPartnerID, "partner-id", "", "filter by partner ID")
	quotesListCmd.Flags().StringVar(&quotesListPartner, "partner", "", "filter by partner name (resolved to ID)")
	quotesListCmd.Flags().StringVar(&quotesListStatus, "status", "", "filter by status (draft|sent|accepted|rejected|cancelled)")
	quotesListCmd.Flags().StringVar(&quotesListFrom, "from", "", "from date (YYYY-MM-DD)")
	quotesListCmd.Flags().StringVar(&quotesListTo, "to", "", "to date (YYYY-MM-DD)")
	quotesListCmd.Flags().StringVar(&quotesListQuery, "query", "", "search query")
	quotesListCmd.Flags().BoolVar(&quotesListAll, "all", false, "fetch all pages")

	quotesCreateCmd.Flags().StringVar(&quotesCreatePartnerID, "partner-id", "", "partner ID (required if --department-id is omitted)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateQuoteDate, "quote-date", "", "quote date YYYY-MM-DD (required)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateExpiredDate, "expired-date", "", "expiry date YYYY-MM-DD (required)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateDepartment, "department-id", "", "department ID (auto-resolved from --partner-id if omitted)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateTitle, "title", "", "quote title")
	quotesCreateCmd.Flags().StringVar(&quotesCreateMemo, "memo", "", "memo")
	quotesCreateCmd.Flags().StringArrayVar(&quotesCreateItemFlags, "item", nil, `line item: "name=X,price=N,quantity=N,excise=10"`)
	quotesCreateCmd.Flags().StringVar(&quotesCreateItemsFile, "items-file", "", "JSON or YAML file with line items")
	quotesCreateCmd.Flags().BoolVar(&quotesCreateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	quotesCreateCmd.Flags().BoolVar(&quotesCreateDryRun, "dry-run", false, "print request body without sending")
	_ = quotesCreateCmd.MarkFlagRequired("quote-date")
	_ = quotesCreateCmd.MarkFlagRequired("expired-date")

	quotesUpdateCmd.Flags().StringVar(&quotesUpdateTitle, "title", "", "quote title")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateMemo, "memo", "", "memo")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateQuoteDate, "quote-date", "", "quote date YYYY-MM-DD")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateExpiredDate, "expired-date", "", "expiry date YYYY-MM-DD")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateItemsFile, "items-file", "", "JSON or YAML file with line items")
	quotesUpdateCmd.Flags().BoolVar(&quotesUpdateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	quotesUpdateCmd.Flags().BoolVar(&quotesUpdateDryRun, "dry-run", false, "print request body without sending")

	quotesDeleteCmd.Flags().BoolVar(&quotesDeleteYes, "yes", false, "skip confirmation prompt")

	quotesSetStatusCmd.Flags().StringVar(&quotesSetStatusValue, "status", "", "quote status (draft|sent|accepted|rejected|cancelled) (required)")
	_ = quotesSetStatusCmd.MarkFlagRequired("status")

	quotesPDFCmd.Flags().BoolVar(&quotesPDFDownload, "download", false, "download PDF file")
	quotesPDFCmd.Flags().StringVar(&quotesPDFOutput, "output", "", "output file path")

	quotesCmd.AddCommand(quotesListCmd)
	quotesCmd.AddCommand(quotesShowCmd)
	quotesCmd.AddCommand(quotesCreateCmd)
	quotesCmd.AddCommand(quotesUpdateCmd)
	quotesCmd.AddCommand(quotesDeleteCmd)
	quotesCmd.AddCommand(quotesSetStatusCmd)
	quotesCmd.AddCommand(quotesToBillingCmd)
	quotesCmd.AddCommand(quotesPDFCmd)
}

func runQuotesList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve --partner name to partner_id.
	partnerID := quotesListPartnerID
	if quotesListPartner != "" {
		partnerID, err = resolvePartnerID(svc, quotesListPartner)
		if err != nil {
			return err
		}
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if quotesListAll {
		allQuotes, err := fetchAll(func(page int) ([]model.Quote, *pagination.Result, error) {
			opts := api.QuoteListOptions{
				Params:    pagination.Params{Page: page, PerPage: 100},
				PartnerID: partnerID,
				Status:    quotesListStatus,
				From:      quotesListFrom,
				To:        quotesListTo,
				Query:     quotesListQuery,
			}
			return svc.ListQuotes(opts)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allQuotes})
		}
		return f.Format(os.Stdout, allQuotes)
	}

	opts := api.QuoteListOptions{
		Params:    pagination.Params{Page: quotesListPage, PerPage: quotesListPerPage},
		PartnerID: partnerID,
		Status:    quotesListStatus,
		From:      quotesListFrom,
		To:        quotesListTo,
		Query:     quotesListQuery,
	}
	quotes, pg, err := svc.ListQuotes(opts)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": quotes, "pagination": pg})
	}
	return f.Format(os.Stdout, quotes)
}

func runQuotesShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	quote, err := svc.GetQuote(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve line items.
	items, err := resolveLineItems(quotesCreateItemFlags, quotesCreateItemsFile, quotesCreateItemsStdin)
	if err != nil {
		return err
	}

	// Auto-resolve department_id if not provided.
	departmentID := quotesCreateDepartment
	if departmentID == "" {
		if quotesCreatePartnerID == "" {
			return fmt.Errorf("either --department-id or --partner-id must be provided")
		}
		depts, err := svc.ListPartnerDepartments(quotesCreatePartnerID)
		if err != nil {
			return fmt.Errorf("resolving department: %w", err)
		}
		if len(depts) == 0 {
			return fmt.Errorf("partner has no departments registered; use --department-id")
		}
		if len(depts) > 1 {
			return fmt.Errorf("multiple departments found for partner; use --department-id to specify one")
		}
		departmentID = depts[0].ID
	}

	params := model.CreateQuoteParams{
		DepartmentID: departmentID,
		QuoteDate:    quotesCreateQuoteDate,
		ExpiredDate:  quotesCreateExpiredDate,
		Title:        quotesCreateTitle,
		Memo:         quotesCreateMemo,
		Items:        items,
	}

	if quotesCreateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	quote, err := svc.CreateQuote(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateQuoteParams
	if cmd.Flags().Changed("title") {
		params.Title = &quotesUpdateTitle
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &quotesUpdateMemo
	}
	if cmd.Flags().Changed("quote-date") {
		params.QuoteDate = &quotesUpdateQuoteDate
	}
	if cmd.Flags().Changed("expired-date") {
		params.ExpiredDate = &quotesUpdateExpiredDate
	}

	// Resolve line items for update.
	items, err := resolveLineItems(nil, quotesUpdateItemsFile, quotesUpdateItemsStdin)
	if err != nil {
		return err
	}
	if items != nil {
		params.Items = items
	}

	if quotesUpdateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	quote, err := svc.UpdateQuote(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesDelete(cmd *cobra.Command, args []string) error {
	if !quotesDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete quote %s?", args[0]))
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

	return svc.DeleteQuote(args[0])
}

func runQuotesSetStatus(cmd *cobra.Command, args []string) error {
	switch quotesSetStatusValue {
	case string(model.QuoteStatusDraft),
		string(model.QuoteStatusSent),
		string(model.QuoteStatusAccepted),
		string(model.QuoteStatusRejected),
		string(model.QuoteStatusCancelled):
	default:
		return fmt.Errorf("invalid quote status %q: must be draft, sent, accepted, rejected, or cancelled", quotesSetStatusValue)
	}

	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	quote, err := svc.SetQuoteStatus(args[0], model.QuoteStatus(quotesSetStatusValue))
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesToBilling(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.ConvertQuoteToBilling(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runQuotesPDF(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	pdfURL, err := svc.GetQuotePDF(args[0])
	if err != nil {
		return err
	}
	if pdfURL == "" {
		return fmt.Errorf("quote %s has no PDF URL available", args[0])
	}

	if !quotesPDFDownload && quotesPDFOutput == "" {
		fmt.Println(pdfURL)
		return nil
	}

	// Download PDF using plain client (PDF URLs are signed/external).
	resp, err := svc.DownloadPDF(pdfURL)
	if err != nil {
		return fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading PDF: unexpected status %d", resp.StatusCode)
	}

	outPath := quotesPDFOutput
	if outPath == "" {
		outPath = args[0] + ".pdf"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(outPath)
		return fmt.Errorf("writing PDF: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing PDF file: %w", closeErr)
	}

	fmt.Fprintf(os.Stderr, "Downloaded to %s\n", outPath)
	return nil
}
```

- [ ] **Step 2: Register quotesCmd in `cmd/invoice/invoice.go`**

Add `InvoiceCmd.AddCommand(quotesCmd)` to the `init()` function in `cmd/invoice/invoice.go`:

```go
func init() {
	InvoiceCmd.AddCommand(officeCmd)
	InvoiceCmd.AddCommand(partnersCmd)
	partnersCmd.AddCommand(partnersDepartmentsCmd)
	InvoiceCmd.AddCommand(itemsCmd)
	InvoiceCmd.AddCommand(billingsCmd)
	InvoiceCmd.AddCommand(quotesCmd)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go build ./cmd/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/quotes.go cmd/invoice/invoice.go
git commit -m "Add quotes cobra commands (list, show, create, update, delete, set-status, to-billing, pdf)"
```

---

### Task 7: Add sent-histories cobra command

**Files:**
- Create: `cmd/invoice/sent_histories.go`
- Modify: `cmd/invoice/invoice.go`

- [ ] **Step 1: Create `cmd/invoice/sent_histories.go`**

```go
package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

var sentHistoriesCmd = &cobra.Command{
	Use:   "sent-histories",
	Short: "Sent history operations",
}

var (
	sentHistoriesListPage    int
	sentHistoriesListPerPage int
	sentHistoriesListAll     bool
)

var sentHistoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sent histories",
	RunE:  runSentHistoriesList,
}

func init() {
	sentHistoriesListCmd.Flags().IntVar(&sentHistoriesListPage, "page", 1, "page number")
	sentHistoriesListCmd.Flags().IntVar(&sentHistoriesListPerPage, "per-page", 25, "items per page (max 100)")
	sentHistoriesListCmd.Flags().BoolVar(&sentHistoriesListAll, "all", false, "fetch all pages")

	sentHistoriesCmd.AddCommand(sentHistoriesListCmd)
}

func runSentHistoriesList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if sentHistoriesListAll {
		allHistories, err := fetchAll(func(page int) ([]model.SentHistory, *pagination.Result, error) {
			return svc.ListSentHistories(pagination.Params{Page: page, PerPage: 100})
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allHistories})
		}
		return f.Format(os.Stdout, allHistories)
	}

	params := pagination.Params{Page: sentHistoriesListPage, PerPage: sentHistoriesListPerPage}
	histories, pg, err := svc.ListSentHistories(params)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": histories, "pagination": pg})
	}
	return f.Format(os.Stdout, histories)
}
```

- [ ] **Step 2: Register sentHistoriesCmd in `cmd/invoice/invoice.go`**

Add `InvoiceCmd.AddCommand(sentHistoriesCmd)` to the `init()` function:

```go
func init() {
	InvoiceCmd.AddCommand(officeCmd)
	InvoiceCmd.AddCommand(partnersCmd)
	partnersCmd.AddCommand(partnersDepartmentsCmd)
	InvoiceCmd.AddCommand(itemsCmd)
	InvoiceCmd.AddCommand(billingsCmd)
	InvoiceCmd.AddCommand(quotesCmd)
	InvoiceCmd.AddCommand(sentHistoriesCmd)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go build ./cmd/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/invoice/sent_histories.go cmd/invoice/invoice.go
git commit -m "Add sent-histories list cobra command"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run all tests**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go test ./... -v`
Expected: All tests PASS

- [ ] **Step 2: Run linter**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go vet ./...`
Expected: no issues

- [ ] **Step 3: Verify build**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go build -o /dev/null .`
Expected: no errors

- [ ] **Step 4: Verify command tree**

Run: `cd /root/dev/planitai/planitai-moneyforward-cli && go run . invoice quotes --help && go run . invoice sent-histories --help`
Expected: shows quotes subcommands (list, show, create, update, delete, set-status, to-billing, pdf) and sent-histories subcommands (list)
