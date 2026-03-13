# planitai-moneyforward-cli

## 개요

Money Forward クラウド の公開 API を CLI で提供するプロジェクト。
`aws`, `gh`, `conoha` CLI のように単一バイナリで動作し、Agent-friendly な設計を目指す。

- **言語**: Go
- **CLI フレームワーク**: cobra (spf13/cobra)
- **参考実装**: `~/dev/crowdy/conoha-cli`
- **ライセンス**: MIT

## アーキテクチャ方針

### SDK 不使用

Money Forward は公式 SDK を提供していない。非公式 SDK は API 更新への追従が遅れるリスクがあるため、
**REST API を直接呼び出す方式** を採用する。OpenAPI/Swagger ドキュメントを参照して実装する。

### Agent-Friendly 設計

- `--format json|yaml|csv|table` による構造化出力
- `--no-input` で非対話モード
- stdout に結果、stderr にエラー・ログ
- 明確な終了コード (0=成功, 1=一般エラー, 2=認証エラー, ...)
- `--quiet` で不要な出力の抑制

### 認証: OAuth2 + マルチプロファイル

- OAuth 2.0 Authorization Code Grant (Money Forward Cloud 共通)
- `gh auth login` / `conoha auth login` と同様のフロー
- 複数アカウント (プロファイル) 対応: `auth list`, `auth switch`
- トークンキャッシュ + 自動リフレッシュ (refresh_token: 540日有効)
- 設定ファイル: `~/.config/mf/` 配下 (config.yaml, tokens.yaml)

## ドキュメント体系

| ファイル | 役割 |
|---------|------|
| `CLAUDE.md` | AI コンテキスト (プロジェクト概要、アーキテクチャ方針) |
| `DESIGN.md` | CLI 設計 (コマンド・フラグ・フェーズ・ADR) |
| `SPEC.md` | API 仕様参照 (エンドポイント、データモデル、enum、認証詳細) |

## ディレクトリ構成

```
.
├── main.go
├── go.mod
├── Makefile
├── .goreleaser.yaml
├── CLAUDE.md
├── DESIGN.md                    # 設計ドキュメント
├── SPEC.md                      # API 仕様書
├── ref/                         # 参考実装 (mf-invoice-mcp, admina-mcp-server)
├── cmd/
│   ├── root.go                  # ルートコマンド
│   ├── version.go
│   ├── completion.go
│   ├── cmdutil/
│   │   ├── args.go
│   │   └── client.go
│   ├── auth/                    # auth login|logout|status|list|switch|remove
│   ├── config/                  # config get|set|list
│   ├── invoice/                 # invoice [subcommand] (Phase 2)
│   ├── expense/                 # expense [subcommand] (Phase 3)
│   ├── payable/                 # payable [subcommand] (Phase 4)
│   ├── payroll/                 # payroll [subcommand] (Phase 5)
│   └── admina/                  # admina [subcommand] (Phase 6)
├── internal/
│   ├── api/
│   │   ├── client.go            # HTTP クライアント (リトライ, デバッグ)
│   │   ├── oauth.go             # OAuth2 フロー
│   │   ├── invoice.go
│   │   ├── expense.go
│   │   ├── payable.go
│   │   ├── payroll.go
│   │   └── admina.go            # Admina API (API Key 認証)
│   ├── config/
│   │   ├── config.go
│   │   └── tokens.go
│   ├── model/                   # API レスポンス構造体
│   ├── output/                  # table/json/yaml/csv フォーマッタ
│   ├── errors/                  # 型付きエラー + 終了コード
│   └── prompt/                  # 対話プロンプト
└── test/
```

## Money Forward API 情報

### 認証エンドポイント

| 項目 | URL |
|------|-----|
| Authorization | `https://api.biz.moneyforward.com/authorize` |
| Token | `https://api.biz.moneyforward.com/token` |
| App Portal | `https://app-portal.moneyforward.com` |

### サービス別 API

| サービス | Base URL | バージョン | 公開状態 | 認証 |
|---------|----------|-----------|---------|------|
| Cloud Invoice | `https://invoice.moneyforward.com/api/v3/` | v3.1.0 | **公開** | OAuth2 |
| Cloud Expense | `https://expense.moneyforward.com/api/external/v1/` | v1 | **公開** | OAuth2 |
| Cloud Payable | `https://payable.moneyforward.com/api/external/v1/` | v1 | **公開** | OAuth2 |
| Cloud Payroll | `https://payroll.moneyforward.com/api/v2/` | v2 | **公開** | IP + API ID |
| Admina | `https://api.itmc.i.moneyforward.com/api/v1` | v1 | **公開** | API Key |
| Cloud Accounting | パートナー限定 | v2 | **非公開** | — |

### 注意事項

- **Accounting API** は非公開 (クローズドAPI) — 一般開発者は利用不可、パートナー申請後に検討
- **Payroll API** は OAuth2 ではなく IP ホワイトリスト + API Identifier 認証
- **MF Kessai** は別会社・別プロダクト (対象外)
- **ME (個人向け)** はクローズド API (パートナー限定、対象外)
- 各サービスの OAuth スコープは異なる → サービスごとにスコープ管理が必要

## 開発コマンド

```bash
make build       # バイナリビルド
make test        # テスト実行
make lint        # リント
make install     # ローカルインストール
```

## 進捗管理

→ `DESIGN.md` の進捗チェックリストを参照
