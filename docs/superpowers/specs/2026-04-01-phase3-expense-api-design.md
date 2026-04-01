# Phase 3: Cloud Expense API Design

> Approved 2026-04-01. OpenAPI spec verified against `expense.moneyforward.com/api/index.json`.

## Overview

Add Cloud Expense API support to the `mf` CLI, following the same model/api/cmd pattern established in Phase 2 (Invoice). Expense API requires `{office_id}` in all endpoint paths (unlike Invoice), and uses v1/v2 mixed base URLs.

## Architecture

```
internal/model/expense.go        — All Expense data types
internal/api/expense.go          — ExpenseService (v1 + v2 base URLs)
cmd/expense/expense.go           — Root command + newExpenseService + resolveOfficeID
cmd/expense/offices.go           — offices list
cmd/expense/departments.go       — departments list, show (API: depts)
cmd/expense/projects.go          — projects list, show
cmd/expense/categories.go        — categories list, show (API: ex_items)
cmd/expense/taxes.go             — taxes list, show (API: excises)
cmd/expense/positions.go         — positions list, show
cmd/expense/members.go           — members list, show, me (v2)
cmd/expense/transactions.go      — transactions list, show, create, update, delete + --scope org
cmd/expense/reports.go           — reports list, show + --scope org
cmd/expense/approvals.go         — approvals list, approve, reject
cmd/expense/journals.go          — journals list --by transactions|reports
cmd/cmdutil/pagination.go        — fetchAll (extracted from cmd/invoice)
```

## Key Design Decisions

### 1. office_id Resolution (Auto-detect + Flag)

Every Expense API endpoint requires `{office_id}` in the path. Resolution order:

1. `--office-id` flag (explicit override)
2. Profile config (`expense.office_id`)
3. Auto-detect: call `GET /offices`, if exactly 1 result, use it. If multiple, error with list of available offices.

```go
func resolveOfficeID(cmd *cobra.Command, svc *api.ExpenseService) (string, error)
```

### 2. ExpenseService with v1/v2 Base URLs

```go
type ExpenseService struct {
    client *Client
    base   string // v1: https://expense.moneyforward.com/api/external/v1
    baseV2 string // v2: https://expense.moneyforward.com/api/external/v2
}
```

- Most endpoints use `base` (v1)
- `office_members` and `/me` use `baseV2` (v2)
- v1 `office_members` is deprecated (2021/05), only v2 is used

### 3. CLI Command Names (User-Friendly)

| CLI command | API resource | Reason |
|-------------|-------------|--------|
| `departments` | `depts` | More intuitive |
| `categories` | `ex_items` | User-friendly (expense categories) |
| `taxes` | `excises` | User-friendly (tax classifications) |
| `transactions` | `ex_transactions` | Drop `ex_` prefix |
| `reports` | `ex_reports` | Drop `ex_` prefix |
| `approvals` | `approving_ex_reports` | Simplified |

### 4. fetchAll Extraction

Move `fetchAll[T]` from `cmd/invoice/pagination_helper.go` to `cmd/cmdutil/pagination.go`. Update imports in `cmd/invoice/`. Reuse in `cmd/expense/`.

### 5. Scope Pattern (--scope org)

For `transactions` and `reports`, `--scope org` switches to organization-wide endpoints:

| Command | Default (personal) | --scope org (admin) |
|---------|-------------------|---------------------|
| transactions list | `/offices/{id}/me/ex_transactions` | `/offices/{id}/ex_transactions` |
| reports list | `/offices/{id}/me/ex_reports` | `/offices/{id}/ex_reports` |
| approvals list | `/offices/{id}/me/approving_ex_reports` | `/offices/{id}/approving_ex_reports` |

### 6. Approval Actions

- Approve: `POST /offices/{office_id}/me/approving_ex_reports/{ex_report_id}/approve` (body: `message`)
- Reject: `POST /offices/{office_id}/me/approving_ex_reports/{ex_report_id}/disapprove` (body: `message`, `wf_step`)

Note: reject action maps to API `disapprove` endpoint.

## Sub-phases

### Phase 3a: OAuth + Offices + Master Data

**Scope**: Foundation + read-only list/show commands for master data.

**Files**:
- `internal/api/expense.go` — ExpenseService with v1/v2 URLs, ListOffices, ListDepts, GetDept, ListProjects, GetProject, ListExItems, GetExItem, ListExcises, GetExcise, ListPositions, GetPosition
- `internal/model/expense.go` — Office, Dept, Project, ExItem, Excise, Position
- `cmd/expense/expense.go` — Root command, newExpenseService, resolveOfficeID
- `cmd/expense/offices.go` — list
- `cmd/expense/departments.go` — list, show
- `cmd/expense/projects.go` — list, show
- `cmd/expense/categories.go` — list, show
- `cmd/expense/taxes.go` — list, show
- `cmd/expense/positions.go` — list, show
- `cmd/cmdutil/pagination.go` — fetchAll extracted from cmd/invoice

**Depends on**: Phase 1 auth (`api.Services["expense"]` already registered)

### Phase 3b: Members + Transactions

**Scope**: Employee lookup (v2) and expense transaction CRUD with --scope org.

**Files**:
- `internal/model/expense.go` += OfficeMemberV2, ExTransaction, ExTransactionCreateInput, ExTransactionUpdateInput
- `internal/api/expense.go` += ListMembersV2, GetMemberV2, GetMe, ListMyTransactions, GetMyTransaction, CreateMyTransaction, UpdateMyTransaction, DeleteMyTransaction, ListOrgTransactions, GetOrgTransaction, UpdateOrgTransaction, DeleteOrgTransaction
- `cmd/expense/members.go` — list, show, me
- `cmd/expense/transactions.go` — list, show, create, update, delete (--scope org)

### Phase 3c: Reports + Approvals

**Scope**: Expense report viewing and approval workflow.

**Files**:
- `internal/model/expense.go` += ExReport
- `internal/api/expense.go` += ListMyReports, GetMyReport, ListOrgReports, GetOrgReport, ListMyApprovals, ListOrgApprovals, ApproveReport, RejectReport
- `cmd/expense/reports.go` — list, show (--scope org)
- `cmd/expense/approvals.go` — list (--scope org), approve, reject

### Phase 3d: Journals

**Scope**: Journal entry listing with --by filter.

**Files**:
- `internal/model/expense.go` += ExJournal
- `internal/api/expense.go` += ListJournalsByTransactions, ListJournalsByReports
- `cmd/expense/journals.go` — list --by transactions|reports (--from, --to)

## Data Models

See SPEC.md Section 3.4 for complete field definitions verified against OpenAPI spec.

## Query Parameters

See SPEC.md Section 3.3 for transaction and report list filter parameters (`query_object[...]` format).

## Gotchas

- All paths require `{office_id}` — resolved via `resolveOfficeID()`
- v1/v2 mixed: members and /me use v2, everything else uses v1
- Approve is POST (not PATCH), reject endpoint is named `disapprove`
- `recognized_at` filter on journals is limited to max 3 months range
- Master data (categories, taxes) requires `public_resource:read` scope
- Org-level endpoints require admin role, general users get 403
