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

**OAuth2 フロー**:
1. `mf auth login` → ローカルサーバー起動 (localhost:PORT)
2. ブラウザで `https://api.biz.moneyforward.com/authorize` を開く
3. ユーザーが承認 → コールバックで authorization code を受信
4. code → token エンドポイントでアクセストークン + リフレッシュトークン取得
5. `~/.config/mf/tokens.yaml` に保存

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
    client_secret: "your-client-secret"
    scopes:
      - "mfc/admin/tenant.read"
  company-b:
    client_id: "another-client-id"
    client_secret: "another-client-secret"
    scopes:
      - "mfc/admin/tenant.read"
      - "office_setting:write"
```

### 2.4 accounting — クラウド会計

```bash
# 事業所一覧
mf accounting offices list

# 勘定科目一覧
mf accounting accounts list [--office-id <id>]

# 仕訳一覧
mf accounting journals list [--office-id <id>] [--from <date>] [--to <date>]

# 仕訳詳細
mf accounting journals show <journal-id>

# 取引先一覧
mf accounting partners list [--office-id <id>]

# 部門一覧
mf accounting departments list [--office-id <id>]

# 税区分一覧
mf accounting taxes list [--office-id <id>]
```

### 2.5 expense — クラウド経費

```bash
# 経費申請一覧
mf expense reports list [--status <draft|pending|approved|rejected>]

# 経費申請詳細
mf expense reports show <report-id>

# 経費明細一覧
mf expense transactions list [--report-id <id>]

# 経費明細詳細
mf expense transactions show <transaction-id>

# 経費カテゴリ一覧
mf expense categories list

# 部門一覧
mf expense departments list

# プロジェクト一覧
mf expense projects list

# 承認
mf expense reports approve <report-id>

# 差戻し
mf expense reports reject <report-id> --reason <text>
```

### 2.6 invoice — クラウド請求書

```bash
# 請求書一覧
mf invoice billings list [--from <date>] [--to <date>] [--status <string>]

# 請求書詳細
mf invoice billings show <billing-id>

# 請求書作成
mf invoice billings create --partner-id <id> --items-file <path>

# 請求書更新
mf invoice billings update <billing-id> [flags]

# 請求書削除
mf invoice billings delete <billing-id>

# 請求書 PDF ダウンロード
mf invoice billings download <billing-id> [--output <path>]

# 取引先一覧
mf invoice partners list

# 取引先詳細
mf invoice partners show <partner-id>

# 品目一覧
mf invoice items list
```

### 2.7 payroll — クラウド給与

> ⚠️ Payroll API は OAuth2 ではなく IP ホワイトリスト + API Identifier 認証を使用

```bash
# 従業員一覧
mf payroll employees list [--office-key <key>]

# 従業員詳細
mf payroll employees show <employee-id>

# 給与明細一覧
mf payroll payslips list [--year <year>] [--month <month>]

# 給与明細詳細
mf payroll payslips show <payslip-id>

# 勤怠データ一覧
mf payroll attendances list [--year <year>] [--month <month>]
```

### 2.8 payable — クラウド債務支払

```bash
# 支払依頼一覧
mf payable requests list [--status <string>]

# 支払依頼詳細
mf payable requests show <request-id>

# 支払一覧
mf payable payments list

# 支払詳細
mf payable payments show <payment-id>

# 取引先一覧
mf payable partners list
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

### Phase 0: プロジェクト基盤 ⬜

- [ ] Go module 初期化 (`go mod init`)
- [ ] ディレクトリ構成作成
- [ ] Makefile 作成
- [ ] .goreleaser.yaml 作成
- [ ] .golangci.yml 作成
- [ ] main.go + cmd/root.go (グローバルフラグ)
- [ ] cmd/version.go
- [ ] cmd/completion.go
- [ ] internal/errors/ (型付きエラー + 終了コード)
- [ ] internal/output/ (table/json/yaml/csv フォーマッタ)

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

### Phase 2: Cloud Accounting API ⬜

- [ ] internal/api/accounting.go
- [ ] internal/model/accounting.go
- [ ] cmd/accounting/offices (list)
- [ ] cmd/accounting/accounts (list)
- [ ] cmd/accounting/journals (list, show)
- [ ] cmd/accounting/partners (list)
- [ ] cmd/accounting/departments (list)
- [ ] cmd/accounting/taxes (list)

### Phase 3: Cloud Expense API ⬜

- [ ] internal/api/expense.go
- [ ] internal/model/expense.go
- [ ] cmd/expense/reports (list, show, approve, reject)
- [ ] cmd/expense/transactions (list, show)
- [ ] cmd/expense/categories (list)
- [ ] cmd/expense/departments (list)
- [ ] cmd/expense/projects (list)

### Phase 4: Cloud Invoice API ⬜

- [ ] internal/api/invoice.go
- [ ] internal/model/invoice.go
- [ ] cmd/invoice/billings (list, show, create, update, delete, download)
- [ ] cmd/invoice/partners (list, show)
- [ ] cmd/invoice/items (list)

### Phase 5: Cloud Payroll API ⬜

- [ ] internal/api/payroll.go (IP + API Identifier 認証)
- [ ] internal/model/payroll.go
- [ ] cmd/payroll/employees (list, show)
- [ ] cmd/payroll/payslips (list, show)
- [ ] cmd/payroll/attendances (list)

### Phase 6: Cloud Payable API ⬜

- [ ] internal/api/payable.go
- [ ] internal/model/payable.go
- [ ] cmd/payable/requests (list, show)
- [ ] cmd/payable/payments (list, show)
- [ ] cmd/payable/partners (list)

### Phase 7: リリース準備 ⬜

- [ ] README.md 作成
- [ ] goreleaser でクロスコンパイルビルド確認
- [ ] Homebrew tap 設定 (オプション)
- [ ] CI/CD (GitHub Actions)
- [ ] テストカバレッジ目標達成

---

## 6. API ドキュメント参照先

| サービス | ドキュメント URL |
|---------|-----------------|
| 共通仕様 | https://developers.biz.moneyforward.com/docs/common/api_common_specifications/ |
| Getting Started | https://developers.biz.moneyforward.com/docs/common/getting-started-moneyforward-cloud-apis/ |
| App Portal | https://app-portal.moneyforward.com |
| Cloud Expense | https://expense.moneyforward.com/api/index.html |
| Cloud Invoice | https://invoice.moneyforward.com/docs/api/v3/index.html |
| Cloud Payroll | https://payroll.moneyforward.com/api/v2/document |
| Cloud Payable | https://payable.moneyforward.com/api/index.html |
| Expense API GitHub | https://github.com/moneyforward/expense-api-doc |

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
