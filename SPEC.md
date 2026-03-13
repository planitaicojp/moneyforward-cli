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

---

## 7. Phase 1 実装計画: 認証基盤

Phase 0 完了 (root, version, completion, errors, output, basic config) を前提に、
全サービスコマンドの基盤となる認証システムを構築する。
conoha-cli (`~/dev/crowdy/conoha-cli`) のアーキテクチャを踏襲し、OAuth2 Authorization Code + PKCE に対応。

### 7.1 conoha-cli との差分

| 観点 | conoha-cli | mf CLI |
|------|-----------|--------|
| 認証方式 | パスワード → Keystone トークン | OAuth2 Authorization Code + PKCE |
| トークン更新 | 保存パスワードで再認証 | refresh_token grant |
| credentials.yaml | パスワード保存 | client_secret 保存 |
| tokens.yaml | プロファイルごとに1トークン | プロファイル×サービスで複数トークン |
| ログインフロー | 非対話 (パスワード入力) | ブラウザ起動 + ローカルコールバック |
| マルチサービス | リージョンごとに1エンドポイント | サービスごとに異なる OAuth エンドポイント |
| PKCE | 不要 | 必須 (code_verifier/code_challenge S256) |

### 7.2 ファイル一覧

#### 新規作成 (10 ファイル)

| ファイル | conoha 移植度 | 責務 |
|---------|-------------|------|
| `internal/config/credentials.go` | 95% | client_secret の保存・読込 (credentials.yaml, 0600) |
| `internal/config/tokens.go` | 60% | サービス別トークン管理 (tokens.yaml, 0600) |
| `internal/api/oauth.go` | **新規** | PKCE生成, 認可URL構築, コールバックサーバー, トークン交換・リフレッシュ |
| `internal/api/client.go` | 80% | HTTP クライアント (Bearer auth, リトライ, デバッグ) |
| `internal/api/debug.go` | 90% | リクエスト/レスポンスのデバッグログ (MF_DEBUG) |
| `internal/prompt/prompt.go` | 95% | 対話入力 (String, Password, Confirm) |
| `internal/prompt/select.go` | 95% | 選択プロンプト (promptui) |
| `cmd/cmdutil/client.go` | 70% | NewClient(cmd, service) ファクトリ |
| `cmd/auth/auth.go` | 40% | auth login\|logout\|status\|list\|switch\|remove\|token |
| `cmd/config/config.go` | 80% | config get\|set\|list\|path |

#### 修正 (2 ファイル)

| ファイル | 変更内容 |
|---------|---------|
| `internal/config/config.go` | 環境変数定数追加 (MF_CLIENT_ID, MF_CLIENT_SECRET, MF_ACCESS_TOKEN 等) |
| `cmd/root.go` | auth, config サブコマンド登録 |

### 7.3 tokens.yaml 構造設計

サービスごとにトークンを分離管理する。Invoice 認証後に Expense を使う際、再ログイン不要。

```yaml
profiles:
  default:
    services:
      invoice:
        access_token: "eyJ..."
        refresh_token: "abc..."
        expires_at: "2026-03-14T00:00:00Z"
        scope: "mfc/invoice/data.read mfc/invoice/data.write"
      expense:
        access_token: "eyJ..."
        refresh_token: "def..."
        expires_at: "2026-03-14T00:00:00Z"
        scope: "office_setting:write transaction:write report:write user_setting:write account:write public_resource:read"
```

**Go struct**:

```go
type TokenStore struct {
    Profiles map[string]ProfileTokens `yaml:"profiles"`
}

type ProfileTokens struct {
    Services map[string]TokenEntry `yaml:"services,omitempty"`
}

type TokenEntry struct {
    AccessToken  string    `yaml:"access_token"`
    RefreshToken string    `yaml:"refresh_token"`
    ExpiresAt    time.Time `yaml:"expires_at"`
    Scope        string    `yaml:"scope"`
}
```

**メソッド**: `Get(profile, service)`, `Set(profile, service, entry)`, `Delete(profile)`, `DeleteService(profile, service)`, `IsValid(profile, service)` (5分バッファ), `ListServices(profile)`, `Load()`, `Save()`

### 7.4 credentials.yaml 構造設計

```yaml
profiles:
  default:
    client_secret: "your-client-secret"
```

**Go struct**:

```go
type CredentialsStore struct {
    Profiles map[string]Credentials `yaml:"profiles"`
}

type Credentials struct {
    ClientSecret string `yaml:"client_secret"`
}
```

**メソッド**: `Get(profile)`, `Set(profile, creds)`, `Delete(profile)`, `Load()`, `Save()` — conoha-cli の `credentials.go` をほぼそのまま移植。Password → ClientSecret に変更のみ。

### 7.5 OAuth2 ログインフロー詳細

```
mf auth login --service invoice
  │
  ├─ 1. config.yaml から profile の client_id 取得 (なければプロンプト)
  ├─ 2. credentials.yaml から client_secret 取得 (なければプロンプト)
  ├─ 3. サービス決定 (--service フラグ or 対話選択)
  ├─ 4. スコープ決定 (--scopes フラグ or サービスデフォルト)
  │
  ├─ 5. PKCE 生成
  │     code_verifier: 43-128文字のランダム文字列 (crypto/rand)
  │     code_challenge: SHA256(code_verifier) → base64url エンコード
  │
  ├─ 6. localhost:38080 でコールバックサーバー起動
  │     GET /callback?code=XXX → codeCh に送信, HTML 成功ページ表示
  │     GET /callback?error=YYY → errCh に送信, HTML エラーページ表示
  │     5分タイムアウト
  │
  ├─ 7. 認可 URL をブラウザで開く
  │     WSL2: cmd.exe /c start <url>
  │     Linux: xdg-open <url>
  │     macOS: open <url>
  │     失敗時: URL を stderr に表示し手動コピーを促す
  │
  ├─ 8. ユーザーが MF Cloud で承認 → /callback?code=XXX 受信
  │
  ├─ 9. トークン交換
  │     POST {tokenURL}
  │     Content-Type: application/x-www-form-urlencoded
  │     Body: grant_type=authorization_code
  │           &code={code}
  │           &client_id={client_id}
  │           &client_secret={client_secret}     ← client_secret_post 方式
  │           &redirect_uri=http://localhost:38080/callback
  │           &code_verifier={code_verifier}     ← PKCE
  │
  ├─ 10. レスポンスから TokenEntry 構築
  │      ExpiresAt = time.Now().Add(time.Duration(expires_in) * time.Second)
  │
  └─ 11. 永続化
        config.yaml: プロファイル (client_id, scopes)
        credentials.yaml: client_secret
        tokens.yaml: サービス別トークン
        stdout: "✓ Logged in as {profile} for {service} (expires in 1 hour)"
```

### 7.6 トークンリフレッシュフロー

```
cmdutil.NewClient(cmd, "invoice")
  → EnsureToken(profile, "invoice", cfg, creds, tokens)
    │
    ├─ 1. MF_ACCESS_TOKEN 環境変数 → そのまま返す (キャッシュスキップ)
    │
    ├─ 2. tokens.IsValid(profile, "invoice")
    │     = expires_at - 5分 > now
    │     → true: キャッシュされた access_token を返す
    │
    ├─ 3. entry.RefreshToken が存在
    │     → POST {tokenURL}
    │       grant_type=refresh_token
    │       &refresh_token={refresh_token}
    │       &client_id={client_id}
    │       &client_secret={client_secret}
    │     → 成功: tokens.yaml 更新, 新 access_token 返す
    │     → 失敗 (refresh_token 期限切れ等):
    │       AuthError("session expired, run 'mf auth login --service invoice'")
    │
    └─ 4. トークンなし
          → AuthError("not authenticated, run 'mf auth login --service invoice'")
```

### 7.7 サービス別 OAuth エンドポイントマップ

```go
type ServiceConfig struct {
    AuthURL       string
    TokenURL      string
    BaseURL       string
    DefaultScopes []string
}

var Services = map[string]ServiceConfig{
    "invoice": {
        AuthURL:       "https://api.biz.moneyforward.com/authorize",
        TokenURL:      "https://api.biz.moneyforward.com/token",
        BaseURL:       "https://invoice.moneyforward.com/api/v3",
        DefaultScopes: []string{"mfc/invoice/data.read", "mfc/invoice/data.write"},
    },
    "expense": {
        AuthURL:       "https://expense.moneyforward.com/oauth/authorize",
        TokenURL:      "https://expense.moneyforward.com/oauth/token",
        BaseURL:       "https://expense.moneyforward.com/api/external/v1",
        DefaultScopes: []string{"office_setting:write", "transaction:write", "report:write", "user_setting:write", "account:write", "public_resource:read"},
    },
    "payable": {
        AuthURL:       "https://payable.moneyforward.com/oauth/authorize",
        TokenURL:      "https://payable.moneyforward.com/oauth/token",
        BaseURL:       "https://payable.moneyforward.com/api/external/v1",
        DefaultScopes: []string{"office_setting:write", "transaction:write", "report:write", "user_setting:write", "account:write", "public_resource:read"},
    },
}
```

### 7.8 HTTP クライアント設計

```go
type Client struct {
    HTTP      *http.Client  // Timeout: 30s
    Token     string        // Bearer token
    BaseURL   string        // サービス別 Base URL
    UserAgent string        // "mf-cli/{version}"
    Debug     bool          // MF_DEBUG 環境変数
}

// メソッド
func (c *Client) Get(path string, params url.Values) (*http.Response, error)
func (c *Client) Post(path string, body any) (*http.Response, error)
func (c *Client) Patch(path string, body any) (*http.Response, error)
func (c *Client) Delete(path string) (*http.Response, error)
func (c *Client) Do(req *http.Request) (*http.Response, error)  // リトライ: 429/5xx, 最大3回, exponential backoff
```

conoha-cli の `client.go` から移植。変更点:
- `X-Auth-Token` → `Authorization: Bearer {token}`
- `TenantID` / `Region` 削除
- `BaseURL` はサービスごとに決定

### 7.9 cmdutil.NewClient 設計

```go
func NewClient(cmd *cobra.Command, service string) (*api.Client, error) {
    // 1. MF_ACCESS_TOKEN 環境変数があればショートカット
    if token := os.Getenv("MF_ACCESS_TOKEN"); token != "" {
        svcCfg := api.Services[service]
        return api.NewClient(token, svcCfg.BaseURL), nil
    }

    // 2. プロファイル解決: --profile フラグ > MF_PROFILE > config.active_profile
    profile := resolveProfile(cmd)

    // 3. 設定ファイル読込
    cfg, _ := config.Load()
    creds, _ := config.LoadCredentials()
    tokens, _ := config.LoadTokens()

    // 4. トークン確保 (自動リフレッシュ含む)
    token, err := api.EnsureToken(profile, service, cfg, creds, tokens)
    if err != nil {
        return nil, err // AuthError → 終了コード 2
    }

    // 5. クライアント構築
    svcCfg := api.Services[service]
    return api.NewClient(token, svcCfg.BaseURL), nil
}
```

### 7.10 ブラウザ起動 (WSL2 対応)

```go
func openBrowser(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "linux":
        // WSL2 判定: /proc/version に "microsoft" が含まれるか
        if isWSL() {
            cmd = exec.Command("cmd.exe", "/c", "start", url)
        } else {
            cmd = exec.Command("xdg-open", url)
        }
    case "darwin":
        cmd = exec.Command("open", url)
    case "windows":
        cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
    }
    return cmd.Start()
}

func isWSL() bool {
    data, err := os.ReadFile("/proc/version")
    if err != nil {
        return false
    }
    return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
```

### 7.11 環境変数一覧

| 環境変数 | 説明 | 使用箇所 |
|---------|------|---------|
| `MF_PROFILE` | アクティブプロファイル | cmdutil.NewClient |
| `MF_CLIENT_ID` | OAuth2 Client ID | auth login (プロンプトスキップ) |
| `MF_CLIENT_SECRET` | OAuth2 Client Secret | auth login, EnsureToken |
| `MF_ACCESS_TOKEN` | アクセストークン直接指定 | cmdutil.NewClient (全キャッシュスキップ) |
| `MF_FORMAT` | 出力形式 | root.go (既存) |
| `MF_CONFIG_DIR` | 設定ディレクトリ | config.Load |
| `MF_NO_INPUT` | 非対話モード | auth login (プロンプト無効化) |
| `MF_DEBUG` | デバッグ出力 | api/debug.go |
| `MF_CALLBACK_PORT` | コールバックポート | auth login (default: 38080) |

### 7.12 コミット計画

#### Commit 1: Config layer

```
internal/config/credentials.go   — CredentialsStore (Get/Set/Delete/Load/Save)
internal/config/tokens.go        — TokenStore (Get/Set/Delete/IsValid/Load/Save, multi-service)
internal/config/config.go        — 環境変数定数追加
go.mod                           — golang.org/x/term, github.com/manifoldco/promptui 追加
```

#### Commit 2: API layer + prompt

```
internal/api/debug.go            — デバッグログ (MF_DEBUG, sensitive masking)
internal/api/client.go           — HTTP クライアント (Bearer, リトライ 429/5xx, User-Agent)
internal/api/oauth.go            — PKCE, ServiceConfig, コールバックサーバー,
                                   ExchangeCode, RefreshToken, EnsureToken
internal/prompt/prompt.go        — String/Password/Confirm
internal/prompt/select.go        — Select (promptui)
cmd/cmdutil/client.go            — NewClient(cmd, service)
```

#### Commit 3: Commands

```
cmd/auth/auth.go                 — login, logout, status, list, switch, remove, token
cmd/config/config.go             — get, set, list, path
cmd/root.go                      — auth, config サブコマンド登録
```

#### Commit 4: Tests

```
internal/config/tokens_test.go      — Set/Get/IsValid/multi-service/round-trip
internal/config/credentials_test.go — Set/Get/Delete
internal/api/oauth_test.go          — PKCE 生成, コールバックサーバー (httptest),
                                      トークン交換 mock (httptest.NewServer)
internal/api/client_test.go         — Bearer ヘッダー, リトライ, エラーパース
```

### 7.13 依存追加

```
golang.org/x/term                # Password 入力 (noecho)
github.com/manifoldco/promptui   # 対話 Select プロンプト
```

### 7.14 注意事項

- **WSL2**: `cmd.exe /c start` フォールバック必要。`isWSL()` で `/proc/version` を確認
- **コールバックポート**: デフォルト 38080, `--port` / `MF_CALLBACK_PORT` で変更可能
- **`--no-input` モード**: `--service` フラグ必須。プロンプト全無効化
- **`office_id`**: ログイン時不要。後で `mf config set profiles.default.office_id <id>` で設定
- **並行リフレッシュ**: v1 は last-writer-wins で許容 (CLI の一般的な使用パターンでは問題なし)
- **リフレッシュ失敗**: refresh_token 期限切れ (540日後) 時は `mf auth login` の再実行を促すエラーメッセージ

### 7.15 検証方法

```bash
# ビルド・静的解析
make build && make lint && make test

# 認証フロー (実際の MF Cloud アカウント必要)
mf auth login --service invoice        # ブラウザ起動, コールバック受信
mf auth status                         # トークン状態表示
mf auth token --service invoice        # access_token 出力
mf auth list                           # プロファイル一覧

# 設定管理
mf config set format json
mf config get format                   # → "json"
mf config list                         # 全設定表示
mf config path                         # → ~/.config/mf/

# トークンリフレッシュ (access_token 期限切れ後)
# → 自動リフレッシュされ、新トークンで API コール成功
```
