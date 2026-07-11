<div align="center">

# OpenConveneCLI

### マルチモデル AI 協調 CLI ツール — ネイティブ CLI で N 個の AI コーディングエージェントをオーケストレーション

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | **日本語** | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## 概要

**OpenConveneCLI** は、マルチモデル AI 協調を実現するオープンソースの Go 製コマンドラインツールです。同じプロンプトを N 個のレスポンダーモデルに同時にディスパッチし（各モデルはネイティブ CLI を読み取り専用モードで使用）、それらの応答を統一的な結論へと統合した上で、エグゼキューターモデルに委譲します。エグゼキューターは統合された結果に基づいて行動します（コードの記述、ファイルの変更、または長期的なエージェントタスクの実行）。このアプローチにより、AI CLI オーケストレーションと AI コード生成をシームレスに統合し、Mixture-of-Agents（MoA）パターンの利点を CLI ベースのワークフローにもたらします。

本アプローチは [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) や [OpenRouter Fusion](https://openrouter.ai/) と方向性が一致していますが、重要な革新を導入しています。それは **CLI-as-Model** です。統一された API を要求するのではなく、各モデルのネイティブ CLI（Devin、Grok、Codex、Antigravity、Cursor、Kimi、Hermes、Aider、OpenCode）をオーケストレーションします。あるモデルに公開 API がなくても、CLI さえあればマルチモデル AI 協調に参加できます。

> **キーワード**: AI CLI オーケストレーション、マルチモデル AI 協調、Mixture-of-Agents、MoA、AI コード生成、マルチエージェントシステム、CLI-as-Model、AI コーディングエージェント、LLM オーケストレーション、ファンアウト AI

---

## 目次

- [インストール](#インストール)
- [クイックスタート](#クイックスタート)
- [仕組み](#仕組み)
- [対応 AI CLI](#対応-ai-cli)
- [コマンド](#コマンド)
- [インタラクティブ REPL](#インタラクティブ-repl)
- [CLI フラグ](#cli-フラグ)
- [なぜ Go なのか](#なぜ-go-なのか)
- [ドキュメント](#ドキュメント)
- [ライセンス](#ライセンス)

---

## インストール

### ワンラインインストール（推奨）

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### Go でインストール

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### ソースからビルド

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> 前提条件：Go 1.24+

---

## クイックスタート

```bash
# 1. Detect installed AI CLIs
openconvene detect

# 2. Generate config
openconvene init --path ~/.config/openconvene/models.yaml

# 3. Run multi-model collaboration
openconvene ask "your question" --responders agy,grok

# 4. Write code (default code mode)
openconvene "fix the bug in foo.go"

# 5. Agent task
openconvene agent "deploy the app"
```

> レスポンダーにはインストール済みの任意の CLI を使用できます：agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode（最低 1 つ必要）。

### 更新

REPLで `/update` と入力すると、お使いのプラットフォーム用の更新コマンドが表示されます。またはインストールコマンドを再実行してください——最新バージョンで古いバイナリが上書きされます。

---

## 仕組み

OpenConveneCLI は、実際の開発者ワークフローに合わせた 3 つのモードを提供します：

| モード | コマンド | パイプライン | 実行？ | 典型的なユースケース |
|------|---------|----------|-----------|-----------------|
| `ask` | `openconvene ask "..."` | N レスポンダー → シンセサイザー → 結論を表示 | いいえ | 技術調査、解決策の比較 |
| `code`（デフォルト） | `openconvene "..."` | N レスポンダー → シンセサイザー（任意） → エグゼキューターがコードを記述 | はい — コードを記述 | 機能の実装、バグ修正 |
| `agent` | `openconvene agent "..."` | N レスポンダー → シンセサイザー → エグゼキューターエージェント | はい — エージェントモード | 複雑なマルチステップタスク |

```
                    ┌──────────┐
                    │  Prompt  │
                    └────┬─────┘
                         │ fan-out
            ┌────────────┼────────────┐
            ▼            ▼            ▼
       ┌────────┐  ┌────────┐  ┌────────┐
       │Responder│  │Responder│  │Responder│
       │  (agy) │  │ (grok) │  │ (codex)│
       └───┬────┘  └───┬────┘  └───┬────┘
           │           │           │
           └───────────┼───────────┘
                       ▼
                ┌─────────────┐
                │ Synthesizer │
                └──────┬──────┘
                       ▼
                ┌──────────┐
                │ Executor │
                └──────────┘
```

---

## 対応 AI CLI

OpenConveneCLI は 9 つの AI コーディングエージェント CLI に標準で対応しています。各 CLI は独自のモデルバックエンドに接続します——OpenConveneCLI 自体はクラウドサービスに依存しません。本ツールを使用するには、少なくとも 1 つの CLI がインストールされている必要があります。

| CLI | 説明 | 読み取り専用 | エグゼキューター | インストール |
|-----|-------------|-----------|----------|---------|
| [Devin](https://devin.ai) | Cognition 社の自律型 AI ソフトウェアエンジニア。シェルアクセス、ブラウザ制御、長時間タスク計画を備えたフルスタックコーディングエージェント。 | 部分的 | はい | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | xAI 社の AI コーディング CLI（Grok モデル搭載）。高速推論とコード生成、リアルタイム知識アクセス。 | 部分的 | はい | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | OpenAI 社のターミナルベースコーディングエージェント。サンドボックス実行——`--sandbox read-only` で安全なリサーチ、`workspace-write` でコード実行。 | はい | はい | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | Google 社の AI コーディングエージェント CLI（Gemini 搭載）。複数ファイル編集、コードレビュー、エージェント型タスク実行（Gemini 2.5 Pro/Flash）。 | 部分的 | はい | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | AI ファーストのコードエディタ（エージェントモード搭載）。`--force` なしで読み取り専用分析、`--force` ありで自律ファイル編集。Claude、GPT-4、Gemini 搭載。 | はい | はい | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Moonshot AI 社のコーディング CLI（Kimi K2 搭載）。長文脈コード理解（256K トークン）、読み取り専用操作は自動承認。 | はい | はい | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | HashiCorp 社の AI エージェント CLI。`chat -q` で単一クエリモード、エージェントモードでマルチステップインフラ・コードタスク。 | 部分的 | はい | `hermes setup --portal` |
| [Aider](https://aider.chat) | オープンソース AI ペアプログラミングツール。Git 統合、GPT-4o、Claude 3.5、DeepSeek、ローカル LLM 対応。編集ファースト設計——デフォルトでファイルを変更。 | いいえ | はい | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | オープンソース AI コーディングエージェント。`run` サブコマンドで非対話単一プロンプト、エージェントモードで自律開発。複数 LLM プロバイダー対応。 | 部分的 | はい | [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) を参照 |

> **読み取り専用**は、CLI がレスポンダーモード（ファイル変更なし）で安全に動作できるかを示します。`はい` = 読み取り専用が強制、`部分的` = 非対話モードだがツールを起動する可能性あり、`いいえ` = デフォルトでファイルを変更（エグゼキューター専用）。

---

## コマンド

```bash
# Single-shot (with task argument)
openconvene "task"              # default code mode (writes code)
openconvene ask "task"          # ask mode (research, no execution)
openconvene agent "task"        # agent mode (agentic actions)

# Interactive mode (no task argument → enters REPL)
openconvene                     # interactive REPL (default code mode)
openconvene ask                 # interactive REPL (ask mode)
openconvene agent               # interactive REPL (agent mode)

# Utility commands
openconvene models              # list configured models
openconvene detect              # detect installed AI CLIs
openconvene init                # generate starter models.yaml
openconvene check               # validate models.yaml
```

---

## インタラクティブ REPL

`openconvene`、`openconvene ask`、または `openconvene agent` をタスク引数なしで実行すると、codex、grok、agy、devin と同様のインタラクティブ REPL に入ります。

REPL 内ではプロンプトを直接入力できるほか、スラッシュコマンドで設定を切り替えることができます：

```
openconvene(code)> fix the bug in main.go     # direct prompt
openconvene(code)> /mode ask                  # switch to ask mode
openconvene(ask)> /executor devin             # switch executor model
openconvene(ask)> /responders agy,grok,codex  # switch responders
openconvene(ask)> /synthesizer grok           # switch synthesizer
openconvene(ask)> /language zh-TW             # set model response language
openconvene(ask)> /status                     # view session status
openconvene(ask)> /usage                      # view per-CLI usage stats
openconvene(ask)> /models                     # list configured models
openconvene(ask)> /detect                     # detect installed CLIs
openconvene(ask)> /config                     # show current config
openconvene(ask)> /new                        # clear session
openconvene(ask)> /help                       # show all commands
openconvene(ask)> /exit                       # exit REPL
```

> **REPL 機能**: fish 風のメニューコンプリート（Tab で補完メニューを表示、上下矢印キーで候補を移動、Enter で確定、Shift-Tab で逆順に巡回）、インクリメンタル履歴検索（Ctrl-R/Ctrl-S）、セッションをまたぐコマンド履歴。[`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4 により提供されています。

### スラッシュコマンド

| コマンド | エイリアス | 説明 |
|---------|---------|-------------|
| `/help` | `/h`, `/?` | 利用可能なすべてのコマンドを表示 |
| `/status` | | セッションステータスを表示（モード、モデル、実行回数） |
| `/mode [ask\|code\|agent]` | | 現在のモードを表示または切り替え |
| `/models` | `/m` | 設定済みのすべてのモデルを一覧表示 |
| `/responders [a,b,c]` | | レスポンダーを表示または設定 |
| `/executor [name]` | | エグゼキューターを表示または設定 |
| `/synthesizer [name]` | | シンセサイザーを表示または設定（`none` でクリア） |
| `/language [lang]` | `/lang` | モデルの応答言語を表示または設定 |
| `/usage` | `/u` | CLI ごとの使用統計を表示 |
| `/config` | `/c`, `/settings` | 現在の設定サマリーを表示 |
| `/detect` | `/d` | インストール済み CLI を検出 |
| `/clear` | `/new` | 画面をクリアしてセッションをリセット |
| `/compact` | | （スタブ）会話を要約してトークンを解放 |
| `/resume` | `/continue` | （スタブ）以前のセッションを再開 |
| `/update` | | （スタブ）アップデートの確認とインストール |
| `/exit` | `/quit`, `/q` | REPL を終了 |

---

## CLI フラグ

| フラグ | 説明 |
|------|-------------|
| `-p`, `--print` | 非インタラクティブなワンショットモード |
| `-m`, `--model <name>` | モデルを指定（`--executor` のエイリアス） |
| `--json` | JSON 出力形式 |
| `--responders <a,b,c>` | レスポンダーを指定 |
| `--executor <name>` | エグゼキューターを指定 |
| `--synthesizer <name>` | シンセサイザーを指定 |
| `--config <path>` | 設定ファイルのパスを指定 |
| `--timeout <sec>` | タイムアウトを上書き |
| `--verbose` | 生の応答とメタデータを表示 |
| `--language <lang>` | モデルの応答言語を設定 |
| `--` | セパレーター（プロンプトの前に追加） |

---

## なぜ Go なのか

- **単一の静的バイナリ** — コンパイル済みの出力はランタイム依存関係がゼロ。`curl + chmod` で動作します
- **Goroutines によるネイティブ並行処理** — N 個のレスポンダーが並列でファンアウトし、Python asyncio よりも軽量です
- **高速な起動** — 約 5ms で起動し、CLI 利用に最適です
- **静的型付け** — 厳密に型付けされた構造体がマップに代わり、リファクタリングが安全です
- **クロスプラットフォーム** — `GOOS=windows/linux/darwin` でワンコマンドのクロスコンパイルが可能です

---

## ドキュメント

| ドキュメント | 内容 |
|----------|---------|
| [Overview](Docs/00-Overview.md) | 設計の動機、Fusion/MoA との比較 |
| [Architecture](Docs/01-Architecture.md) | システムアーキテクチャ、Go モジュール構造、データフロー |
| [Usage Guide](Docs/02-Usage-Guide.md) | 完全な利用ガイド（インストール、設定、フラグ、モード） |
| [Model Adapters](Docs/03-Model-Adapters.md) | 9 つの CLI アダプターの設計、読み取り専用機能マトリックス |
| [Configuration](Docs/04-Configuration.md) | 完全な `models.yaml` スキーマと例 |
| [Examples](Docs/05-Examples.md) | 各モードの実践的な利用例 |
| [Troubleshooting](Docs/06-Troubleshooting.md) | 一般的な問題と解決策 |

---

## ライセンス

[MIT](LICENSE)
