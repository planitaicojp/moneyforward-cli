# Money Forward CLI (mf) — 設計ドキュメント

## 1. コンセプト

```
mf <service> <command> [flags]
```

AWS CLI (`aws s3 ls`), GitHub CLI (`gh pr list`), ConoHa CLI (`conoha server list`) と同様の
UX を提供する、Money Forward クラウド向け Agent-Friendly CLI ツール。

**バイナリ名**: `mf`

---

## 2. Usage 一覧

### 2.1 グローバルフラグ

```
Global Flags:
      --profile string   使用するプロファイル名 (default: active profile)
      --format string    出力形式: table|json|yaml|csv (default: "table")
      --no-input         非対話モード (プロンプトを無効化)
      --quiet            最小限の出力のみ表示
      --verbose          詳細なデバッグ出力
      --no-color         カラー出力を無効化
  -h, --help             ヘルプを表示
```

### 2.2 auth — 認証管理

```bash
# OAuth2 ログイン (ブラウザが開く)
mf auth login [--scopes <scope1,scope2>]

# 認証状態の確認
mf auth status

# 登録済みプロファイル一覧
mf auth list

# プロファイル切り替え
mf auth switch <profile-name>

# ログアウト (トークン削除)
mf auth logout [--profile <name>]

# プロファイル削除 (設定ごと削除)
mf auth remove <profile-name>

# アクセストークンを stdout に出力 (スクリプト用)
mf auth token
```

**OAuth2 フロー (PKCE 対応)**:
1. `mf auth login` → `code_verifier` 生成、`code_challenge` (S256) 算出
2. ローカルサーバー起動 (localhost:PORT)
3. ブラウザで authorize エンドポイントを開く (`code_challenge` パラメータ付き)
4. ユーザーが承認 → コールバックで authorization code を受信
5. code + `code_verifier` → token エンドポイントでアクセストークン + リフレッシュトークン取得
6. `~/.config/mf/tokens.yaml` に保存

> **注意**: OAuth2 エンドポイントはサービスごとに異なる
> - **Invoice**: `https://api.biz.moneyforward.com/authorize`, `/token`
> - **Expense**: `https://expense.moneyforward.com/oauth/authorize`, `/oauth/token`
> - **Payable**: `https://payable.moneyforward.com/oauth/authorize`, `/oauth/token`
> - **Payroll**: OAuth2 ではなく IP + API Identifier 認証 (別フロー)

### 2.3 config — 設定管理

```bash
# 設定値の取得
mf config get <key>

# 設定値の変更
mf config set <key> <value>

# 全設定の一覧
mf config list

# 設定ファイルの場所を表示
mf config path
```

**設定ファイル**: `~/.config/mf/config.yaml`

```yaml
version: 1
active_profile: default
defaults:
  format: table
profiles:
  default:
    client_id: "your-client-id"
    office_id: "your-office-id"
    scopes:
      - "mfc/admin/tenant.read"
  company-b:
    client_id: "another-client-id"
    office_id: "another-office-id"
    scopes:
      - "mfc/admin/tenant.read"
      - "office_setting:write"
```

> **注意**: `client_secret` は `credentials.yaml` に保存する (config.yaml には含めない)

### 2.4 invoice — クラウド請求書

> **Base URL**: `https://invoice.moneyforward.com/api/v3/`
> **認証**: OAuth2 (`api.biz.moneyforward.com`)
> **スコープ**: `mfc/invoice/data.read`, `mfc/invoice/data.write`
> **レート制限**: 3 req/sec
> **ページネーション**: page-based (`page`, `per_page`)
> **データモデル・エンドポイント詳細**: → `SPEC.md` Section 2

```bash
### 事業所
mf invoice office show

### 取引先
mf invoice partners list [--page N] [--per-page N] [--query <q>] [--all]
mf invoice partners show <id>
mf invoice partners create --name <name> [--name-kana <kana>] [--code <code>] [--memo <text>]
mf invoice partners update <id> [--name <name>] [--name-kana <kana>] [--code <code>] [--memo <text>]
mf invoice partners delete <id>
mf invoice partners departments list <partner-id>

### 品目
mf invoice items list [--page N] [--per-page N] [--query <q>] [--all]
mf invoice items show <id>
mf invoice items create --name <name> [--code <code>] [--detail <text>] [--unit <unit>] [--price <n>] [--quantity <n>] [--excise <type>]
mf invoice items update <id> [--name <name>] [--code <code>] [--detail <text>] [--unit <unit>] [--price <n>] [--quantity <n>] [--excise <type>]
mf invoice items delete <id>

### 請求書
mf invoice billings list [--page N] [--per-page N] [--partner-id <id>] [--partner <name>] [--payment-status <unsettled|settled>] [--from <YYYY-MM-DD>] [--to <YYYY-MM-DD>] [--query <q>] [--all]
mf invoice billings show <id>
mf invoice billings create --partner-id <id> --billing-date <YYYY-MM-DD> [--item "name=X,price=N,quantity=N,excise=10" ...] [--items-file <path>] [--title <text>] [--memo <text>] [--payment-condition <text>] [--due-date <YYYY-MM-DD>] [--sales-date <YYYY-MM-DD>] [--department-id <id>] [--dry-run]
mf invoice billings update <id> [--title <text>] [--memo <text>] [--payment-condition <text>] [--billing-date <YYYY-MM-DD>] [--due-date <YYYY-MM-DD>] [--sales-date <YYYY-MM-DD>] [--items-file <path>] [--dry-run]
mf invoice billings delete <id>
mf invoice billings set-payment-status <id> --status <unsettled|settled>
mf invoice billings pdf <id> [--download] [--output <path>]

### 見積書
mf invoice quotes list [--page N] [--per-page N] [--partner-id <id>] [--partner <name>] [--status <draft|sent|accepted|rejected|cancelled>] [--from <YYYY-MM-DD>] [--to <YYYY-MM-DD>] [--query <q>] [--all]
mf invoice quotes show <id>
mf invoice quotes create --partner-id <id> --quote-date <YYYY-MM-DD> --expired-date <YYYY-MM-DD> [--item "name=X,price=N,quantity=N,excise=10" ...] [--items-file <path>] [--title <text>] [--memo <text>] [--dry-run]
mf invoice quotes update <id> [--title <text>] [--memo <text>] [--quote-date <YYYY-MM-DD>] [--expired-date <YYYY-MM-DD>] [--items-file <path>] [--dry-run]
mf invoice quotes delete <id>
mf invoice quotes set-status <id> --status <draft|sent|accepted|rejected|cancelled>
mf invoice quotes to-billing <id>
mf invoice quotes pdf <id> [--download] [--output <path>]

### 送付履歴
mf invoice sent-histories list [--page N] [--per-page N] [--all]
```

#### 明細行の入力方法 (3通り)

```bash
# 1. --item フラグ (繰り返し可、最も手軽)
mf invoice billings create --partner-id X --billing-date 2026-04-01 \
  --item "name=コンサルティング,price=100000,quantity=1,excise=10" \
  --item "name=交通費,price=10000,quantity=3,excise=10"

# 2. --items-file (複雑なケース用)
mf invoice billings create --partner-id X --billing-date 2026-04-01 \
  --items-file items.json

# 3. stdin (Agent-friendly)
echo '[{"name":"開発費","price":500000,"quantity":1,"excise":"ten_percent"}]' | \
  mf invoice billings create --partner-id X --billing-date 2026-04-01 -
```

#### excise 短縮エイリアス

| 短縮 | 正式値 | 説明 |
|------|--------|------|
| `10` | `ten_percent` | 10% |
| `8` | `eight_percent` | 8% |
| `8r` | `eight_percent_as_reduced_tax_rate` | 軽減8% |
| `5` | `five_percent` | 5% |
| `0` | `untaxable` | 不課税 |
| `exempt` | `tax_exemption` | 免税 |
| `non` | `non_taxable` | 非課税 |

### 2.5 expense — クラウド経費

> **Base URL**: `https://expense.moneyforward.com/api/external/v1/` (v2 for office_members)
> **認証**: OAuth2 (`expense.moneyforward.com/oauth/`)
> **スコープ**: `office_setting:write`, `transaction:write`, `report:write`, `user_setting:write`, `account:write`, `public_resource:read`
> **ページネーション**: page-based (`page`), max 25-200件/ページ

```bash
# 事業所
mf expense offices list

# 経費明細 (自分)
mf expense transactions list [--limit N] [--page N] [--all]
mf expense transactions show <id>
mf expense transactions create [flags]
mf expense transactions update <id> [flags]
mf expense transactions delete <id>

# 経費明細 (組織全体)
mf expense transactions list --scope org [--limit N] [--page N] [--all]

# 申請
mf expense reports list [--limit N] [--page N] [--all]
mf expense reports show <id>
mf expense reports list --scope org

# 承認
mf expense approvals list
mf expense approvals approve <report-id> [--message <text>]
mf expense approvals reject <report-id> [--message <text>]

# マスタデータ
mf expense departments list [--limit N] [--page N] [--all]
mf expense projects list [--limit N] [--page N] [--all]
mf expense categories list [--limit N] [--page N] [--all]
mf expense taxes list
mf expense positions list

# 従業員
mf expense members list [--limit N] [--page N] [--all]
mf expense members show <id>
mf expense members me

# 仕訳
mf expense journals list --by transactions [--from <date>] [--to <date>]
mf expense journals list --by reports [--from <date>] [--to <date>]
```

### 2.6 payable — クラウド債務支払

> **Base URL**: `https://payable.moneyforward.com/api/external/v1/`
> **認証**: OAuth2 (`payable.moneyforward.com/oauth/`)
> **スコープ**: Expense と同一 (mf_expense_oauth)
> **ページネーション**: page-based

```bash
# 事業所
mf payable offices list

# 支払依頼
mf payable reports list [--limit N] [--page N] [--all]

# 承認
mf payable approvals approve <report-id>
mf payable approvals reject <report-id>

# 取引先
mf payable counterparties list [--limit N] [--page N] [--all]
mf payable counterparties show <id>
mf payable counterparties create [flags]
mf payable counterparties update <id> [flags]
mf payable counterparties delete <id>

# マスタデータ (Expense と共有)
mf payable departments list [--limit N] [--page N] [--all]
mf payable projects list [--limit N] [--page N] [--all]
mf payable categories list [--limit N] [--page N] [--all]
mf payable taxes list
mf payable positions list

# 仕訳
mf payable journals list [--limit N] [--page N] [--all]
```

### 2.7 payroll — クラウド給与

> ⚠️ Payroll API は OAuth2 ではなく IP ホワイトリスト + API Identifier 認証を使用
> **Base URL**: `https://payroll.moneyforward.com/api/v2/`
> **認証**: IP ホワイトリスト + API Identifier (`?identifier=UUID`)
> **会社識別子**: `office_api_key` (UUID)

```bash
# 従業員
mf payroll employees list
mf payroll employees show <id>

# 部門
mf payroll departments list

# 給与
mf payroll payrolls list
mf payroll payrolls show <id>

# 賞与
mf payroll bonuses list
mf payroll bonuses show <id>

# 設定項目
mf payroll payment-items list
mf payroll deduction-items list
mf payroll attendance-items list
```

### 2.8 admina — Admina (IT デバイス・SaaS 管理)

> **Base URL**: `https://api.itmc.i.moneyforward.com/api/v1`
> **認証**: API Key + Organization ID (Bearer トークン)
> **ページネーション**: cursor-based (`limit`, `cursor`)
> **データモデル・エンドポイント詳細**: → `SPEC.md` Section 6

```bash
### 認証 (API Key)
mf admina auth setup --api-key <key> --org-id <org-id>
mf admina auth status

### 組織情報
mf admina org show

### 従業員 (Identity)
mf admina identities list [--status <active|on_leave|retired|...>] [--type <full_time_employee|...>] [--department <name>] [--keyword <q>] [--limit N] [--cursor <cursor>] [--all]
mf admina identities show <id>
mf admina identities create --first-name <name> --last-name <name> --status <status> --type <type> [--email <email>] [--employee-id <id>] [--department <name>] [--job-title <title>]
mf admina identities update <id> [flags]
mf admina identities delete <id>
mf admina identities stats
mf admina identities check --email <email>

### デバイス
mf admina devices list [--type <pc|phone|other>] [--status <active|in_stock|...>] [--keyword <q>] [--limit N] [--cursor <cursor>] [--all]
mf admina devices create --type <type> --asset-number <num> --model-name <name> [flags]
mf admina devices update <id> [flags]
mf admina devices update-meta <id> [--status <status>] [--assignee-id <id>]

### SaaS サービス・アカウント
mf admina services list [--limit N] [--cursor <cursor>] [--all]
mf admina accounts list --service-id <id> [--keyword <q>] [--role <admin|guest|other>] [--limit N] [--cursor <cursor>] [--all]
mf admina accounts list --people-id <id> [--keyword <q>] [--limit N] [--cursor <cursor>] [--all]
```

### 2.9 ユーティリティ

```bash
# バージョン表示
mf version

# シェル補完スクリプト生成
mf completion bash|zsh|fish|powershell

# ヘルプ
mf help [command]
```

---

## 3. 設定ファイル構成

```
~/.config/mf/
├── config.yaml        # プロファイル設定 (client_id, scopes 等)
├── tokens.yaml        # アクセストークン + リフレッシュトークン (0600)
└── credentials.yaml   # client_secret 等の機密情報 (0600)
```

**環境変数オーバーライド**:

| 環境変数 | 説明 |
|---------|------|
| `MF_PROFILE` | アクティブプロファイル |
| `MF_CLIENT_ID` | OAuth2 Client ID |
| `MF_CLIENT_SECRET` | OAuth2 Client Secret |
| `MF_ACCESS_TOKEN` | アクセストークン (キャッシュをスキップ) |
| `MF_FORMAT` | 出力形式 |
| `MF_CONFIG_DIR` | 設定ディレクトリ |
| `MF_NO_INPUT` | 非対話モード |
| `MF_DEBUG` | デバッグレベル |

---

## 4. 技術仕様

### Go module path

`github.com/planitaicojp/moneyforward-cli`

### OAuth2 エンドポイント (サービス別)

| サービス | Authorization URL | Token URL |
|---------|-------------------|-----------|
| Invoice | `https://api.biz.moneyforward.com/authorize` | `https://api.biz.moneyforward.com/token` |
| Expense | `https://expense.moneyforward.com/oauth/authorize` | `https://expense.moneyforward.com/oauth/token` |
| Payable | `https://payable.moneyforward.com/oauth/authorize` | `https://payable.moneyforward.com/oauth/token` |
| Payroll | N/A (IP + API Identifier) | N/A |
| Admina | N/A (API Key + Org ID) | N/A |

> CLI は public client のため、**PKCE (RFC 7636)** を必須とする。
> `code_challenge_method=S256` を使用。
> クライアント認証方式は `client_secret_post` (リクエストボディに含める)。

### Admina API エンドポイント

| 項目 | 値 |
|------|---|
| Base URL | `https://api.itmc.i.moneyforward.com/api/v1` |
| 認証ヘッダー | `Authorization: Bearer {ADMINA_API_KEY}` |
| 追加ヘッダー | `X-Request-Source: mcp` |

### ページネーション

2種類のページネーション方式をサポートする:

#### Page-based (Invoice / Expense / Payable)

- `--per-page N`: 1ページあたりの取得件数 (API パラメータ名 `per_page` に準拠)
- `--page N`: ページ番号指定
- `--all`: 全件取得 (自動ページング)

#### Cursor-based (Admina)

- `--limit N`: 1回あたりの取得件数 (max 200)
- `--cursor <cursor>`: カーソル指定
- `--all`: 全件取得 (自動カーソル送り)

### Invoice レート制限

- 3 req/sec
- 429 レスポンス時は `Retry-After` ヘッダーに従いリトライ

### 依存ライブラリ (最小構成)

| パッケージ | 用途 |
|-----------|------|
| `github.com/spf13/cobra` | CLI フレームワーク |
| `gopkg.in/yaml.v3` | YAML 読み書き |
| `github.com/manifoldco/promptui` | 対話プロンプト |

> SDK は使用しない。REST API を直接呼び出す。

### HTTP クライアント仕様

- Go 標準 `net/http` を使用
- タイムアウト: 30秒
- リトライ: 429 (Rate Limit) / 5xx に対して最大3回 (exponential backoff)
- デバッグモードでリクエスト/レスポンスのログ出力
- User-Agent: `mf-cli/<version>`

### エラーハンドリング

| 終了コード | 意味 |
|-----------|------|
| 0 | 成功 |
| 1 | 一般エラー |
| 2 | 認証エラー |
| 3 | リソース未検出 |
| 4 | バリデーションエラー |
| 5 | API エラー |
| 6 | ネットワークエラー |
| 10 | キャンセル |

---

## 5. 実装フェーズと進捗

### Phase 0: プロジェクト基盤 ✅

- [x] Go module 初期化 (`go mod init`)
- [x] ディレクトリ構成作成
- [x] Makefile 作成
- [x] .goreleaser.yaml 作成
- [x] .golangci.yml 作成
- [x] main.go + cmd/root.go (グローバルフラグ)
- [x] cmd/version.go
- [x] cmd/completion.go
- [x] internal/errors/ (型付きエラー + 終了コード)
- [x] internal/output/ (table/json/yaml/csv フォーマッタ)

### Phase 1: 認証基盤 ⬜

- [ ] internal/config/ (config.yaml, tokens.yaml, credentials.yaml)
- [ ] internal/api/oauth.go (OAuth2 Authorization Code フロー)
- [ ] internal/api/client.go (HTTP クライアント, リトライ, デバッグ)
- [ ] internal/prompt/ (対話プロンプト)
- [ ] cmd/auth/login (ブラウザ起動 + コールバック受信)
- [ ] cmd/auth/logout
- [ ] cmd/auth/status
- [ ] cmd/auth/list
- [ ] cmd/auth/switch
- [ ] cmd/auth/remove
- [ ] cmd/auth/token
- [ ] cmd/config/ (get, set, list, path)
- [ ] 環境変数オーバーライド

### Phase 2: Cloud Invoice API ⬜

> ドキュメント最良 (v3 Swagger UI)、CRUD 完備、実用性高い

- [ ] internal/api/invoice.go
- [ ] internal/model/invoice.go
- [ ] cmd/invoice/office (show)
- [ ] cmd/invoice/partners (list, show, create, update, delete)
- [ ] cmd/invoice/partners/departments (list)
- [ ] cmd/invoice/items (list, show, create, update, delete)
- [ ] cmd/invoice/billings (list, show, create, update, delete, set-payment-status, pdf)
- [ ] cmd/invoice/quotes (list, show, create, update, delete, set-status, to-billing, pdf)
- [ ] cmd/invoice/sent-histories (list)
- [ ] --item フラグ (繰り返し可) + --items-file + stdin 入力
- [ ] excise 短縮エイリアス (10, 8, 8r, 5, 0, exempt, non)
- [ ] department_id 自動解決 (パートナーの最初の部署)
- [ ] --dry-run (変更系コマンド)

### Phase 3: Cloud Expense API ⬜

> OpenAPI spec あり、エンドポイント最多

- [ ] internal/api/expense.go
- [ ] internal/model/expense.go
- [ ] cmd/expense/offices (list)
- [ ] cmd/expense/transactions (list, show, create, update, delete; --scope org)
- [ ] cmd/expense/reports (list, show; --scope org)
- [ ] cmd/expense/approvals (list, approve, reject)
- [ ] cmd/expense/departments (list)
- [ ] cmd/expense/projects (list)
- [ ] cmd/expense/categories (list)
- [ ] cmd/expense/taxes (list)
- [ ] cmd/expense/positions (list)
- [ ] cmd/expense/members (list, show, me)
- [ ] cmd/expense/journals (list --by transactions|reports)

### Phase 4: Cloud Payable API ⬜

> Expense と OAuth/マスタを共有、追加実装少

- [ ] internal/api/payable.go
- [ ] internal/model/payable.go
- [ ] cmd/payable/offices (list)
- [ ] cmd/payable/reports (list)
- [ ] cmd/payable/approvals (approve, reject)
- [ ] cmd/payable/counterparties (list, show, create, update, delete)
- [ ] cmd/payable/departments (list)
- [ ] cmd/payable/projects (list)
- [ ] cmd/payable/journals (list)

### Phase 5: Cloud Payroll API ⬜

> 別認証 (IP + API Identifier)、読み取り専用

- [ ] internal/api/payroll.go (IP + API Identifier 認証)
- [ ] internal/model/payroll.go
- [ ] cmd/payroll/employees (list, show)
- [ ] cmd/payroll/departments (list)
- [ ] cmd/payroll/payrolls (list, show)
- [ ] cmd/payroll/bonuses (list, show)
- [ ] cmd/payroll/payment-items (list)
- [ ] cmd/payroll/deduction-items (list)
- [ ] cmd/payroll/attendance-items (list)

### ~~Phase: Cloud Accounting API~~ (保留)

> ⚠️ Accounting API は非公開 (クローズドAPI)。一般開発者は利用不可。パートナー申請後に検討。

### Phase 6: Admina API ⬜

> IT デバイス・SaaS 管理。API Key 認証、cursor-based ページネーション。

- [ ] internal/api/admina.go
- [ ] internal/model/admina.go
- [ ] cmd/admina/auth (setup, status)
- [ ] cmd/admina/org (show)
- [ ] cmd/admina/identities (list, show, create, update, delete, stats, check)
- [ ] cmd/admina/devices (list, create, update, update-meta)
- [ ] cmd/admina/services (list)
- [ ] cmd/admina/accounts (list --service-id, list --people-id)

### Phase 7: リリース準備 ⬜

- [ ] README.md 作成
- [ ] goreleaser でクロスコンパイルビルド確認
- [ ] Homebrew tap 設定 (オプション)
- [ ] CI/CD (GitHub Actions)
- [ ] テストカバレッジ目標達成

---

## 6. API ドキュメント参照先

| サービス | ドキュメント URL | 公開状態 |
|---------|-----------------|---------|
| 共通仕様 | https://developers.biz.moneyforward.com/docs/common/api_common_specifications/ | — |
| Getting Started | https://developers.biz.moneyforward.com/docs/common/getting-started-moneyforward-cloud-apis/ | — |
| App Portal | https://app-portal.moneyforward.com | — |
| Cloud Invoice | https://invoice.moneyforward.com/docs/api/v3/index.html | **公開 (Open API)** |
| Cloud Expense | https://expense.moneyforward.com/api/index.html | **公開 (Open API)** |
| Cloud Payable | https://payable.moneyforward.com/api/index.html | **公開 (Open API)** |
| Cloud Payroll | https://payroll.moneyforward.com/api/v2/document | **公開 (Open API)** |
| Cloud Accounting | パートナー限定 | **非公開 (Closed API)** |
| Expense OpenAPI Spec | https://expense.moneyforward.com/api/index.json | JSON |
| Expense API GitHub | https://github.com/moneyforward/expense-api-doc | — |

---

## 7. 設計上の決定事項 (ADR)

### ADR-001: SDK を使用しない

- **決定**: Money Forward API を直接 REST で呼び出す
- **理由**: 公式 SDK が存在しない。非公式 SDK は API 更新への追従が遅延するリスクがある
- **影響**: API 変更時は手動で対応が必要だが、OpenAPI spec からの自動生成も将来的に検討可能

### ADR-002: conoha-cli アーキテクチャを踏襲

- **決定**: `~/dev/crowdy/conoha-cli` の3層構造 (cmd → api → config/model) を採用
- **理由**: 実績のあるパターン、Agent-friendly 設計が既に組み込まれている
- **差分**: 認証が OpenStack Keystone → OAuth2 に変更

### ADR-003: マルチプロファイル対応

- **決定**: `gh auth list` / `conoha auth list` と同様の複数アカウント管理
- **理由**: 複数事業所や複数環境 (本番/テスト) を切り替えて利用するユースケースが想定される
- **実装**: profile ごとに client_id, scopes, tokens を分離管理

### ADR-004: Payroll API の認証分離

- **決定**: Payroll API は別認証 (IP + API Identifier) として扱う
- **理由**: 他の MF Cloud サービスと OAuth2 エンドポイントが異なる
- **実装**: `mf payroll` サブコマンド内で独自の認証設定を持つ

### ADR-005: PKCE 必須

- **決定**: OAuth2 Authorization Code フローに PKCE (RFC 7636) を使用する
- **理由**: CLI はブラウザと異なり client_secret を安全に保持できない (public client)。PKCE により authorization code interception 攻撃を防止する
- **実装**: `code_verifier` (43-128文字のランダム文字列) → SHA256 → base64url = `code_challenge`

### ADR-006: サービス別 OAuth2 エンドポイント

- **決定**: サービスごとに異なる OAuth2 エンドポイントを使い分ける
- **理由**: Invoice は `api.biz.moneyforward.com`、Expense/Payable は各サービスドメインで OAuth2 を提供
- **実装**: プロファイルにサービスごとのトークンを保持。`mf auth login --service <invoice|expense|payable>` で対象サービスを指定

### ADR-007: Cloud Accounting API は対象外

- **決定**: Cloud Accounting (会計) API は実装対象外とする
- **理由**: Accounting API は非公開 (クローズドAPI) であり、一般開発者には利用不可。社内関係者や特定パートナー企業向けのみ提供
- **影響**: パートナー申請が承認された場合に将来的に実装を検討

### ADR-008: Admina API 追加

- **決定**: Admina (IT デバイス・SaaS 管理) を `mf admina` として Phase 6 に追加
- **理由**: Money Forward グループの IT 資産管理サービスであり、CLI での操作ニーズがある。ref リポジトリ (`admina-mcp-server`) で API 仕様が確認済み
- **スコープ**: コアリソース (identities, devices, services, accounts) に集中。カスタムフィールド CRUD と merge コマンドは CLI スコープ外
- **認証**: API Key + Organization ID (OAuth2 ではなく Bearer トークン方式)

### ADR-009: 2種類のページネーション対応

- **決定**: page-based (Invoice/Expense/Payable) と cursor-based (Admina) の2方式をサポート
- **理由**: API サービスごとにページネーション方式が異なる
- **実装**: page-based は `--page`, `--per-page` フラグ、cursor-based は `--limit`, `--cursor` フラグ。`--all` は両方式で全件取得をサポート
- **CLI パラメータ名**: API パラメータ名に準拠 (`--per-page` = API の `per_page`, `--limit` = API の `limit`)

### ADR-010: department_id 自動解決

- **決定**: 請求書・見積書作成時に `department_id` をパートナーの最初の部署から自動解決する
- **理由**: mf-invoice-mcp の実装パターンを踏襲。ユーザーは `partner_id` のみ指定すればよい
- **実装**: `GET /partners/{id}/departments` → `departments[0].id` を使用。部署が0件の場合はエラー。`--department-id` フラグで明示的な上書きも可能

### ADR-011: 明細行の複数入力方式

- **決定**: `--item` (繰り返し可), `--items-file`, stdin の3通りの入力方式をサポート
- **理由**: CLI 直接入力 (手軽)、JSON ファイル (複雑なケース)、パイプ (Agent-friendly) の各ユースケースに対応
- **excise 短縮エイリアス**: `10`→`ten_percent`, `8`→`eight_percent`, `8r`→`eight_percent_as_reduced_tax_rate` 等。正式値もそのまま使用可能
