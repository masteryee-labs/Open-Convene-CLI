<div align="center">

# OpenConveneCLI

### 多模型 AI 協作 CLI 工具 — 透過原生 CLI 編排 N 個 AI 程式碼代理

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | **繁體中文** | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## 概覽

**OpenConveneCLI** 是一款開源 Go 命令列工具，實現了**多模型協作** — 將同一個 prompt 同時分派給 N 個回應模型（各自透過原生 CLI 以唯讀模式執行），彙整它們的回應產出統一結論，再交由執行模型根據彙整結果採取行動（撰寫程式碼、修改檔案或執行長跨度代理任務）。此方法構成了完整的 AI CLI 編排與 AI 程式碼生成流程。

此方法與 [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) 及 [OpenRouter Fusion](https://openrouter.ai/) 一致，但引入了一項關鍵創新：**CLI-as-Model** — 不要求統一 API，而是編排每個模型的原生 CLI（Devin、Grok、Codex、Antigravity、Cursor、Kimi、Hermes、Aider、OpenCode）。即使某個模型沒有公開 API，只要它有 CLI，就能參與多模型協作。

> **關鍵詞**: AI CLI 編排, 多模型協作, Mixture-of-Agents, MoA, AI 程式碼生成, multi-agent system, CLI-as-Model, AI coding agent, LLM orchestration, fan-out AI

---

## 目錄

- [安裝](#安裝)
- [快速開始](#快速開始)
- [運作原理](#運作原理)
- [支援的 AI CLI](#支援的-ai-cli)
- [命令](#命令)
- [互動式 REPL](#互動式-repl)
- [CLI 旗標](#cli-旗標)
- [為何選擇 Go](#為何選擇-go)
- [文件](#文件)
- [授權條款](#授權條款)

---

## 安裝

### 一鍵安裝（推薦）

**Linux / macOS：**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows（PowerShell）：**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### 使用 Go 安裝

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### 從源碼編譯

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> 先決條件：Go 1.24+

---

## 快速開始

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

> 回應模型可使用任何已安裝的 CLI：agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode（至少需要 1 個）。

### 更新

在 REPL 中輸入 `/update` 查看適合你平台的更新命令。或者再次執行安裝命令——它會用最新版本覆蓋舊的二進位檔。

---

## 運作原理

OpenConveneCLI 提供三種模式，對應真實開發者工作流程：

| 模式 | 命令 | 流程 | 是否執行？ | 典型使用情境 |
|------|---------|----------|-----------|-----------------|
| `ask` | `openconvene ask "..."` | N 個回應模型 → 彙整模型 → 印出結論 | 否 | 技術研究、方案比較 |
| `code`（預設） | `openconvene "..."` | N 個回應模型 → 彙整模型（可選）→ 執行模型撰寫程式碼 | 是 — 撰寫程式碼 | 實作功能、修復 bug |
| `agent` | `openconvene agent "..."` | N 個回應模型 → 彙整模型 → 執行模型代理 | 是 — 代理模式 | 複雜多步驟任務 |

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

## 支援的 AI CLI

OpenConveneCLI 開箱即支援 9 款 AI 程式碼代理 CLI。每個 CLI 連接各自的模型後端——OpenConveneCLI 本身不依賴任何雲端服務。至少需安裝 1 款 CLI 方可使用本工具。

| CLI | 說明 | 唯讀 | 執行模型 | 安裝命令 |
|-----|------|------|----------|----------|
| [Devin](https://devin.ai) | Cognition 的自主 AI 軟體工程師。全端 coding agent，具備 shell 存取、瀏覽器控制與長時程任務規劃能力。 | 視情況 | 是 | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | xAI 的 AI coding CLI，由 Grok 模型驅動。快速推理與程式碼生成，具備即時知識存取能力。 | 視情況 | 是 | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | OpenAI 的終端機 coding agent。沙箱執行——`--sandbox read-only` 用於安全研究，`workspace-write` 用於程式碼執行。 | 是 | 是 | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | Google 的 AI coding agent CLI，由 Gemini 驅動。支援多檔案編輯、程式碼審查與 agentic 任務執行（Gemini 2.5 Pro/Flash）。 | 視情況 | 是 | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | AI 優先的程式碼編輯器，具備 agent 模式。無 `--force` 時為唯讀分析；加 `--force` 時自主編輯檔案。由 Claude、GPT-4、Gemini 驅動。 | 是 | 是 | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Moonshot AI 的 coding CLI，由 Kimi K2 驅動。長上下文程式碼理解（256K tokens），唯讀操作自動批准。 | 是 | 是 | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | HashiCorp 的 AI agent CLI。`chat -q` 單次查詢模式；agentic 模式用於多步驟基礎設施與程式碼任務。 | 視情況 | 是 | `hermes setup --portal` |
| [Aider](https://aider.chat) | 開源 AI 配對程式設計工具。與 Git 整合，支援 GPT-4o、Claude 3.5、DeepSeek 及本地 LLM。編輯優先設計——預設會修改檔案。 | 否 | 是 | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | 開源 AI coding agent。`run` 子命令用於非互動單一 prompt；agentic 模式用於自主開發。支援多個 LLM 供應商。 | 視情況 | 是 | 請見 [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> **唯讀**欄表示該 CLI 能否安全地以回應模型模式運作（不修改檔案）。`是` = 強制唯讀、`視情況` = 非互動模式但可能觸發工具、`否` = 預設會修改檔案（僅作為執行模型）。

---

## 命令

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

## 互動式 REPL

執行 `openconvene`、`openconvene ask` 或 `openconvene agent` 而不帶任務參數時，會進入互動式 REPL，類似於 codex、grok、agy 與 devin。

在 REPL 中，您可以直接輸入 prompt，或使用斜線命令切換設定：

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

> **REPL 功能**：fish 風格選單補全（Tab 顯示補全選單，上/下方向鍵瀏覽候選項，Enter 確認，Shift-Tab 反向循環）、增量歷史搜尋（Ctrl-R/Ctrl-S）、跨工作階段命令歷史。由 [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4 提供。

### 斜線命令

| 命令 | 別名 | 說明 |
|---------|---------|-------------|
| `/help` | `/h`, `/?` | 顯示所有可用命令 |
| `/status` | | 顯示工作階段狀態（模式、模型、執行次數） |
| `/mode [ask\|code\|agent]` | | 顯示或切換當前模式 |
| `/models` | `/m` | 列出所有已設定的模型 |
| `/responders [a,b,c]` | | 顯示或設定回應模型 |
| `/executor [name]` | | 顯示或設定執行模型 |
| `/synthesizer [name]` | | 顯示或設定彙整模型（`none` 可清除） |
| `/language [lang]` | `/lang` | 顯示或設定模型回應語言 |
| `/usage` | `/u` | 顯示各 CLI 使用統計 |
| `/config` | `/c`, `/settings` | 顯示當前設定摘要 |
| `/detect` | `/d` | 偵測已安裝的 CLI |
| `/clear` | `/new` | 清除畫面並重置工作階段 |
| `/compact` | | （預留）摘要對話以釋放 token |
| `/resume` | `/continue` | （預留）恢復先前的工作階段 |
| `/update` | | （預留）檢查並安裝更新 |
| `/exit` | `/quit`, `/q` | 結束 REPL |

---

## CLI 旗標

| 旗標 | 說明 |
|------|-------------|
| `-p`, `--print` | 非互動式單次執行模式 |
| `-m`, `--model <name>` | 指定模型（`--executor` 的別名） |
| `--json` | JSON 輸出格式 |
| `--responders <a,b,c>` | 指定回應模型 |
| `--executor <name>` | 指定執行模型 |
| `--synthesizer <name>` | 指定彙整模型 |
| `--config <path>` | 指定設定檔路徑 |
| `--timeout <sec>` | 覆寫逾時設定 |
| `--verbose` | 顯示原始回應與詮釋資料 |
| `--language <lang>` | 設定模型回應語言 |
| `--` | 分隔符號（置於 prompt 之前） |

---

## 為何選擇 Go

- **單一靜態二進位檔** — 編譯產出零執行期相依；`curl + chmod` 即可運作
- **Goroutine 原生並行** — N 個回應模型 fan-out 並行執行，比 Python asyncio 更輕量
- **快速啟動** — 約 5ms 啟動時間，適合 CLI 使用
- **靜態型別** — 強型別結構體取代 map，重構更安全
- **跨平台** — `GOOS=windows/linux/darwin` 一鍵交叉編譯

---

## 文件

| 文件 | 內容 |
|----------|---------|
| [Overview](Docs/00-Overview.md) | 設計動機、與 Fusion/MoA 的比較 |
| [Architecture](Docs/01-Architecture.md) | 系統架構、Go 模組結構、資料流程 |
| [Usage Guide](Docs/02-Usage-Guide.md) | 完整使用指南（安裝、設定、旗標、模式） |
| [Model Adapters](Docs/03-Model-Adapters.md) | 9 款 CLI 介接卡設計、唯讀能力矩陣 |
| [Configuration](Docs/04-Configuration.md) | 完整 `models.yaml` 結構與範例 |
| [Examples](Docs/05-Examples.md) | 各模式的真實使用範例 |
| [Troubleshooting](Docs/06-Troubleshooting.md) | 常見問題與解決方案 |

---

## 授權條款

[MIT](LICENSE)
