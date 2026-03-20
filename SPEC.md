# Money Forward API Specification

> **この文書の位置づけ**: 実装者が「このファイルだけで」APIを呼び出せる状態を目指す。
> コマンド設計・フェーズ計画は `DESIGN.md` へ、プロジェクト概要は `CLAUDE.md` へ。

---

## 目次

- [0. 凡例と読み方](#0-凡例と読み方)
- [1. 共通仕様](#1-共通仕様)
  - [1.1 認証方式マトリクス](#11-認証方式マトリクス)
  - [1.2 OAuth2 PKCE フロー](#12-oauth2-pkce-フロー)
  - [1.3 HTTP リクエスト規約](#13-http-リクエスト規約)
  - [1.4 ページネーション](#14-ページネーション)
  - [1.5 レート制限](#15-レート制限)
  - [1.6 エラーレスポンス](#16-エラーレスポンス)
  - [1.7 日付・型規約](#17-日付型規約)
  - [1.8 環境変数](#18-環境変数)
  - [1.9 実装上の共通 Gotchas](#19-実装上の共通-gotchas)
- [2. Invoice API v3](#2-invoice-api-v3)
- [3. Expense API v1](#3-expense-api-v1)
- [4. Payable API v1](#4-payable-api-v1)
- [5. Payroll API v2](#5-payroll-api-v2)
- [6. Admina API v1](#6-admina-api-v1)
- [7. App Portal 設定ガイド](#7-app-portal-設定ガイド)

---

## 0. 凡例と読み方

| マーク | 意味 |
|-------|------|
| *(確定)* | ref リポジトリや公式ドキュメントで確認済み |
| *(要確認)* | 公式仕様から推定。実装前に OpenAPI spec または実際の API レスポンスで検証すること |
| ⚠️ | 実装上の落とし穴。経験的に問題になりやすい箇所 |
| ❌ | 使ってはいけない方法・廃止されたエンドポイント |

**各サービスセクションのテンプレート**: Base URL → エンドポイント一覧 → データモデル → Enum → クエリパラメータ → Gotchas

---

## 1. 共通仕様

### 1.1 認証方式マトリクス

| サービス | 認証方式 | OAuth エンドポイント | 備考 |
|---------|---------|---------------------|------|
| Invoice | OAuth2 + PKCE | `api.biz.moneyforward.com` | *(確定)* |
| Expense | OAuth2 + PKCE | `expense.moneyforward.com/oauth/` | *(確定)* |
| Payable | OAuth2 + PKCE | `payable.moneyforward.com/oauth/` | *(確定)* |
| Payroll | IP ホワイトリスト + API Identifier | N/A | *(確定)* クエリパラメータ方式 |
| Admina | API Key (Bearer) | N/A | *(確定)* Organization ID 必須 |

### 1.2 OAuth2 PKCE フロー

#### STEP 1: Authorization URL 構築

```
GET {AuthURL}
  ?response_type=code
  &client_id={CLIENT_ID}
  &redirect_uri=http://localhost:{PORT}/callback
  &scope={SCOPE}               ← space-separated (OAuth2 標準)
  &state={RANDOM_STATE}        ← CSRF 防止用 (crypto/rand, 16バイト以上)
  &code_challenge={CHALLENGE}
  &code_challenge_method=S256
```

**PKCE 生成規則** *(確定, RFC 7636)*:

```
code_verifier  : crypto/rand で 32バイト生成 → base64url (no padding)
                 結果は 43文字 (32バイト → base64url = ceil(32*4/3))
                 ※ RFC は 43-128文字を許容するが、32バイト固定で十分
code_challenge : SHA256(code_verifier バイト列) → base64url (no padding)
                 S256 なので常に 43文字
```

#### STEP 2: コールバック受信

- ローカル HTTP サーバーを `localhost:{PORT}/callback` で起動
- **ポート決定**: `MF_CALLBACK_PORT` 環境変数 → ランダム空きポート (`net.Listen("tcp", ":0")`)
  - ⚠️ 固定ポート (38080 等) は避ける。CI 環境や複数セッションで衝突する
- **state 検証**: callback の `?state=` が STEP1 で生成した値と一致するか検証。不一致は即エラー
- タイムアウト: 5分

#### STEP 3: Token Exchange (POST)

```
POST {TokenURL}
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code={AUTHORIZATION_CODE}
&client_id={CLIENT_ID}
&client_secret={CLIENT_SECRET}      ← client_secret_post 方式
&redirect_uri=http://localhost:{PORT}/callback
&code_verifier={CODE_VERIFIER}      ← PKCE
```

> ⚠️ `client_secret_basic` (Authorization ヘッダー) ではなく `client_secret_post` (ボディ) を使う。*(確定)*

#### STEP 4: Token Refresh (POST)

```
POST {TokenURL}
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={REFRESH_TOKEN}
&client_id={CLIENT_ID}
&client_secret={CLIENT_SECRET}
```

#### Token Response

```json
{
  "access_token":  "eyJ...",
  "refresh_token": "abc...",
  "token_type":    "Bearer",
  "expires_in":    3600,
  "scope":         "mfc/invoice/data.read mfc/invoice/data.write"
}
```

**トークン有効期限** *(確定)*:

| トークン | 有効期限 | 自動リフレッシュトリガー |
|---------|---------|----------------------|
| access_token | 3600秒 (1時間) | `now > expires_at - 5分` |
| refresh_token | 540日 | 期限切れ → 再ログイン |

**Go canonical 型**:

```go
type TokenEntry struct {
    AccessToken  string    `yaml:"access_token"`
    RefreshToken string    `yaml:"refresh_token"`
    ExpiresAt    time.Time `yaml:"expires_at"`   // ← time.Time が正。expires_in からの変換: time.Now().Add(time.Duration(expiresIn) * time.Second)
    Scope        string    `yaml:"scope"`
}
```

> ⚠️ API レスポンスの `expires_in` (int 秒数) を `ExpiresAt` (time.Time) に変換する責務は **クライアント側**。

#### サービス別 OAuth エンドポイント

```go
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
        DefaultScopes: []string{"office_setting:write", "transaction:write", "report:write",
                                "user_setting:write", "account:write", "public_resource:read"},
    },
    "payable": {
        AuthURL:       "https://payable.moneyforward.com/oauth/authorize",
        TokenURL:      "https://payable.moneyforward.com/oauth/token",
        BaseURL:       "https://payable.moneyforward.com/api/external/v1",
        DefaultScopes: []string{"office_setting:write", "transaction:write", "report:write",
                                "user_setting:write", "account:write", "public_resource:read"},
    },
}
```

#### スコープ形式変換

CLI フラグは comma-separated (`--scopes "read,write"`) だが、OAuth2 リクエストには **space-separated** で送る。

```go
// CLI → OAuth2
scope := strings.Join(strings.Split(flagScopes, ","), " ")
```

### 1.3 HTTP リクエスト規約

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json        (ボディあり POST/PATCH のみ)
User-Agent: mf-cli/{version}
```

### 1.4 ページネーション

#### Page-based (Invoice / Expense / Payable)

**リクエストパラメータ**:

| パラメータ | 型 | デフォルト | 最大値 |
|-----------|---|----------|-------|
| `page` | int | 1 | — |
| `per_page` | int | 25 | Invoice: 100 *(要確認)*, Expense: 100 *(要確認)* |

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

**`--all` 実装パターン** (全件取得):

```go
for page := 1; ; page++ {
    resp := fetchPage(page, perPage)
    results = append(results, resp.Data...)
    if resp.Pagination.CurrentPage >= resp.Pagination.TotalPages {
        break
    }
    // Invoice: 3 req/sec 制限に注意 → time.Sleep(400 * time.Millisecond)
}
```

#### Cursor-based (Admina)

**リクエストパラメータ**:

| パラメータ | 型 | 最大値 | 説明 |
|-----------|---|-------|------|
| `limit` | int | 200 | 1回あたりの取得件数 |
| `cursor` | string | — | 前回レスポンスの `nextCursor` |

**レスポンス**: *(要確認 — 実際のフィールド名は OpenAPI spec で確認)*

```json
{
  "results": [...],
  "nextCursor": "base64encodedCursor=="
}
```

> ⚠️ `nextCursor` が `null` または空文字列の場合が最終ページ。

### 1.5 レート制限

| サービス | 制限 | 超過時のステータス | 対処 |
|---------|------|-----------------|------|
| Invoice | **3 req/sec** *(確定)* | 429 | `Retry-After` ヘッダーの秒数だけ待機してリトライ |
| Expense | 不明 *(要確認)* | — | 429 受信時は exponential backoff |
| Payable | 不明 *(要確認)* | — | 同上 |
| Payroll | 不明 *(要確認)* | — | 同上 |
| Admina | 不明 *(要確認)* | — | 同上 |

**リトライポリシー** (全サービス共通):

```
429: Retry-After ヘッダーがあればその秒数待機、なければ 1秒
5xx: exponential backoff (1s, 2s, 4s), 最大 3回
それ以外: リトライしない
```

### 1.6 エラーレスポンス

#### Invoice / Expense / Payable

```json
{
  "code":    "validation_error",
  "message": "入力値が不正です",
  "errors":  { "billing_date": ["必須項目です"] }
}
```

```go
type APIErrorBody struct {
    Code    string              `json:"code"`
    Message string              `json:"message"`
    Errors  map[string][]string `json:"errors,omitempty"`
}
```

**HTTP Status → Go エラー型**:

| HTTP Status | Go エラー型 | 終了コード |
|------------|------------|----------|
| 401 | `AuthError` | 2 |
| 403 | `AuthError` | 2 |
| 404 | `NotFoundError` | 3 |
| 422 | `ValidationError` | 4 |
| 429 | *(リトライ後失敗で)* `APIError` | 5 |
| 5xx | `APIError` | 5 |

> ⚠️ 401 は「トークン期限切れ」と「スコープ不足」で発生する。refresh_token でリフレッシュ後に同じ 401 が返る場合は **スコープ不足**なので再ログインを促す。

#### Admina

```json
{
  "errorId":      "INVALID_REQUEST",
  "errorDetails": {},
  "status":       400
}
```

| HTTP Status | 意味 |
|------------|------|
| 400 | InvalidRequest |
| 401 | Unauthenticated |
| 403 | PermissionDenied |
| 404 | NotFound |
| 408 | RequestTimeout |
| 422 | ValidationError |
| 500 | SystemError |
| 504 | GatewayTimeout |

### 1.7 日付・型規約

| 用途 | フォーマット | 例 |
|-----|------------|---|
| 日付フィールド (billing_date, due_date 等) | `YYYY-MM-DD` | `"2026-04-01"` |
| タイムスタンプ (created_at, updated_at 等) | RFC3339 | `"2026-04-01T12:00:00.000Z"` |
| Go 型 | `string` (変換コストなし。表示時に必要に応じてパース) | — |
| タイムゾーン | UTC (API は常に UTC で返す) | — |

> ⚠️ 請求書の `billing_date` に時刻は不要。`"2026-04-01T00:00:00Z"` を渡すと API 側でエラーになる場合がある。`YYYY-MM-DD` のみ使うこと。

### 1.8 環境変数

| 環境変数 | 説明 | 優先度 |
|---------|------|-------|
| `MF_PROFILE` | アクティブプロファイル名 | config.yaml の `active_profile` より優先 |
| `MF_CLIENT_ID` | OAuth2 Client ID | プロファイル設定より優先 (auth login 時) |
| `MF_CLIENT_SECRET` | OAuth2 Client Secret | credentials.yaml より優先 |
| `MF_ACCESS_TOKEN` | アクセストークン直接指定 | 設定・キャッシュを完全スキップ |
| `MF_FORMAT` | 出力形式 (`table`/`json`/`yaml`/`csv`) | `--format` フラグより低優先 |
| `MF_CONFIG_DIR` | 設定ディレクトリ | デフォルト: `~/.config/mf/` |
| `MF_NO_INPUT` | 非対話モード (`1` で有効) | `--no-input` フラグと同等 |
| `MF_DEBUG` | デバッグ出力レベル (`1`=headers, `2`=body) | — |
| `MF_CALLBACK_PORT` | OAuth コールバックポート | 未設定時はランダム空きポート |

> **CI/CD での使い方**: `MF_ACCESS_TOKEN` を設定すれば OAuth フロー不要。GitHub Actions の secrets に設定する想定。

### 1.9 実装上の共通 Gotchas

#### ① concurrent token refresh (race condition)

複数のサブコマンドが並走しない CLI では実用上問題ないが、将来の並行実行に備えて記録:
- 複数 goroutine が同時に期限切れトークンを検出した場合、両方がリフレッシュリクエストを送る
- MF は同じ refresh_token を2回使えるかどうか保証していない *(要確認)*
- **現在の対処**: last-writer-wins を許容 (CLI の一般的な使用パターンで問題なし)
- **将来の対処**: ファイルロック (`flock`) または sync.Mutex でシリアライズ

#### ② refresh_token 期限切れ後の 401

通常の 401 (access_token 期限切れ) と同じステータスコードが返る。
区別方法: refresh_token リクエスト自体が 401/400 を返した場合 → 再ログインが必要。
エラーメッセージ例: `"session expired, run 'mf auth login --service invoice'"`

#### ③ WSL2 でのブラウザ起動

```go
func isWSL() bool {
    data, _ := os.ReadFile("/proc/version")
    return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func openBrowser(rawURL string) error {
    switch {
    case runtime.GOOS == "darwin":
        return exec.Command("open", rawURL).Start()
    case runtime.GOOS == "linux" && isWSL():
        return exec.Command("cmd.exe", "/c", "start", rawURL).Start()
    case runtime.GOOS == "linux":
        return exec.Command("xdg-open", rawURL).Start()
    default:
        return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
    }
}
```

ブラウザ起動失敗時は URL を stderr に出力して手動コピーを促すこと。

#### ④ `office_id` の扱い

`office_id` は `config.yaml` に保存するが、**OAuth2 認証には不要**。
Invoice API の `/office` エンドポイントで取得できる。
使用箇所: Expense/Payable API でオフィス固有のリソースを扱う場合 *(要確認)*。

---

## 2. Invoice API v3

### 2.1 Meta

| 項目 | 値 |
|------|---|
| Base URL | `https://invoice.moneyforward.com/api/v3` |
| OpenAPI Spec | `https://invoice.moneyforward.com/docs/api/v3/index.html` |
| バージョン | v3.1.0 |
| 認証 | OAuth2 (`api.biz.moneyforward.com`) |
| スコープ | `mfc/invoice/data.read mfc/invoice/data.write` |
| レート制限 | **3 req/sec** |
| ページネーション | page-based |

### 2.2 エンドポイント一覧

#### 事業所

| Method | Path | 説明 |
|--------|------|------|
| GET | `/office` | 事業所情報取得 |

#### 取引先 (Partners)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/partners` | 一覧 |
| POST | `/partners` | 作成 |
| GET | `/partners/{id}` | 詳細 |
| PATCH | `/partners/{id}` | 更新 |
| DELETE | `/partners/{id}` | 削除 |
| GET | `/partners/{id}/departments` | 部署一覧 |

#### 品目 (Items)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/items` | 一覧 |
| POST | `/items` | 作成 |
| GET | `/items/{id}` | 詳細 |
| PATCH | `/items/{id}` | 更新 |
| DELETE | `/items/{id}` | 削除 |

#### 請求書 (Billings)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/billings` | 一覧 |
| ❌ POST | `/billings` | ❌ レガシー作成 (使用禁止) |
| **POST** | `/invoice_template_billings` | **作成 (インボイス制度対応)** |
| GET | `/billings/{id}` | 詳細 |
| PATCH | `/billings/{id}` | 更新 |
| DELETE | `/billings/{id}` | 削除 |
| GET | `/billings/{id}/pdf` | PDF URL 取得 |

#### 見積書 (Quotes)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/quotes` | 一覧 |
| POST | `/quotes` | 作成 |
| GET | `/quotes/{id}` | 詳細 |
| PATCH | `/quotes/{id}` | 更新 |
| DELETE | `/quotes/{id}` | 削除 |
| GET | `/quotes/{id}/pdf` | PDF URL 取得 |
| POST | `/quotes/{id}/convert_to_billing` | 請求書変換 |

#### 送付履歴

| Method | Path | 説明 |
|--------|------|------|
| GET | `/sent_histories` | 一覧 |

### 2.3 リクエストボディ ラッピング規則

⚠️ エンドポイントによってラッピング有無が異なる。必ず確認すること。

| Method | Path | ラッピング | 例 |
|--------|------|----------|---|
| POST | `/invoice_template_billings` | **なし** (直接パラメータ) | `{"billing_date": "...", "items": [...]}` |
| POST | `/quotes` | **なし** | `{"quote_date": "...", "items": [...]}` |
| POST | `/billings` *(❌ 非推奨)* | `{ "billing": {...} }` | — |
| PATCH | `/billings/{id}` | `{ "billing": {...} }` | — |
| POST | `/quotes/{id}/convert_to_billing` | `{}` (空オブジェクト) | — |
| PATCH | `/quotes/{id}` | *(要確認)* | — |
| POST/PATCH | `/partners`, `/items` | *(要確認)* | — |

### 2.4 Go データモデル

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
```

#### BillingItem と InvoiceTemplateLine の違い

⚠️ この2つは似ているが別物。混同しないこと。

| 用途 | 型 | 使用箇所 |
|------|---|---------|
| API レスポンス内の明細行 | `BillingItem` | `Billing.Items`, `Quote.Items` |
| 作成・更新リクエストの明細行 | `InvoiceTemplateLine` | `CreateBillingParams.Items`, `CreateQuoteParams.Items` |
| レガシー更新 | `BillingItem` | `UpdateBillingParams.Items` *(非推奨パスのみ)* |

```go
// API レスポンス内の明細行 (読み取り用)
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

// 作成・更新リクエストの明細行 (インボイス制度対応)
type InvoiceTemplateLine struct {
    ItemID                 string     `json:"item_id,omitempty"`   // 品目マスタ参照
    Name                   string     `json:"name,omitempty"`
    DeliveryNumber         string     `json:"delivery_number,omitempty"`
    DeliveryDate           string     `json:"delivery_date,omitempty"`
    Detail                 string     `json:"detail,omitempty"`
    Unit                   string     `json:"unit,omitempty"`
    Price                  int        `json:"price"`
    Quantity               int        `json:"quantity"`
    IsDeductWithholdingTax *bool      `json:"is_deduct_withholding_tax,omitempty"`
    Excise                 ExciseType `json:"excise"` // 必須
}

// 請求書作成パラメータ (POST /invoice_template_billings)
type CreateBillingParams struct {
    DepartmentID     string                `json:"department_id"`   // 必須
    BillingDate      string                `json:"billing_date"`    // 必須 (YYYY-MM-DD)
    Title            string                `json:"title,omitempty"`
    Memo             string                `json:"memo,omitempty"`
    PaymentCondition string                `json:"payment_condition,omitempty"`
    DueDate          string                `json:"due_date,omitempty"`
    SalesDate        string                `json:"sales_date,omitempty"`
    BillingNumber    string                `json:"billing_number,omitempty"`
    Note             string                `json:"note,omitempty"`
    DocumentName     string                `json:"document_name,omitempty"`
    TagNames         []string              `json:"tag_names,omitempty"`
    Items            []InvoiceTemplateLine `json:"items,omitempty"`
}

// 見積書作成パラメータ (POST /quotes)
type CreateQuoteParams struct {
    DepartmentID string                `json:"department_id"`  // 必須
    QuoteDate    string                `json:"quote_date"`     // 必須
    ExpiredDate  string                `json:"expired_date"`   // 必須
    Title        string                `json:"title,omitempty"`
    Memo         string                `json:"memo,omitempty"`
    Note         string                `json:"note,omitempty"`
    DocumentName string                `json:"document_name,omitempty"`
    TagNames     []string              `json:"tag_names,omitempty"`
    Items        []InvoiceTemplateLine `json:"items,omitempty"`
}
```

### 2.5 Enum 値

#### ExciseType (税区分) — 日本の消費税区分

⚠️ これは経理上重要な分類。判断に迷う場合は経理担当に確認すること。

```go
type ExciseType string

const (
    // 課税取引
    ExciseTenPercent                 ExciseType = "ten_percent"                    // 10%: 2019年10月〜の標準税率
    ExciseEightPercent               ExciseType = "eight_percent"                  // 8%: 2014〜2019年の標準税率 (レガシー請求書用)
    ExciseEightPercentReducedTaxRate ExciseType = "eight_percent_as_reduced_tax_rate" // 軽減税率8%: 飲食料品・新聞 (2019年10月〜の複数税率対応)
    ExciseFivePercent                ExciseType = "five_percent"                   // 5%: 〜2014年の税率 (歴史的請求書用)

    // 非課税・不課税・免税 (日本の消費税法上の3分類)
    ExciseUntaxable    ExciseType = "untaxable"     // 不課税: 消費税の課税対象外 (給与、寄付金、補助金、損害賠償等)
    ExciseTaxExemption ExciseType = "tax_exemption" // 免税: 輸出取引・国際サービス等 (消費税ゼロ、仕入税額控除は適用可)
    ExciseNonTaxable   ExciseType = "non_taxable"   // 非課税: 法定非課税 (土地、住宅家賃、医療、学校教育、金融等)
)
```

> **untaxable vs non_taxable vs tax_exemption の違い**:
> - `non_taxable` (非課税): 消費税法が課税しないと定めた特定取引 (仕入税額控除不可)
> - `untaxable` (不課税): そもそも消費税の対象となる「資産の譲渡等」に該当しない取引
> - `tax_exemption` (免税): 消費税がかかるが税率0%。輸出業者が使う。仕入税額控除は適用可

**CLI 短縮エイリアス**:

| 短縮入力 | 正式値 |
|---------|-------|
| `10` | `ten_percent` |
| `8` | `eight_percent` |
| `8r` | `eight_percent_as_reduced_tax_rate` |
| `5` | `five_percent` |
| `0` | `untaxable` |
| `exempt` | `tax_exemption` |
| `non` | `non_taxable` |

#### PaymentStatus (入金ステータス)

```go
type PaymentStatus string

const (
    PaymentStatusUnsettled PaymentStatus = "unsettled" // 未入金
    PaymentStatusSettled   PaymentStatus = "settled"   // 入金済み
)
```

> ステータス変更: `PATCH /billings/{id}` に `payment_status` フィールドを含めるか、専用エンドポイントを使う *(要確認)*。

#### QuoteStatus (見積書ステータス) — 状態遷移

```go
type QuoteStatus string

const (
    QuoteStatusDraft     QuoteStatus = "draft"     // 下書き
    QuoteStatusSent      QuoteStatus = "sent"       // 送付済み
    QuoteStatusAccepted  QuoteStatus = "accepted"   // 承認済み (受注)
    QuoteStatusRejected  QuoteStatus = "rejected"   // 否認 (失注)
    QuoteStatusCancelled QuoteStatus = "cancelled"  // キャンセル
)
```

**有効な状態遷移** *(要確認 — 以下はビジネスロジックからの推定)*:

```
draft → sent → accepted → cancelled
              → rejected
              → cancelled
```

> ⚠️ `accepted` から `draft` への巻き戻しが可能かどうかは未確認。`convert_to_billing` は `accepted` 状態でのみ使用可能と推定。

### 2.6 クエリパラメータ

#### GET /partners, GET /items

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | 名前・コードの部分一致検索 |

#### GET /billings

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | フリーワード検索 |
| `partner_id` | string | 取引先 ID でフィルタ |
| `payment_status` | string | `unsettled` \| `settled` |
| `from` | string | 開始日 (YYYY-MM-DD) |
| `to` | string | 終了日 (YYYY-MM-DD) |

> ⚠️ **取引先名検索**: API は `partner_id` のみ受け付ける。CLI の `--partner <name>` は名前で `GET /partners?q=<name>` を叩いて `partner_id` に変換してから billings を検索する。

#### GET /quotes

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `page` | int | ページ番号 |
| `per_page` | int | 件数/ページ |
| `q` | string | フリーワード検索 |
| `partner_id` | string | 取引先 ID でフィルタ |
| `status` | string | `draft` \| `sent` \| `accepted` \| `rejected` \| `cancelled` |
| `from` | string | 開始日 |
| `to` | string | 終了日 |

### 2.7 department_id 解決パターン

Invoice API では取引先に紐づく「部署 ID」(`department_id`) が請求書・見積書の必須パラメータ。
ユーザーは `partner_id` のみ知っているケースが多いため、以下のパターンで自動解決する。

```
1. partner_id を受け取る
2. GET /partners/{partner_id}/departments
3. departments[0].id を department_id として使用
4. departments が空の場合はエラー: "取引先に部署が登録されていません"
5. --department-id フラグで明示的な上書きも可能
```

### 2.8 インボイス制度 (適格請求書) 対応

2023年10月施行の適格請求書等保存方式 (インボイス制度) への対応:

- 新規作成は **必ず `/invoice_template_billings`** を使う (`/billings` POST は制度非対応)
- 明細行の `excise` (税区分) が必須
- `delivery_date` (納品日) は任意だが、区分記載請求書では推奨
- 適格請求書発行事業者の登録番号は `document_name` 等で管理 *(要確認)*

---

## 3. Expense API v1

### 3.1 Meta

| 項目 | 値 |
|------|---|
| Base URL (v1) | `https://expense.moneyforward.com/api/external/v1` |
| Base URL (v2) | `https://expense.moneyforward.com/api/external/v2` (office_members のみ) |
| OpenAPI Spec | `https://expense.moneyforward.com/api/index.html` |
| OpenAPI JSON | `https://expense.moneyforward.com/api/index.json` |
| GitHub | `https://github.com/moneyforward/expense-api-doc` |
| 認証 | OAuth2 (`expense.moneyforward.com/oauth/`) |
| スコープ | `office_setting:write transaction:write report:write user_setting:write account:write public_resource:read` |
| ページネーション | page-based (`page`, per_page デフォルト 25) |

### 3.2 エンドポイント一覧

#### 自分の情報

| Method | Path | スコープ | 説明 |
|--------|------|---------|------|
| GET | `/me` | — | 自分のオフィスメンバー情報 *(要確認)* |

#### 事業所

| Method | Path | 説明 |
|--------|------|------|
| GET | `/offices` | 所属事業所一覧 *(要確認)* |

#### 経費明細 (自分)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/ex_transactions` | 自分の経費明細一覧 |
| GET | `/ex_transactions/{id}` | 詳細 |
| POST | `/ex_transactions` | 作成 |
| PATCH | `/ex_transactions/{id}` | 更新 |
| DELETE | `/ex_transactions/{id}` | 削除 |

#### 経費明細 (組織全体 — 管理者権限)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/office/ex_transactions` | 全員の経費明細 *(要確認: パス)* |

#### 申請 (Reports)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/ex_reports` | 自分の申請一覧 |
| GET | `/ex_reports/{id}` | 詳細 |
| GET | `/office/ex_reports` | 全申請一覧 (管理者) *(要確認)* |

#### 承認

| Method | Path | 説明 |
|--------|------|------|
| GET | `/approvals` | 承認待ち一覧 *(要確認)* |
| PATCH | `/ex_reports/{id}/approve` | 承認 *(要確認: パス)* |
| PATCH | `/ex_reports/{id}/reject` | 否認 *(要確認: パス)* |

#### マスタデータ

| Method | Path | 必要スコープ | 説明 |
|--------|------|------------|------|
| GET | `/departments` | `public_resource:read` | 部門一覧 |
| GET | `/projects` | `public_resource:read` | プロジェクト一覧 |
| GET | `/ex_categories` | `public_resource:read` | 経費カテゴリ (勘定科目) 一覧 |
| GET | `/taxes` | `public_resource:read` | 税区分一覧 |
| GET | `/positions` | `public_resource:read` | 役職一覧 |

#### 従業員

| Method | Path | 説明 |
|--------|------|------|
| GET | `/v2/office_members` | 従業員一覧 (v2) *(要確認: フルパス)* |
| GET | `/v2/office_members/{id}` | 詳細 (v2) |

#### 仕訳

| Method | Path | 説明 |
|--------|------|------|
| GET | `/journals/ex_transactions` | 経費明細ベースの仕訳一覧 *(要確認)* |
| GET | `/journals/ex_reports` | 申請ベースの仕訳一覧 *(要確認)* |

> ⚠️ 上記エンドポイントの正確なパスは OpenAPI spec (`expense.moneyforward.com/api/index.json`) で確認すること。特に管理者向けエンドポイントはパスが変わる可能性がある。

### 3.3 Go データモデル (主要型のみ)

```go
type ExTransaction struct {
    ID          string `json:"id"`
    Amount      int    `json:"amount"`
    Memo        string `json:"memo,omitempty"`
    GeneratedAt string `json:"generated_at"` // YYYY-MM-DD
    CategoryID  string `json:"ex_category_id,omitempty"`
    DepartmentID string `json:"department_id,omitempty"`
    ProjectID   string `json:"project_id,omitempty"`
    TaxID       string `json:"tax_id,omitempty"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
    // *(要確認: フィールド一覧は OpenAPI spec 参照)*
}

type ExReport struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Status string `json:"status"` // submitted | approved | rejected | etc. *(要確認)*
    // *(要確認: フィールド一覧は OpenAPI spec 参照)*
}

type ExCategory struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    // *(要確認)*
}
```

### 3.4 Gotchas

- **v1 / v2 混在**: `office_members` のみ v2 エンドポイント。Base URL の切り替えが必要。
- **スコープ**: `public_resource:read` がないとマスタデータ (カテゴリ等) が取得できない。
- **管理者権限**: `office/` 配下のエンドポイントは Expense の管理者ロールが必要。一般ユーザーは 403。

---

## 4. Payable API v1

### 4.1 Meta

| 項目 | 値 |
|------|---|
| Base URL | `https://payable.moneyforward.com/api/external/v1` |
| OpenAPI Spec | `https://payable.moneyforward.com/api/index.html` |
| 認証 | OAuth2 (`payable.moneyforward.com/oauth/`) |
| スコープ | Expense と同一 (`mf_expense_oauth`) |
| ページネーション | page-based |

### 4.2 エンドポイント一覧

#### 事業所

| Method | Path | 説明 |
|--------|------|------|
| GET | `/offices` | 事業所一覧 *(要確認)* |

#### 支払依頼 (Reports)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/ap_reports` | 支払依頼一覧 *(要確認: パス)* |
| GET | `/ap_reports/{id}` | 詳細 |

#### 承認

| Method | Path | 説明 |
|--------|------|------|
| PATCH | `/ap_reports/{id}/approve` | 承認 *(要確認)* |
| PATCH | `/ap_reports/{id}/reject` | 否認 *(要確認)* |

#### 取引先 (Counterparties)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/ap_counterparties` | 一覧 *(要確認: パス)* |
| GET | `/ap_counterparties/{id}` | 詳細 |
| POST | `/ap_counterparties` | 作成 |
| PATCH | `/ap_counterparties/{id}` | 更新 |
| DELETE | `/ap_counterparties/{id}` | 削除 |

#### マスタデータ (Expense と共通)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/departments` | 部門一覧 |
| GET | `/projects` | プロジェクト一覧 |
| GET | `/ex_categories` | カテゴリ一覧 |
| GET | `/taxes` | 税区分一覧 |
| GET | `/positions` | 役職一覧 |

#### 仕訳

| Method | Path | 説明 |
|--------|------|------|
| GET | `/journals` | 仕訳一覧 *(要確認)* |

> ⚠️ Payable は Expense と OAuth エンドポイント・スコープを共有するが、**別の OAuth クライアント**として登録が必要。同一 client_id で両方を使えるかは App Portal の設定による *(要確認)*。

---

## 5. Payroll API v2

### 5.1 Meta

| 項目 | 値 |
|------|---|
| Base URL | `https://payroll.moneyforward.com/api/v2` |
| OpenAPI Spec | `https://payroll.moneyforward.com/api/v2/document` |
| 認証 | **IP ホワイトリスト + API Identifier** (OAuth2 ではない) |
| 認証方式詳細 | クエリパラメータ `?identifier={API_IDENTIFIER_UUID}` |
| ページネーション | *(要確認)* |
| アクセス権限 | 読み取り専用 (参照系のみ) |

### 5.2 認証設定

```
GET /employees?identifier={API_IDENTIFIER}
```

**設定項目**:
- `office_api_key`: 会社固有の UUID (設定ファイルで管理)
- IP ホワイトリスト: 管理コンソールで許可 IP を登録する必要がある

**設定ファイル** (`~/.config/mf/config.yaml`):

```yaml
profiles:
  default:
    payroll_api_identifier: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

> ⚠️ Payroll は OAuth2 フローが不要。`mf auth login --service payroll` ではなく `mf payroll setup --identifier <uuid>` のような独立した設定コマンドが必要。

### 5.3 エンドポイント一覧

#### 従業員

| Method | Path | 説明 |
|--------|------|------|
| GET | `/employees` | 従業員一覧 *(要確認)* |
| GET | `/employees/{id}` | 詳細 |

#### 部門

| Method | Path | 説明 |
|--------|------|------|
| GET | `/departments` | 部門一覧 *(要確認)* |

#### 給与明細

| Method | Path | 説明 |
|--------|------|------|
| GET | `/payrolls` | 給与明細一覧 *(要確認)* |
| GET | `/payrolls/{id}` | 詳細 |

#### 賞与

| Method | Path | 説明 |
|--------|------|------|
| GET | `/bonuses` | 賞与一覧 *(要確認)* |
| GET | `/bonuses/{id}` | 詳細 |

#### 設定項目

| Method | Path | 説明 |
|--------|------|------|
| GET | `/payment_items` | 支給項目 *(要確認)* |
| GET | `/deduction_items` | 控除項目 *(要確認)* |
| GET | `/attendance_items` | 勤怠項目 *(要確認)* |

### 5.4 Gotchas

- **IP ホワイトリスト**: 開発環境 (自宅 IP) と本番環境 (サーバー IP) を両方登録する必要がある。動的 IP の場合は NAT ゲートウェイ経由を推奨。
- **全体的に *(要確認)*** : Payroll API は他サービスに比べてドキュメントが少ない。公式ドキュメント (`payroll.moneyforward.com/api/v2/document`) を必ず参照すること。

---

## 6. Admina API v1

### 6.1 Meta

| 項目 | 値 |
|------|---|
| Base URL | `https://api.itmc.i.moneyforward.com/api/v1` |
| 認証 | Bearer Token (API Key) |
| 認証ヘッダー | `Authorization: Bearer {ADMINA_API_KEY}` |
| Organization ID | 全パスに `{orgId}` が必要 |
| ページネーション | Cursor-based (`limit` max 200, `cursor`) |

> **`X-Request-Source` ヘッダーについて**: ref 実装 (`admina-mcp-server`) では `X-Request-Source: mcp` を送っているが、これは MCP サーバー用の識別子。CLI からは `X-Request-Source: mf-cli` を送るか、ヘッダー自体を省略する *(要確認: 必須かどうか)*。

> **Organization ID の取得**: Admina 管理コンソール → 設定 → API → Organization ID を確認。API Key もここで発行。

### 6.2 エンドポイント一覧

#### 組織

| Method | Path | 説明 |
|--------|------|------|
| GET | `/organizations/{orgId}` | 組織情報取得 |

#### 従業員 (Identity)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/organizations/{orgId}/identity` | 一覧 (cursor-based) |
| POST | `/organizations/{orgId}/identity` | 作成 |
| GET | `/organizations/{orgId}/identity/{id}` | 詳細 |
| PUT | `/organizations/{orgId}/identity/{id}` | 更新 (全フィールド置換) |
| DELETE | `/organizations/{orgId}/identity/{id}` | 削除 |
| GET | `/organizations/{orgId}/identity/stats` | 統計 (ステータス別カウント) |
| GET | `/organizations/{orgId}/identity/check` | 管理タイプ確認 (email で検索) |

#### デバイス

| Method | Path | 説明 |
|--------|------|------|
| POST | `/organizations/{orgId}/devices/search` | 検索 (GET ではなく POST で検索条件を送る) |
| POST | `/organizations/{orgId}/devices` | 作成 |
| PATCH | `/organizations/{orgId}/devices/{id}` | 更新 (preset フィールド) |
| PATCH | `/organizations/{orgId}/devices/{id}/meta` | メタ更新 (status, 担当者) |

> ⚠️ デバイス検索は GET ではなく **POST**。検索条件を JSON ボディで送る。

#### SaaS サービス・アカウント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/organizations/{orgId}/services` | サービス一覧 |
| GET | `/organizations/{orgId}/services/{id}/accounts` | サービスのアカウント一覧 |
| GET | `/organizations/{orgId}/people/{id}/accounts` | 人物の SaaS アカウント一覧 |

### 6.3 Go データモデル

#### Identity (従業員)

> ⚠️ Admina API は **camelCase JSON キー** を使う (Invoice の snake_case とは異なる)。

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
    AssetNumber         string `json:"asset_number"`
    Subtype             string `json:"subtype"`
    ModelName           string `json:"model_name"`
    SerialNumber        string `json:"serial_number,omitempty"`
    ModelNumber         string `json:"model_number,omitempty"`
    Memory              string `json:"memory,omitempty"`
    HDDSSD              string `json:"hdd_ssd,omitempty"`
    CPU                 string `json:"cpu,omitempty"`
    OS                  string `json:"os,omitempty"`
    Size                string `json:"size,omitempty"`
    Manufacturer        string `json:"manufacturer,omitempty"`
    Supplier            string `json:"supplier,omitempty"`
    ProcurementMethod   string `json:"procurement_method,omitempty"`
    PurchaseDate        string `json:"purchase_date,omitempty"`
    PurchaseCost        string `json:"purchase_cost,omitempty"`
    WarrantyPeriod      string `json:"warranty_period,omitempty"`
    DecommissionDate    string `json:"decommission_date,omitempty"`
    ScheduledReturnDate string `json:"scheduled_return_date,omitempty"`
    FixedAsset          string `json:"fixed_asset,omitempty"`
    PhoneNumber         string `json:"phone_number,omitempty"`
    SIMNumber           string `json:"sim_number,omitempty"`
    MobilePlan          string `json:"mobile_plan,omitempty"`
    Hostname            string `json:"hostname,omitempty"`
    Version             string `json:"version,omitempty"`
    KeyboardLayout      string `json:"keyboard_layout,omitempty"`
    UsageStartDate      string `json:"usage_start_date,omitempty"`
    UsageEndDate        string `json:"usage_end_date,omitempty"`
}
```

#### Service / Account

```go
type Service struct {
    ID   string `json:"id"`
    Name string `json:"name,omitempty"`
    // *(要確認: icon, category 等のフィールドが存在する可能性あり)*
}

type Account struct {
    ID    string      `json:"id"`
    Email string      `json:"email,omitempty"`
    Role  AccountRole `json:"role,omitempty"`
    Type  AccountType `json:"type,omitempty"`
}
```

### 6.4 Enum 値

#### EmployeeStatus — 従業員ステータス

```go
type EmployeeStatus string

const (
    EmployeeStatusActive    EmployeeStatus = "active"     // 在籍
    EmployeeStatusOnLeave   EmployeeStatus = "on_leave"   // 休職中
    EmployeeStatusDraft     EmployeeStatus = "draft"      // 招待前 (アカウント未作成)
    EmployeeStatusPreactive EmployeeStatus = "preactive"  // 入社前
    EmployeeStatusRetired   EmployeeStatus = "retired"    // 退職済み
    EmployeeStatusUntracked EmployeeStatus = "untracked"  // 未追跡 (SaaS アカウントのみ検出)
    EmployeeStatusArchived  EmployeeStatus = "archived"   // アーカイブ済み
)
```

#### EmployeeType — 雇用形態

```go
type EmployeeType string

const (
    // 一般的な雇用形態
    EmployeeTypeBoard        EmployeeType = "board_member"         // 役員
    EmployeeTypeFullTime     EmployeeType = "full_time_employee"   // 正社員
    EmployeeTypeFixedTime    EmployeeType = "fixed_time_employee"  // 契約社員
    EmployeeTypeTemporary    EmployeeType = "temporary_employee"   // 派遣社員
    EmployeeTypePartTime     EmployeeType = "part_time_employee"   // パートタイム
    EmployeeTypeSecondment   EmployeeType = "secondment_employee"  // 出向
    EmployeeTypeContract     EmployeeType = "contract_employee"    // 業務委託
    EmployeeTypeCollaborator EmployeeType = "collaborator"         // 外部協力者 (業務委託・フリーランス等)

    // 特殊アカウント種別 (実際の人物ではない)
    EmployeeTypeGroupAddress  EmployeeType = "group_address"   // グループメールアドレス (部署共用等)
    EmployeeTypeSharedAddress EmployeeType = "shared_address"  // 共有アカウント
    EmployeeTypeTestAddress   EmployeeType = "test_address"    // テスト用アカウント

    // その他
    EmployeeTypeOther        EmployeeType = "other"
    EmployeeTypeUnknown      EmployeeType = "unknown"
    EmployeeTypeUnregistered EmployeeType = "unregistered"
)
```

> ⚠️ `group_address`, `shared_address`, `test_address` は人物ではなく特殊なアカウント種別。`identities list` でフィルタリングしたい場合は `--type` フラグで除外することを検討。

#### ManagementType

```go
type ManagementType string

const (
    ManagementTypeManaged      ManagementType = "managed"      // Admina 管理下
    ManagementTypeExternal     ManagementType = "external"     // 外部 (Admina 外で管理)
    ManagementTypeSystem       ManagementType = "system"       // システムアカウント
    ManagementTypeUnknown      ManagementType = "unknown"
    ManagementTypeUnregistered ManagementType = "unregistered"
)
```

#### DeviceType / DeviceSubtype

```go
type DeviceType string

const (
    DeviceTypePC    DeviceType = "pc"
    DeviceTypePhone DeviceType = "phone"
    DeviceTypeOther DeviceType = "other"
)

type DeviceSubtype string

const (
    DeviceSubtypeDesktopPC        DeviceSubtype = "desktop_pc"
    DeviceSubtypeLaptopPC         DeviceSubtype = "laptop_pc"
    DeviceSubtypeTabletPC         DeviceSubtype = "tablet_pc"
    DeviceSubtypePhone            DeviceSubtype = "phone"
    DeviceSubtypeMonitor          DeviceSubtype = "monitor"
    DeviceSubtypeServer           DeviceSubtype = "server"
    DeviceSubtypePeripheralDevice DeviceSubtype = "peripheral_device"
    DeviceSubtypeOther            DeviceSubtype = "other"
)
```

#### DeviceStatus

```go
type DeviceStatus string

const (
    DeviceStatusOnOrder        DeviceStatus = "on_order"        // 発注中
    DeviceStatusInStock        DeviceStatus = "in_stock"        // 在庫あり (未割当)
    DeviceStatusPreUse         DeviceStatus = "pre_use"         // 使用準備中
    DeviceStatusActive         DeviceStatus = "active"          // 使用中
    DeviceStatusMissing        DeviceStatus = "missing"         // 紛失
    DeviceStatusMalfunction    DeviceStatus = "malfunction"     // 故障
    DeviceStatusDecommissioned DeviceStatus = "decommissioned"  // 廃棄済み
)
```

#### AccountRole / AccountType

```go
type AccountRole string

const (
    AccountRoleAdmin AccountRole = "admin"
    AccountRoleGuest AccountRole = "guest"
    AccountRoleOther AccountRole = "other"
)

type AccountType string

const (
    AccountTypeEmployee AccountType = "employee"
    AccountTypeGuest    AccountType = "guest"
    AccountTypeSystem   AccountType = "system"
    AccountTypeUnknown  AccountType = "unknown"
)
```

### 6.5 Cursor ページネーション実装パターン

```go
var results []Identity
var cursor string

for {
    resp, err := client.GetIdentities(orgID, limit, cursor)
    results = append(results, resp.Results...)
    if resp.NextCursor == "" {
        break
    }
    cursor = resp.NextCursor
}
```

---

## 7. App Portal 設定ガイド

OAuth2 を使う前に `https://app-portal.moneyforward.com` でアプリケーションを登録する必要がある。

### 7.1 アプリケーション登録手順

1. `https://app-portal.moneyforward.com` にログイン
2. 「アプリケーション作成」→ アプリ名・説明を入力
3. **Redirect URI** を登録:
   - 開発環境: `http://localhost` (ポートなし、またはワイルドカード *(要確認)*)
   - ⚠️ MF App Portal がポート付き URI (`http://localhost:38080/callback`) を許可するか確認が必要
   - 許可しない場合: 固定ポートを1つ登録し、`MF_CALLBACK_PORT` で固定する
4. 必要なスコープを選択 (サービス別に異なる — Section 1.2 参照)
5. `client_id` と `client_secret` を取得

### 7.2 サービス別 client_id

各サービス (Invoice, Expense, Payable) で別々のアプリ登録が必要か、または1つの登録で複数スコープを付与できるかは *(要確認)*。

現時点の推定:
- Invoice は `api.biz.moneyforward.com` の OAuth → Invoice 専用アプリ登録
- Expense/Payable は各サービスドメインの OAuth → 別々の登録が必要と推定

### 7.3 Admina API Key の取得

1. Admina 管理コンソール (`https://app.admina.mntsq.co.jp` 等) にログイン
2. 設定 → API 連携 → API Key を発行
3. **Organization ID** (orgId) も同画面で確認できる
4. `mf admina auth setup --api-key <key> --org-id <org-id>` で設定

### 7.4 Payroll API Identifier の取得・IP 登録

1. MF クラウド給与の管理者コンソールにログイン
2. 外部連携 → API 設定 → API Identifier (UUID) を確認
3. 同画面でアクセス元 IP アドレスを登録 (IPv4, CIDR 記法対応 *(要確認)*)
4. CI/CD 環境の場合は NAT ゲートウェイの固定 IP を登録
