# Phase 2c: Quotes + Sent-Histories Design

## Overview

Add Quote CRUD commands (8) and Sent-Histories list (1) to the `mf invoice` CLI. Reuses the billings pattern from Phase 2b with key differences in wrapping rules, status enum, and a new quote-to-billing conversion command.

## Commands

### Quotes (8 commands)

```bash
mf invoice quotes list [--page N] [--per-page N] [--partner-id <id>] [--partner <name>] [--status <s>] [--from <date>] [--to <date>] [--query <q>] [--all]
mf invoice quotes show <id>
mf invoice quotes create --partner-id <id> --quote-date <date> --expired-date <date> [--item "..." ...] [--items-file <path>] [--items-stdin] [--department-id <id>] [--title <text>] [--memo <text>] [--dry-run]
mf invoice quotes update <id> [--title <text>] [--memo <text>] [--quote-date <date>] [--expired-date <date>] [--items-file <path>] [--items-stdin] [--dry-run]
mf invoice quotes delete <id> [--yes]
mf invoice quotes set-status <id> --status <draft|sent|accepted|rejected|cancelled>
mf invoice quotes to-billing <id>
mf invoice quotes pdf <id> [--download] [--output <path>]
```

### Sent-Histories (1 command)

```bash
mf invoice sent-histories list [--page N] [--per-page N] [--all]
```

## API Mapping

| Command | Method | Path | Body |
|---------|--------|------|------|
| quotes list | GET | `/quotes` | — |
| quotes show | GET | `/quotes/{id}` | — |
| quotes create | POST | `/quotes` | Direct (no wrapping) |
| quotes update | PATCH | `/quotes/{id}` | Direct (no wrapping) |
| quotes delete | DELETE | `/quotes/{id}` | — |
| quotes set-status | PATCH | `/quotes/{id}` | Direct: `{"status": "<value>"}` |
| quotes to-billing | POST | `/quotes/{id}/convert_to_billing` | `{}` (empty object) |
| quotes pdf | GET | `/quotes/{id}` | — (reads pdf_url from response) |
| sent-histories list | GET | `/sent_histories` | — |

## Differences from Billings (Phase 2b)

| Aspect | Billings | Quotes |
|--------|----------|--------|
| Create endpoint | `POST /invoice_template_billings` | `POST /quotes` |
| Update wrapping | `{"billing": {...}}` | Direct (no wrapping) |
| Status command | `set-payment-status` (2 values) | `set-status` (5 values) |
| Required create fields | `--billing-date` | `--quote-date` + `--expired-date` |
| Conversion | — | `to-billing` (new) |

## Data Models

### New models in `internal/model/invoice.go`

```go
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

type CreateQuoteParams struct {
    DepartmentID string                `json:"department_id"`
    QuoteDate    string                `json:"quote_date"`
    ExpiredDate  string                `json:"expired_date"`
    Title        string                `json:"title,omitempty"`
    Memo         string                `json:"memo,omitempty"`
    Items        []InvoiceTemplateLine `json:"items,omitempty"`
}

type UpdateQuoteParams struct {
    Title       *string               `json:"title,omitempty"`
    Memo        *string               `json:"memo,omitempty"`
    QuoteDate   *string               `json:"quote_date,omitempty"`
    ExpiredDate *string               `json:"expired_date,omitempty"`
    Items       []InvoiceTemplateLine  `json:"items,omitempty"`
}

type SentHistory struct {
    ID          string `json:"id"`
    Type        string `json:"type,omitempty"`
    DocumentID  string `json:"document_id,omitempty"`
    Operator    string `json:"operator,omitempty"`
    SentAt      string `json:"sent_at,omitempty"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}
```

## API Service Methods

New methods in `internal/api/invoice.go`:

```go
// QuoteListOptions holds filter parameters for listing quotes.
type QuoteListOptions struct {
    pagination.Params
    PartnerID string
    Status    string
    From      string
    To        string
    Query     string
}

func (s *InvoiceService) ListQuotes(opts QuoteListOptions) ([]model.Quote, *pagination.Result, error)
func (s *InvoiceService) GetQuote(id string) (*model.Quote, error)
func (s *InvoiceService) CreateQuote(params model.CreateQuoteParams) (*model.Quote, error)
func (s *InvoiceService) UpdateQuote(id string, params model.UpdateQuoteParams) (*model.Quote, error)
func (s *InvoiceService) DeleteQuote(id string) error
func (s *InvoiceService) SetQuoteStatus(id string, status model.QuoteStatus) (*model.Quote, error)
func (s *InvoiceService) ConvertQuoteToBilling(id string) (*model.Billing, error)
func (s *InvoiceService) GetQuotePDF(id string) (string, error)
func (s *InvoiceService) ListSentHistories(params pagination.Params) ([]model.SentHistory, *pagination.Result, error)
```

## File Changes

| File | Action |
|------|--------|
| `internal/model/invoice.go` | Add Quote, QuoteItem, QuoteStatus, CreateQuoteParams, UpdateQuoteParams, SentHistory |
| `internal/api/invoice.go` | Add QuoteListOptions, 9 Quote/SentHistory methods |
| `internal/api/invoice_test.go` | Add httptest tests for new methods |
| `cmd/invoice/quotes.go` | New file: 8 quote commands |
| `cmd/invoice/sent_histories.go` | New file: 1 sent-histories command |
| `cmd/invoice/invoice.go` | Register quotesCmd and sentHistoriesCmd |

## Reuse from Phase 2b

- `resolvePartnerID()` — partner name resolution (already in billings.go, accessible within package)
- `resolveLineItems()` / `parseItemFlag()` — `--item`, `--items-file`, `--items-stdin` parsing
- `fetchAll[T]()` — `--all` auto-pagination
- `newInvoiceService()` — service initialization
- `DownloadPDF()` — PDF download (already on InvoiceService)
- `buildListQuery()` — shared query builder
- Excise aliases — already implemented

## Test Strategy

- httptest mock server tests for all new `InvoiceService` methods
- Verify request path, method, headers, and body content
- Verify response parsing for Quote, SentHistory structs
- `ConvertQuoteToBilling` test: verify empty `{}` body and Billing response parsing
