# OpenConveneCLI

**OpenConveneCLI** 是一支獨立的 Go 命令列工具，實現「多模型協作」：將同一個問題同時派發給 N 個 responder 模型（各自透過其原生 CLI 以 read-only 模式回答），由 synthesizer 整合 N 份回應成一份結論，再交由 executor 根據整合結果執行（寫碼、改檔、或長時間 agent 任務）。

## Quick Start

```bash
# 1. 安裝
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest

# 2. 偵測系統已安裝的 9 個 CLI adapter
openconvene detect

# 3. 產生 config
openconvene init --path ~/.config/openconvene/models.yaml

# 4. 執行多模型協作
openconvene ask "你的問題" --responders agy,grok

# 5. 寫碼（預設 code mode）
openconvene "fix the bug in foo.go"

# 6. agent 任務
openconvene agent "deploy the app"
```

> responders 可用任何已安裝的 CLI：agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode（至少裝 1 個）。

## What it does

OpenConveneCLI 提供三種模式，對應開發者真實需求：

| 模式 | 指令 | 流程 | 執行？ | 典型用途 |
|------|------|------|--------|---------|
| `ask` | `openconvene ask "..."` | N responder → synthesizer → 印出結論 | ✗ 不執行 | 技術調研、方案比較 |
| `code` | `openconvene "..."` (預設) | N responder → synthesizer（可選）→ executor 寫碼 | ✓ 寫碼 | 實作功能、修 bug |
| `agent` | `openconvene agent "..."` | N responder → synthesizer → executor agent | ✓ agent | 複雜多步任務 |

核心概念對齊 [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) 與 OpenRouter Fusion，但創新之處在於**CLI-as-Model**——不依賴統一 API，而是編排各模型的原生 CLI（Devin、Grok、Codex、agy、Cursor、Kimi、Hermes、Aider、OpenCode），即使某模型沒有公開 API，只要有 CLI 就能參與協作。

## Commands

```
# 單次執行（帶 task 參數）
openconvene "task"              # 預設 code mode（寫碼）
openconvene ask "task"          # ask mode（研究，不執行）
openconvene agent "task"        # agent mode（廣義 agentic 動作）

# 互動模式（不帶 task，進入 REPL）
openconvene                     # 進入互動 REPL（預設 code mode）
openconvene ask                 # 進入互動 REPL（ask mode）
openconvene agent               # 進入互動 REPL（agent mode）

# 輔助指令
openconvene models              # 列出已設定的模型
openconvene detect              # 偵測系統已安裝的 CLI
openconvene init                # 生成 starter models.yaml
openconvene check               # 驗證 models.yaml
```

進階 flags：`--responders`、`--executor`/`--model`/`-m`、`--synthesizer`、`--config`、`--timeout`、`--verbose`、`--json`、`-p`

## Interactive REPL

不帶 task 參數執行 `openconvene`、`openconvene ask` 或 `openconvene agent` 時，會進入互動式 REPL，類似 codex、grok、agy、devin 的互動模式。

在 REPL 中可以直接輸入提示詞執行，或使用 slash 指令切換設定（對齊 Devin/Codex/agy/Grok 慣例）：

```
openconvene(code)> fix the bug in main.go     # 直接下提示詞
openconvene(code)> /mode ask                  # 切換到 ask 模式
openconvene(ask)> /model devin                # 切換 executor 模型（同 /executor）
openconvene(ask)> /responders agy,grok,codex  # 切換 responders
openconvene(ask)> /synthesizer grok           # 切換 synthesizer
openconvene(ask)> /status                     # 查看當前 session 狀態
openconvene(ask)> /usage                      # 查看本次 session 各 CLI 使用量
openconvene(ask)> /models                     # 列出已設定的模型
openconvene(ask)> /detect                     # 偵測已安裝的 CLI
openconvene(ask)> /config                     # 顯示當前設定
openconvene(ask)> /new                        # 清除 session 重新開始
openconvene(ask)> /help                       # 顯示所有指令
openconvene(ask)> /exit                       # 離開 REPL
```

### Slash 指令一覽

| 指令 | 簡寫/別名 | 說明 | 對齊 |
|------|----------|------|------|
| `/help` | `/h`, `/?` | 顯示所有可用指令 | Devin, Codex, agy |
| `/status` | | 顯示 session 狀態（模式、模型、執行數） | Codex |
| `/mode [ask\|code\|agent]` | | 顯示或切換當前模式 | Devin, Codex |
| `/model [name]` | | 顯示或切換 executor 模型 | Devin, Codex, agy |
| `/models` | `/m` | 列出已設定的模型 | OpenConvene 獨有 |
| `/responders [a,b,c]` | | 顯示或設定 responders | OpenConvene 獨有 |
| `/executor [name]` | | 顯示或設定 executor | OpenConvene 獨有 |
| `/synthesizer [name]` | | 顯示或設定 synthesizer | OpenConvene 獨有 |
| `/usage` | `/u` | 顯示各 CLI 使用量統計 | agy |
| `/config` | `/c`, `/settings` | 顯示當前設定摘要 | agy |
| `/detect` | `/d` | 偵測已安裝的 CLI | OpenConvene 獨有 |
| `/clear` | `/new` | 清除螢幕並重置 session | Devin, Codex |
| `/compact` | | (stub) 壓縮對話以釋放 token | Devin, Codex |
| `/resume` | `/continue` | (stub) 恢復之前的 session | Devin, agy |
| `/update` | | (stub) 檢查並安裝更新 | Devin |
| `/exit` | `/quit`, `/q` | 離開 REPL | Devin, agy |

### CLI Flags（進入前 `--` 指令）

| Flag | 說明 | 對齊 |
|------|------|------|
| `-p`, `--print` | 單輪模式（非互動） | Devin, agy, Grok |
| `-m`, `--model <name>` | 指定模型（`--executor` 的別名） | Codex, agy, Grok |
| `--json` | JSON 輸出格式 | Grok (`--output-format json`) |
| `--responders <a,b,c>` | 指定 responders | OpenConvene 獨有 |
| `--executor <name>` | 指定 executor | OpenConvene 獨有 |
| `--synthesizer <name>` | 指定 synthesizer | OpenConvene 獨有 |
| `--config <path>` | 指定 config 路徑 | OpenConvene 獨有 |
| `--timeout <sec>` | 覆寫 timeout | OpenConvene 獨有 |
| `--verbose` | 顯示原始回應和 metadata | OpenConvene 獨有 |
| `--` | 分隔符（提示詞前加 `--`） | Devin |

## Why Go

- **單一靜態二進位**：編譯產出無 runtime 依賴，`curl + chmod` 即可用
- **goroutines 天生並行**：N 個 responder 平行 fan-out，比 Python asyncio 更輕量
- **快速啟動**：~5ms 啟動，適合 CLI 場景
- **靜態型別**：強型別 struct 取代 map，重構安全
- **跨平台**：`GOOS=windows/linux/darwin` 一鍵交叉編譯

## Documentation

- [Overview](Docs/00-Overview.md) — 概覽、設計動機、與 Fusion/MoA 比較
- [Architecture](Docs/01-Architecture.md) — 系統架構圖、Go module 結構、資料流
- [Usage Guide](Docs/02-Usage-Guide.md) — 完整使用指南（安裝、設定、參數、模式）
- [Model Adapters](Docs/03-Model-Adapters.md) — 9 個 CLI adapter 設計、read_only 能力矩陣
- [Configuration](Docs/04-Configuration.md) — models.yaml 完整 schema + 範例
- [Examples](Docs/05-Examples.md) — 各模式使用範例
- [Troubleshooting](Docs/06-Troubleshooting.md) — 常見問題與解決方案

## Build from Source

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> 前置條件：Go 1.22+

## License

MIT
