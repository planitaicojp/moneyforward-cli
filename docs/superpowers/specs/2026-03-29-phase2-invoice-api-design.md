# Phase 2: Cloud Invoice API Design

## Overview

Implement Cloud Invoice API commands for the `mf` CLI. Invoice API (v3) provides billing, quote, partner, and item management for Money Forward Cloud Invoice.

**Base URL**: `https://invoice.moneyforward.com/api/v3`
**Auth**: OAuth2 Bearer token (scopes: `mfc/invoice/data.read mfc/invoice/data.write`)
**Rate Limit**: 3 requests/second

## PR Split Strategy

Phase 2 is delivered in 3 PRs:

| PR | Scope | Purpose |
|----|-------|---------|
| **2a** | Foundation + office + partners | Establish API client pattern, pagination, model, output integration |
| **2b** | items + billings | CRUD + `--item`/`--items-file`, excise aliases, department_id auto-resolution, `--dry-run`, PDF |
| **2c** | quotes + sent-histories | Reuse billings pattern, quote-to-billing conversion |

## Architecture

### API Client Pattern

`InvoiceService` wraps the Phase 1 `Client` with Invoice-specific methods:

```go
// internal/api/invoice.go
type InvoiceService struct {
    client *Client
    base   string // "https://invoice.moneyforward.com/api/v3"
}

func NewInvoiceService(token, version string, verbose bool) *InvoiceService
```

Each resource gets typed methods:

```go
func (s *InvoiceService) ListPartners(opts ListPartnersOptions) ([]model.Partner, *pagination.Result, error)
func (s *InvoiceService) GetPartner(id string) (*model.Partner, error)
func (s *InvoiceService) CreatePartner(params model.CreatePartnerParams) (*model.Partner, error)
func (s *InvoiceService) UpdatePartner(id string, params model.UpdatePartnerParams) (*model.Partner, error)
func (s *InvoiceService) DeletePartner(id string) error
```

This pattern repeats for Items, Billings, Quotes, and is reused in Phase 3-6 (`ExpenseService`, `PayableService`, etc.).

### Request Body Wrapping Rules

Wrapping varies by endpoint (critical to get right):

| Method | Path | Wrapping |
|--------|------|----------|
| POST | `/partners`, `/items` | Direct (no wrapping) |
| PATCH | `/partners/{id}`, `/items/{id}` | Direct (no wrapping) |
| POST | `/invoice_template_billings` | Direct (Invoice Act endpoint) |
| PATCH | `/billings/{id}` | `{"billing": {...}}` |
| POST | `/quotes` | Direct |
| PATCH | `/quotes/{id}` | `{"quote": {...}}` |
| POST | `/quotes/{id}/convert_to_billing` | Empty `{}` |

### Pagination

Page-based pagination shared across Invoice, Expense, and Payable APIs:

```go
// internal/pagination/pagination.go
type Params struct {
    Page    int
    PerPage int
}

type Result struct {
    TotalCount  int `json:"total_count"`
    TotalPages  int `json:"total_pages"`
    CurrentPage int `json:"current_page"`
    PerPage     int `json:"per_page"`
}

func (p Params) QueryString() string // "page=1&per_page=25"
```

- `--page` (default: 1), `--per-page` (default: 25, max: 100)
- `--all` auto-pagination deferred to PR 2b

## File Structure

```
internal/
├── api/
│   └── invoice.go              # InvoiceService
├── model/
│   └── invoice.go              # All Invoice structs + enums
└── pagination/
    └── pagination.go           # Shared pagination

cmd/invoice/
├── invoice.go                  # "mf invoice" root command
├── office.go                   # office show
├── partners.go                 # partners list|show|create|update|delete
├── partners_departments.go     # partners departments list
├── items.go                    # items list|show|create|update|delete
├── billings.go                 # billings list|show|create|update|delete|set-payment-status|pdf
├── quotes.go                   # quotes list|show|create|update|delete|set-status|to-billing|pdf
└── sent_histories.go           # sent-histories list
```

## Data Models

### Response structs (read)

Map API JSON directly. All Invoice API uses snake_case JSON keys.

```go
type Office struct {
    Name        string `json:"name"`
    Zip         string `json:"zip"`
    Prefecture  string `json:"prefecture"`
    Address1    string `json:"address1"`
    Address2    string `json:"address2"`
    Tel         string `json:"tel"`
    Fax         string `json:"fax"`
}

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
```

Full struct definitions for Item, Billing, Quote follow the same pattern per SPEC.md Section 2.4.

### Request structs (write)

Separate from response structs to exclude read-only fields:

```go
type CreatePartnerParams struct {
    Name       string `json:"name"`
    NameKana   string `json:"name_kana,omitempty"`
    NameSuffix string `json:"name_suffix,omitempty"`
    Code       string `json:"code,omitempty"`
    Memo       string `json:"memo,omitempty"`
}

type UpdatePartnerParams struct {
    Name       *string `json:"name,omitempty"`
    NameKana   *string `json:"name_kana,omitempty"`
    NameSuffix *string `json:"name_suffix,omitempty"`
    Code       *string `json:"code,omitempty"`
    Memo       *string `json:"memo,omitempty"`
}
```

Update params use `*string` so omitted flags don't send empty values that overwrite existing data.

### Enums

```go
type PaymentStatus string
const (
    PaymentStatusUnsettled PaymentStatus = "unsettled"
    PaymentStatusSettled   PaymentStatus = "settled"
)

type QuoteStatus string
const (
    QuoteStatusDraft     QuoteStatus = "draft"
    QuoteStatusSent      QuoteStatus = "sent"
    QuoteStatusAccepted  QuoteStatus = "accepted"
    QuoteStatusRejected  QuoteStatus = "rejected"
    QuoteStatusCancelled QuoteStatus = "cancelled"
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
```

## Command Definitions

### PR 2a Commands (7 commands)

```bash
mf invoice office show

mf invoice partners list [--page N] [--per-page N] [--query <q>]
mf invoice partners show <id>
mf invoice partners create --name <name> [--name-kana <kana>] [--code <code>] [--memo <text>]
mf invoice partners update <id> [--name <name>] [--code <code>] [--memo <text>]
mf invoice partners delete <id> [--yes]

mf invoice partners departments list <partner-id>
```

### PR 2b Commands (12 commands)

```bash
mf invoice items list [--page N] [--per-page N] [--query <q>] [--all]
mf invoice items show <id>
mf invoice items create --name <name> [--code <code>] [--price <n>] [--quantity <n>] [--excise <type>]
mf invoice items update <id> [--name <name>] [--code <code>] [--price <n>]
mf invoice items delete <id> [--yes]

mf invoice billings list [--page N] [--per-page N] [--partner-id <id>] [--payment-status <s>] [--from <date>] [--to <date>] [--query <q>] [--all]
mf invoice billings show <id>
mf invoice billings create --partner-id <id> --billing-date <YYYY-MM-DD> [--item "..." ...] [--items-file <path>] [--department-id <id>] [--dry-run]
mf invoice billings update <id> [--title <text>] [--memo <text>] [--items-file <path>] [--dry-run]
mf invoice billings delete <id> [--yes]
mf invoice billings set-payment-status <id> --status <unsettled|settled>
mf invoice billings pdf <id> [--download] [--output <path>]
```

### PR 2c Commands (9 commands)

```bash
mf invoice quotes list [--page N] [--per-page N] [--partner-id <id>] [--status <s>] [--from <date>] [--to <date>] [--query <q>] [--all]
mf invoice quotes show <id>
mf invoice quotes create --partner-id <id> --quote-date <date> --expired-date <date> [--item "..." ...] [--items-file <path>] [--dry-run]
mf invoice quotes update <id> [--title <text>] [--memo <text>] [--items-file <path>] [--dry-run]
mf invoice quotes delete <id> [--yes]
mf invoice quotes set-status <id> --status <draft|sent|accepted|rejected|cancelled>
mf invoice quotes to-billing <id>
mf invoice quotes pdf <id> [--download] [--output <path>]

mf invoice sent-histories list [--page N] [--per-page N] [--all]
```

## Command Behavior Details

### Output Integration

- All list/show commands respect `--format` global flag (table/json/yaml/csv)
- Table mode: tabwriter with header row, pagination info in footer
- JSON mode: raw API response (preserves envelope with pagination)
- Delete commands output nothing on success (exit 0), error on failure

### Authentication Flow

Each invoice command:
1. Resolves profile via `cmdutil.GetProfile(cmd)`
2. Gets valid token via `cmdutil.EnsureValidToken(profile, api.Services["invoice"])`
3. Creates `InvoiceService` with the token
4. Executes the API call

### Error Handling

- API errors: `cerrors.APIError` (from Phase 1 client.DoJSON)
- Auth errors: `cerrors.AuthError` (token missing/expired)
- Validation: cobra `MarkFlagRequired` + `cerrors.ValidationError`
- Delete confirmation: `prompt.Confirm` unless `--yes`

## PR 2b Specific Features

### Line Item Input (`--item` flag)

Repeatable flag with key=value format:

```bash
--item "name=Consulting,price=100000,quantity=1,excise=10"
```

Excise short aliases (ADR-011):

| Input | Maps To |
|-------|---------|
| `10` | `ten_percent` |
| `8` | `eight_percent` |
| `8r` | `eight_percent_as_reduced_tax_rate` |
| `5` | `five_percent` |
| `0` | `untaxable` |
| `exempt` | `tax_exemption` |
| `non` | `non_taxable` |

### `--items-file` Input

JSON or YAML file containing line items array. Detected by file extension.

### `--dry-run`

Prints the request body to stdout without making the API call. Useful for Agent workflows that want to inspect before committing.

### department_id Auto-Resolution (ADR-010)

When creating billings/quotes without `--department-id`:
1. `GET /partners/{id}/departments`
2. Use `departments[0].id`
3. Error if empty: "partner has no departments registered"

### PDF Download

`mf invoice billings pdf <id>`:
- Default: prints the PDF URL to stdout
- `--download`: downloads to `<billing_number>.pdf`
- `--output <path>`: downloads to specified path

## Test Strategy

| Layer | Method | PR |
|-------|--------|-----|
| `internal/api/invoice.go` | httptest mock server — verify request params/headers, response parsing | 2a, 2b, 2c |
| `internal/pagination/` | Unit tests — QueryString, edge cases | 2a |
| `internal/model/` | JSON marshal/unmarshal round-trip if needed | As needed |
| `cmd/invoice/` | No unit tests (cobra wiring tested via E2E in Phase 7) | — |

## Non-Goals for Phase 2

- Rate limiter (3 req/sec) — rely on existing 429 retry logic in Phase 1 client
- Webhook support
- Batch operations
- Accounting API integration (closed API)
