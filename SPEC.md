# Money Forward API Specification

実装時の参照ドキュメント。ref リポジトリ (`mf-invoice-mcp`, `admina-mcp-server`) の調査結果に基づく。

> **ドキュメント体系**:
> - `CLAUDE.md` — AI コンテキスト (プロジェクト概要、アーキテクチャ方針)
> - `DESIGN.md` — CLI 設計 (コマンド・フラグ・フェーズ・ADR)
> - `SPEC.md` — **API 仕様参照** (エンドポイント、データモデル、enum、認証詳細)

---

## 1. 共通仕様

### 1.1 認証方式一覧

| サービス | 認証方式 | 詳細 |
|---------|---------|------|
| Invoice | OAuth2 Authorization Code Grant + PKCE | `api.biz.moneyforward.com` |
| Expense | OAuth2 Authorization Code Grant + PKCE | `expense.moneyforward.com/oauth/` |
| Payable | OAuth2 Authorization Code Grant + PKCE | `payable.moneyforward.com/oauth/` |
| Payroll | IP ホワイトリスト + API Identifier | 別認証体系 |
| Admina | API Key + Organization ID | Bearer トークン |

### 1.2 OAuth2 フロー詳細

#### Authorization Code Grant + PKCE

**Authorization URL パラメータ**:

```
GET https://api.biz.moneyforward.com/authorize
  ?response_type=code
  &client_id={CLIENT_ID}
  &redirect_uri=http://localhost:{PORT}/callback
  &scope=mfc/invoice/data.read mfc/invoice/data.write
  &code_challenge={CODE_CHALLENGE}
  &code_challenge_method=S256
```

**Token Exchange (POST)**:

```
POST https://api.biz.moneyforward.com/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code={AUTHORIZATION_CODE}
&client_id={CLIENT_ID}
&client_secret={CLIENT_SECRET}
&redirect_uri=http://localhost:{PORT}/callback
&code_verifier={CODE_VERIFIER}
```

> **重要**: クライアント認証方式は `client_secret_post` (リクエストボディに含める)。
> `client_secret_basic` (Authorization ヘッダー) ではない。

**Token Refresh (POST)**:

```
POST https://api.biz.moneyforward.com/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={REFRESH_TOKEN}
&client_id={CLIENT_ID}
&client_secret={CLIENT_SECRET}
```

**Token Response**:

```json
{
  "access_token": "string",
  "refresh_token": "string",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "mfc/invoice/data.read mfc/invoice/data.write"
}
```

**Go struct**:

```go
type OAuthTokens struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    ExpiresAt    int64  `json:"expires_at,omitempty"` // Unix ms (計算値: now + expires_in*1000)
    Scope        string `json:"scope"`
}
```

**トークン有効期限**:

| トークン | 有効期限 |
|---------|---------|
| access_token | 3600秒 (1時間) |
| refresh_token | 540日 |

- 自動リフレッシュ: 期限5分前にトリガー (`now > expires_at - 5*60*1000`)
- refresh_token 期限切れ時は再認証が必要

#### スコープ一覧 (サービス別)

| サービス | スコープ |
|---------|---------|
| Invoice | `mfc/invoice/data.read`, `mfc/invoice/data.write` |
| Expense | `office_setting:write`, `transaction:write`, `report:write`, `user_setting:write`, `account:write`, `public_resource:read` |
| Payable | Expense と同一 (`mf_expense_oauth`) |

### 1.3 HTTP リクエスト規約

```
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
User-Agent: mf-cli/{version}
```

### 1.4 ページネーション

#### Page-based (Invoice / Expense / Payable)

**リクエスト**:

| パラメータ | 型 | デフォルト | 説明 |
|-----------|---|----------|------|
| `page` | int | 1 | ページ番号 |
| `per_page` | int | 25 | 1ページあたりの件数 |

**レスポンス**:

```json
{
  "data": [...],
  "pagination": {
    "total_count": 150,
    "total_pages": 6,
    "current_page": 1,
    "per_page": 25
  }
}
```

**Go struct**:

```go
type Pagination struct {
    TotalCount  int `json:"total_count"`
    TotalPages  int `json:"total_pages"`
    CurrentPage int `json:"current_page"`
    PerPage     int `json:"per_page"`
}

type ListResponse[T any] struct {
    Data       []T        `json:"data"`
    Pagination Pagination `json:"pagination"`
}
```

#### Cursor-based (Admina)

**リクエスト**:

| パラメータ | 型 | デフォルト | 説明 |
|-----------|---|----------|------|
| `limit` | int | — | 最大件数 (max 200) |
| `cursor` | string | — | Base64 エンコードされたカーソル |

**レスポンス**: 次のページがある場合、レスポンスに `cursor` フィールドが含まれる。

### 1.5 レート制限

| サービス | 制限 | HTTP ステータス | リトライ |
|---------|------|----------------|---------|
| Invoice | 3 req/sec | 429 Too Many Requests | `Retry-After` ヘッダーに従う |
| Admina | 不明 | — | — |

### 1.6 エラーレスポンス形式

#### Invoice / Expense / Payable

```json
{
  "code": "string",
  "message": "string",
  "errors": {
    "field_name": ["error message 1", "error message 2"]
  }
}
```

**Go struct**:

```go
type APIError struct {
    Code    string              `json:"code"`
    Message string              `json:"message"`
    Errors  map[string][]string `json:"errors,omitempty"`
}
```

**HTTP ステータス → Go エラー型マッピング**:

| HTTP Status | 終了コード | エラー型 |
|------------|----------|---------|
| 401 | 2 | AuthError |
| 403 | 2 | AuthError |
| 404 | 3 | NotFoundError |
| 422 | 4 | ValidationError |
| 429 | 5 | RateLimitError → リトライ |
| 5xx | 5 | APIError |

#### Admina

```json
{
  "errorId": "string",
  "errorDetails": {},
  "status": 400
}
```

| HTTP Status | エラー |
|------------|--------|
| 400 | InvalidRequestError |
| 401 | AuthenticationError |
| 403 | PermissionError |
| 404 | NotFoundError |
| 408 | RequestTimeoutError |
| 422 | ValidationError |
| 500 | SystemError |
| 504 | TimeoutError |

### 1.7 日付形式

- `YYYY-MM-DD` (ISO 8601) — billing_date, due_date, quote_date 等
- `YYYY-MM-DDTHH:MM:SS.sssZ` (ISO 8601) — created_at, updated_at

---

## 2. Invoice API v3

### 2.1 Base URL・認証

| 項目 | 値 |
|------|---|
| Base URL | `https://invoice.moneyforward.com/api/v3` |
| 認証 | OAuth2 (`api.biz.moneyforward.com`) |
| スコープ | `mfc/invoice/data.read mfc/invoice/data.write` |
| レート制限 | 3 req/sec |

### 2.2 エンドポイント一覧

#### 事業所

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/office` | 事業所情報取得 |

#### 取引先

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/partners` | 取引先一覧 |
| POST | `/partners` | 取引先作成 |
| GET | `/partners/{id}` | 取引先詳細 |
| PATCH | `/partners/{id}` | 取引先更新 |
| DELETE | `/partners/{id}` | 取引先削除 |
| GET | `/partners/{id}/departments` | 取引先の部署一覧 |

#### 品目

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/items` | 品目一覧 |
| POST | `/items` | 品目作成 |
| GET | `/items/{id}` | 品目詳細 |
| PATCH | `/items/{id}` | 品目更新 |
| DELETE | `/items/{id}` | 品目削除 |

#### 請求書

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/billings` | 請求書一覧 |
| POST | `/billings` | 請求書作成 (レガシー) |
| POST | `/invoice_template_billings` | 請求書作成 (インボイス制度対応) |
| GET | `/billings/{id}` | 請求書詳細 |
| PATCH | `/billings/{id}` | 請求書更新 |
| DELETE | `/billings/{id}` | 請求書削除 |
| GET | `/billings/{id}/pdf` | 請求書 PDF URL 取得 |

#### 見積書

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/quotes` | 見積書一覧 |
| POST | `/quotes` | 見積書作成 |
| GET | `/quotes/{id}` | 見積書詳細 |
| PATCH | `/quotes/{id}` | 見積書更新 |
| DELETE | `/quotes/{id}` | 見積書削除 |
| GET | `/quotes/{id}/pdf` | 見積書 PDF URL 取得 |
| POST | `/quotes/{id}/convert_to_billing` | 見積書→請求書変換 |

#### 送付履歴

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/sent_histories` | 送付履歴一覧 |

### 2.3 Go データモデル

#### Partner

```go
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

#### Item

```go
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
```

#### Billing

```go
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

type CreateBillingParams struct {
    DepartmentID     string               `json:"department_id"`
    BillingDate      string               `json:"billing_date"`
    Title            string               `json:"title,omitempty"`
    Memo             string               `json:"memo,omitempty"`
    PaymentCondition string               `json:"payment_condition,omitempty"`
    DueDate          string               `json:"due_date,omitempty"`
    SalesDate        string               `json:"sales_date,omitempty"`
    BillingNumber    string               `json:"billing_number,omitempty"`
    Note             string               `json:"note,omitempty"`
    DocumentName     string               `json:"document_name,omitempty"`
    TagNames         []string             `json:"tag_names,omitempty"`
    Items            []InvoiceTemplateLine `json:"items,omitempty"`
}

type UpdateBillingParams struct {
    DepartmentID     string        `json:"department_id,omitempty"`
    PartnerID        string        `json:"partner_id,omitempty"`
    PartnerName      string        `json:"partner_name,omitempty"`
    PartnerDetail    string        `json:"partner_detail,omitempty"`
    Title            string        `json:"title,omitempty"`
    Memo             string        `json:"memo,omitempty"`
    PaymentCondition string        `json:"payment_condition,omitempty"`
    BillingDate      string        `json:"billing_date,omitempty"`
    DueDate          string        `json:"due_date,omitempty"`
    SalesDate        string        `json:"sales_date,omitempty"`
    Items            []BillingItem `json:"items,omitempty"`
}
```

#### Quote

```go
type Quote struct {
    ID            string      `json:"id"`
    PDFURL        string      `json:"pdf_url,omitempty"`
    OperatorID    string      `json:"operator_id,omitempty"`
    DepartmentID  string      `json:"department_id,omitempty"`
    PartnerID     string      `json:"partner_id,omitempty"`
    PartnerName   string      `json:"partner_name,omitempty"`
    PartnerDetail string      `json:"partner_detail,omitempty"`
    MemberID      string      `json:"member_id,omitempty"`
    MemberName    string      `json:"member_name,omitempty"`
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
    DepartmentID string               `json:"department_id"`
    QuoteDate    string               `json:"quote_date"`
    ExpiredDate  string               `json:"expired_date"`
    Title        string               `json:"title,omitempty"`
    Memo         string               `json:"memo,omitempty"`
    Note         string               `json:"note,omitempty"`
    DocumentName string               `json:"document_name,omitempty"`
    TagNames     []string             `json:"tag_names,omitempty"`
    Items        []InvoiceTemplateLine `json:"items,omitempty"`
}

type UpdateQuoteParams struct {
    Title       string               `json:"title,omitempty"`
    Memo        string               `json:"memo,omitempty"`
    QuoteDate   string               `json:"quote_date,omitempty"`
    ExpiredDate string               `json:"expired_date,omitempty"`
    Items       []InvoiceTemplateLine `json:"items,omitempty"`
}
```

#### InvoiceTemplateLine (インボイス制度対応の明細行)

```go
type InvoiceTemplateLine struct {
    ItemID                 string    `json:"item_id,omitempty"`
    Name                   string    `json:"name,omitempty"`
    DeliveryNumber         string    `json:"delivery_number,omitempty"`
    DeliveryDate           string    `json:"delivery_date,omitempty"`
    Detail                 string    `json:"detail,omitempty"`
    Unit                   string    `json:"unit,omitempty"`
    Price                  int       `json:"price"`
    Quantity               int       `json:"quantity"`
    IsDeductWithholdingTax *bool     `json:"is_deduct_withholding_tax,omitempty"`
    Excise                 ExciseType `json:"excise"`
}
```

### 2.4 Enum 値

#### ExciseType (税区分)

```go
type ExciseType string

const (
    ExciseTenPercent                  ExciseType = "ten_percent"
    ExciseEightPercent                ExciseType = "eight_percent"
    ExciseEightPercentReducedTaxRate  ExciseType = "eight_percent_as_reduced_tax_rate"
    ExciseFivePercent                 ExciseType = "five_percent"
    ExciseUntaxable                   ExciseType = "untaxable"
    ExciseTaxExemption                ExciseType = "tax_exemption"
    ExciseNonTaxable                  ExciseType = "non_taxable"
)
```

#### PaymentStatus (入金ステータス)

```go
type PaymentStatus string

const (
    PaymentStatusUnsettled PaymentStatus = "unsettled"
    PaymentStatusSettled   PaymentStatus = "settled"
)
```

#### QuoteStatus (見積書ステータス)

```go
type QuoteStatus string

const (
    QuoteStatusDraft     QuoteStatus = "draft"
    QuoteStatusSent      QuoteStatus = "sent"
    QuoteStatusAccepted  QuoteStatus = "accepted"
    QuoteStatusRejected  QuoteStatus = "rejected"
    QuoteStatusCancelled QuoteStatus = "cancelled"
)
```

### 2.5 エンドポイント別クエリパラメータ

#### GET /partners, GET /items

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | 検索キーワード |

#### GET /billings

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | 検索キーワード |
| `partner_id` | string | 取引先 ID |
| `payment_status` | string | `unsettled` \| `settled` |
| `from` | string | 開始日 (YYYY-MM-DD) |
| `to` | string | 終了日 (YYYY-MM-DD) |

#### GET /quotes

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | 検索キーワード |
| `partner_id` | string | 取引先 ID |
| `status` | string | `draft` \| `sent` \| `accepted` \| `rejected` \| `cancelled` |
| `from` | string | 開始日 (YYYY-MM-DD) |
| `to` | string | 終了日 (YYYY-MM-DD) |

### 2.6 特記事項

#### /invoice_template_billings (インボイス制度対応)

- 新規請求書作成には `/invoice_template_billings` を使用する (レガシー `/billings` POST ではなく)
- `department_id` と `billing_date` が必須
- 明細行には `excise` (税区分) が必須
- リクエストボディにラッピングなし (直接パラメータ)

#### department_id の自動取得パターン

```
1. partner_id を取得
2. GET /partners/{partner_id}/departments で部署一覧取得
3. departments[0].id を department_id として使用
4. 部署が0件の場合はエラー
```

#### リクエストボディのラッピング規則

| エンドポイント | メソッド | ラッピング |
|--------------|---------|----------|
| `/quotes` | POST | なし (直接パラメータ) |
| `/invoice_template_billings` | POST | なし (直接パラメータ) |
| `/billings` | POST | `{ "billing": {...} }` |
| `/billings/{id}` | PATCH | `{ "billing": {...} }` |
| `/billings/{id}/items` | POST | `{ "item": {...} }` |
| `/quotes/{id}/convert_to_billing` | POST | `{}` (空オブジェクト) |

---

## 3. Expense API v1

> TBD: OpenAPI spec (`expense.moneyforward.com/api/index.json`) から取得予定

### 3.1 Base URL・認証

| 項目 | 値 |
|------|---|
| Base URL | `https://expense.moneyforward.com/api/external/v1/` (v2 for office_members) |
| 認証 | OAuth2 (`expense.moneyforward.com/oauth/`) |
| スコープ | `office_setting:write`, `transaction:write`, `report:write`, `user_setting:write`, `account:write`, `public_resource:read` |

---

## 4. Payable API v1

> TBD: Expense と共有部分多い

### 4.1 Base URL・認証

| 項目 | 値 |
|------|---|
| Base URL | `https://payable.moneyforward.com/api/external/v1/` |
| 認証 | OAuth2 (`payable.moneyforward.com/oauth/`) |
| スコープ | Expense と同一 (`mf_expense_oauth`) |

---

## 5. Payroll API v2

> TBD

### 5.1 Base URL・認証

| 項目 | 値 |
|------|---|
| Base URL | `https://payroll.moneyforward.com/api/v2/` |
| 認証 | IP ホワイトリスト + API Identifier (`?identifier=UUID`) |

---

## 6. Admina API v1

### 6.1 Base URL・認証

| 項目 | 値 |
|------|---|
| Base URL | `https://api.itmc.i.moneyforward.com/api/v1` |
| 認証 | Bearer Token (API Key) |
| ヘッダー | `Authorization: Bearer {ADMINA_API_KEY}` |
| 追加ヘッダー | `X-Request-Source: mcp` |
| 必須設定 | API Key + Organization ID |
| ページネーション | Cursor-based (`limit`, `cursor`) |

### 6.2 エンドポイント一覧

#### 組織

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/organizations/{orgId}` | 組織情報取得 |

#### 従業員 (Identity)

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/organizations/{orgId}/identity` | 従業員一覧 |
| POST | `/organizations/{orgId}/identity` | 従業員作成 |
| GET | `/organizations/{orgId}/identity/{id}` | 従業員詳細 |
| PUT | `/organizations/{orgId}/identity/{id}` | 従業員更新 |
| DELETE | `/organizations/{orgId}/identity/{id}` | 従業員削除 |
| GET | `/organizations/{orgId}/identity/stats` | 従業員統計 |
| GET | `/organizations/{orgId}/identity/check` | 管理タイプ確認 |

#### デバイス

| メソッド | パス | 説明 |
|---------|------|------|
| POST | `/organizations/{orgId}/devices/search` | デバイス検索 (POST で検索) |
| POST | `/organizations/{orgId}/devices` | デバイス作成 |
| PATCH | `/organizations/{orgId}/devices/{id}` | デバイス更新 |
| PATCH | `/organizations/{orgId}/devices/{id}/meta` | デバイスメタ更新 (ステータス・割当) |

#### SaaS サービス

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/organizations/{orgId}/services` | サービス一覧 |
| GET | `/organizations/{orgId}/services/{id}/accounts` | サービスのアカウント一覧 |
| GET | `/organizations/{orgId}/people/{id}/accounts` | 人物の SaaS アカウント一覧 |

### 6.3 Go データモデル

#### Identity (従業員)

```go
type Identity struct {
    ID              string         `json:"id"`
    FirstName       string         `json:"firstName"`
    LastName        string         `json:"lastName"`
    DisplayName     string         `json:"displayName,omitempty"`
    PrimaryEmail    string         `json:"primaryEmail,omitempty"`
    SecondaryEmails []string       `json:"secondaryEmails,omitempty"`
    EmployeeStatus  EmployeeStatus `json:"employeeStatus"`
    EmployeeType    EmployeeType   `json:"employeeType"`
    ManagementType  ManagementType `json:"managementType,omitempty"`
    CompanyName     string         `json:"companyName,omitempty"`
    WorkLocation    string         `json:"workLocation,omitempty"`
    Department      *Department    `json:"department,omitempty"`
    JobTitle        string         `json:"jobTitle,omitempty"`
    EmployeeID      string         `json:"employeeId,omitempty"`
    Lifecycle       *Lifecycle     `json:"lifecycle,omitempty"`
    Note            string         `json:"note,omitempty"`
    Manager         *Manager       `json:"manager,omitempty"`
}

type Department struct {
    Name string `json:"name,omitempty"`
}

type Lifecycle struct {
    ContractStartAt   string `json:"contractStartAt,omitempty"`
    ContractEndAt     string `json:"contractEndAt,omitempty"`
    SuspensionStartAt string `json:"suspensionStartAt,omitempty"`
    SuspensionEndAt   string `json:"suspensionEndAt,omitempty"`
}

type Manager struct {
    ID string `json:"id,omitempty"`
}
```

#### Device

```go
type Device struct {
    ID     string       `json:"id"`
    Type   DeviceType   `json:"type"`
    Status DeviceStatus `json:"status,omitempty"`
    Memo   string       `json:"memo,omitempty"`
    Preset DevicePreset `json:"preset"`
}

type DevicePreset struct {
    AssetNumber        string `json:"asset_number"`
    Subtype            string `json:"subtype"`
    ModelName          string `json:"model_name"`
    SerialNumber       string `json:"serial_number,omitempty"`
    ModelNumber        string `json:"model_number,omitempty"`
    Memory             string `json:"memory,omitempty"`
    HDDSSD             string `json:"hdd_ssd,omitempty"`
    CPU                string `json:"cpu,omitempty"`
    OS                 string `json:"os,omitempty"`
    Size               string `json:"size,omitempty"`
    Manufacturer       string `json:"manufacturer,omitempty"`
    Supplier           string `json:"supplier,omitempty"`
    ProcurementMethod  string `json:"procurement_method,omitempty"`
    PurchaseDate       string `json:"purchase_date,omitempty"`
    PurchaseCost       string `json:"purchase_cost,omitempty"`
    WarrantyPeriod     string `json:"warranty_period,omitempty"`
    DecommissionDate   string `json:"decommission_date,omitempty"`
    ScheduledReturnDate string `json:"scheduled_return_date,omitempty"`
    FixedAsset         string `json:"fixed_asset,omitempty"`
    PhoneNumber        string `json:"phone_number,omitempty"`
    SIMNumber          string `json:"sim_number,omitempty"`
    MobilePlan         string `json:"mobile_plan,omitempty"`
    Hostname           string `json:"hostname,omitempty"`
    Version            string `json:"version,omitempty"`
    KeyboardLayout     string `json:"keyboard_layout,omitempty"`
    UsageStartDate     string `json:"usage_start_date,omitempty"`
    UsageEndDate       string `json:"usage_end_date,omitempty"`
}
```

#### Service / Account

```go
type Service struct {
    ID   string `json:"id"`
    Name string `json:"name,omitempty"`
}

type Account struct {
    ID     string      `json:"id"`
    Email  string      `json:"email,omitempty"`
    Role   AccountRole `json:"role,omitempty"`
    Type   AccountType `json:"type,omitempty"`
}
```

### 6.4 Enum 値

#### EmployeeStatus

```go
type EmployeeStatus string

const (
    EmployeeStatusActive    EmployeeStatus = "active"
    EmployeeStatusOnLeave   EmployeeStatus = "on_leave"
    EmployeeStatusDraft     EmployeeStatus = "draft"
    EmployeeStatusPreactive EmployeeStatus = "preactive"
    EmployeeStatusRetired   EmployeeStatus = "retired"
    EmployeeStatusUntracked EmployeeStatus = "untracked"
    EmployeeStatusArchived  EmployeeStatus = "archived"
)
```

#### EmployeeType

```go
type EmployeeType string

const (
    EmployeeTypeBoard           EmployeeType = "board_member"
    EmployeeTypeFullTime        EmployeeType = "full_time_employee"
    EmployeeTypeFixedTime       EmployeeType = "fixed_time_employee"
    EmployeeTypeTemporary       EmployeeType = "temporary_employee"
    EmployeeTypePartTime        EmployeeType = "part_time_employee"
    EmployeeTypeSecondment      EmployeeType = "secondment_employee"
    EmployeeTypeContract        EmployeeType = "contract_employee"
    EmployeeTypeCollaborator    EmployeeType = "collaborator"
    EmployeeTypeGroupAddress    EmployeeType = "group_address"
    EmployeeTypeSharedAddress   EmployeeType = "shared_address"
    EmployeeTypeTestAddress     EmployeeType = "test_address"
    EmployeeTypeOther           EmployeeType = "other"
    EmployeeTypeUnknown         EmployeeType = "unknown"
    EmployeeTypeUnregistered    EmployeeType = "unregistered"
)
```

#### ManagementType

```go
type ManagementType string

const (
    ManagementTypeManaged      ManagementType = "managed"
    ManagementTypeExternal     ManagementType = "external"
    ManagementTypeSystem       ManagementType = "system"
    ManagementTypeUnknown      ManagementType = "unknown"
    ManagementTypeUnregistered ManagementType = "unregistered"
)
```

#### DeviceType

```go
type DeviceType string

const (
    DeviceTypePC    DeviceType = "pc"
    DeviceTypePhone DeviceType = "phone"
    DeviceTypeOther DeviceType = "other"
)
```

#### DeviceSubtype

```go
type DeviceSubtype string

const (
    DeviceSubtypeDesktopPC       DeviceSubtype = "desktop_pc"
    DeviceSubtypeLaptopPC        DeviceSubtype = "laptop_pc"
    DeviceSubtypeTabletPC        DeviceSubtype = "tablet_pc"
    DeviceSubtypePhone           DeviceSubtype = "phone"
    DeviceSubtypeMonitor         DeviceSubtype = "monitor"
    DeviceSubtypeServer          DeviceSubtype = "server"
    DeviceSubtypePeripheralDevice DeviceSubtype = "peripheral_device"
    DeviceSubtypeOther           DeviceSubtype = "other"
)
```

#### DeviceStatus

```go
type DeviceStatus string

const (
    DeviceStatusInStock        DeviceStatus = "in_stock"
    DeviceStatusPreUse         DeviceStatus = "pre_use"
    DeviceStatusActive         DeviceStatus = "active"
    DeviceStatusMissing        DeviceStatus = "missing"
    DeviceStatusMalfunction    DeviceStatus = "malfunction"
    DeviceStatusDecommissioned DeviceStatus = "decommissioned"
    DeviceStatusOnOrder        DeviceStatus = "on_order"
)
```

#### AccountRole

```go
type AccountRole string

const (
    AccountRoleAdmin AccountRole = "admin"
    AccountRoleGuest AccountRole = "guest"
    AccountRoleOther AccountRole = "other"
)
```

#### AccountType

```go
type AccountType string

const (
    AccountTypeEmployee AccountType = "employee"
    AccountTypeGuest    AccountType = "guest"
    AccountTypeSystem   AccountType = "system"
    AccountTypeUnknown  AccountType = "unknown"
)
```
