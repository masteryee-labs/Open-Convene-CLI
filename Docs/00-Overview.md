# 00 — OpenConveneCLI 概覽

> **版本**：v1.0（S1 架構 Session 產出）
> **語言**：Go >= 1.22
> **Module**：`github.com/masteryee-labs/open-convene-cli`

---

## 1. 一段話描述

**OpenConveneCLI** 是一支獨立的 Go 命令列工具，實現「多模型協作」：將同一個問題同時派發給 N 個 responder 模型（各自透過其原生 CLI 以 read-only 模式回答），由 synthesizer 整合 N 份回應成一份結論，再交由 executor 根據整合結果執行（寫碼、改檔、或長時間 agent 任務）。它提供 `research`、`code`、`agent` 三種模式，讓使用者以單一命令協調多個 AI CLI（Devin、Grok、Codex、Antigravity、Cursor、Kimi Code、Hermes、Aider、OpenCode），獲得比單一模型更穩健、更全面的產出。

---

## 2. 設計動機

### 2.1 為什麼不用 Devin Skill？

| 面向 | Devin Skill | 獨立 Go CLI（OpenConveneCLI） |
|------|-------------|------------------------------|
| **模型範圍** | 只能調度 Devin 自身 | 可協調 9 種異質 CLI（Devin、Grok、Codex、agy、Cursor、Kimi、Hermes、Aider、OpenCode） |
| **執行環境** | 綁定 Devin 平台/額度 | 本機 subprocess，不依賴任何單一平台 |
| **可組合性** | Skill 是 Devin 內部概念，無法被其他工具 import | Go module，可被任何 Go 專案 import（`internal/` 封裝核心） |
| **離線/自架** | 必須連 Devin 雲端 | 各 CLI 各自連線；OpenConveneCLI 本身不需雲端 |
| **測試隔離** | 難以 mock Devin 內部 | Adapter 介面 + AdapterFactory 依賴注入，S6 可完整 mock |

**結論**：Devin Skill 只能解決「在 Devin 內部做事」，無法解決「跨 9 種 CLI 協作」。OpenConveneCLI 的核心價值是**異質模型編排**，這是任何單一平台 Skill 都做不到的。

### 2.2 為什麼是獨立 CLI（而非函式庫或腳本）？

- **使用者體驗**：目標使用者是開發者，希望一行命令就能協調多模型——`openconvene ask "如何設計 X?"`。也可不帶 task 進入互動式 REPL（`openconvene`）。函式庫需要寫 Go 程式才能用，門檻太高。
- **與既有 CLI 生態對齊**：9 個目標 CLI（codex、agy、aider…）全都是獨立 CLI。OpenConveneCLI 作為 CLI 來編排 CLI，互動模式一致。
- **可獨立部署**：單一靜態二進位（Go 編譯產物），無 runtime 依賴，`curl + chmod` 即可用。
- **跨平台**：Go 跨平台編譯，一份程式碼出 Windows / Linux / macOS 三平台二進位。

### 2.3 為什麼選 Go？

| 決策點 | Go 的優勢 |
|--------|----------|
| **並發** | goroutines + channels 原生支援 fan-out（N 個 responder 平行呼叫），比 Python asyncio 更輕量、更安全 |
| **subprocess** | `os/exec` + `context.WithTimeout` 是管理外部 CLI 進程的標準方案，跨平台一致 |
| **單一二進位** | 編譯成靜態二進位，無 runtime/依賴地獄。Python 需要 venv + pip，Node 需要 node_modules |
| **跨平台** | `GOOS=windows/linux/darwin` 一鍵交叉編譯，無需 CI 矩陣 |
| **CLI 生態** | cobra（spf13）是 Go 最成熟的 CLI 框架；yaml.v3 是標準 YAML 解析 |
| **效能** | 編譯型語言，啟動快（<50ms），適合 CLI 場景。Python 啟動 + import 已 200ms+ |
| **類型安全** | 強型別 struct（DefaultsConfig）取代 map[string]interface{}，重構時編譯器抓錯 |
| **內建工具** | `go test` / `go vet` / `go fmt` / `go build` 零配置即用 |

---

## 3. 與 OpenRouter Fusion / Mixture-of-Agents 的異同

### 3.1 概念來源

OpenConveneCLI 的設計概念對齊兩個來源：

- **OpenRouter Fusion**：OpenRouter 平台的多模型組合功能，將多個模型的輸出融合。
- **Mixture-of-Agents（MoA）**：arXiv:2406.04692 提出的架構——N 個 responder 模型各自回答，synthesizer 整合，可多層疊加。

### 3.2 相同點

| 概念 | OpenRouter Fusion | MoA | OpenConveneCLI |
|------|-------------------|-----|----------------|
| N 個 responder 平行回答 | ✓ | ✓ | ✓（goroutines fan-out） |
| synthesizer 整合多份回應 | ✓ | ✓ | ✓（可選；不指定則 executor 兼任） |
| 藉多模型多樣性提升品質 | ✓ | ✓ | ✓ |
| 分層 / 多輪疊加 | 部分 | ✓（multi-round） | 未來可擴展（v1 單層） |

### 3.3 差異點

| 面向 | OpenRouter Fusion | MoA（論文） | OpenConveneCLI |
|------|-------------------|------------|----------------|
| **模型存取方式** | OpenRouter 雲端 API（單一 endpoint） | 假設可呼叫任意 LLM API | **本機 CLI subprocess**——每個模型透過其原生 CLI（codex / agy / aider…）呼叫，不經過統一 API |
| **執行能力** | 純文字生成，不執行 | 純文字生成，不執行 | **executor 可執行**——agent 模式下 executor CLI 以 agentic 模式長時間執行（改檔、跑指令） |
| **read-only 安全性** | N/A（API 本來就 read-only） | N/A | **顯式 read_only 能力矩陣**——各 CLI 的 respond 模式是否真正 read-only 需標記（true/false/maybe） |
| **部署型態** | SaaS（需 API key + 網路） | 研究概念 | **獨立本機二進位**——無雲端依賴，各 CLI 各自連線 |
| **配置粒度** | 平台預設 | 實驗設定 | **models.yaml**——每個 CLI 的命令模板、timeout、read_only、executor_capable 全可配置 |
| **目標場景** | 通用 LLM 呼叫 | 學術基準 | **開發者工作流**——research / code / agent 三模式對應實際開發需求 |

### 3.4 核心創新

OpenConveneCLI 相對於 Fusion / MoA 的獨特之處：

1. **CLI-as-Model**：不依賴統一 API，而是編排各模型的原生 CLI。這意味著即使某模型沒有公開 API（如某些 CLI-only 工具），只要它有 CLI 就能參與協作。
2. **read-only 分層**：明確區分 respond（read-only，只回答）與 execute（可執行工具）。responder 只做 read-only，executor 才執行。這讓多模型「腦力激盪」與「動手執行」安全分離。
3. **三模式工作流**：research（只研究不執行）、code（寫碼改檔）、agent（長時間 agent 任務），對應開發者真實需求，而非學術基準。
4. **跨平台本機部署**：單一 Go 二進位，Windows / Linux / macOS 通吃，無雲端依賴。

---

## 4. 三種模式速覽

| 模式 | 指令 | 流程 | 執行？ | 典型用途 |
|------|------|------|--------|---------|
| `research`（ask） | `openconvene ask "task"` | N responder → synthesizer → **印出結論** | ✗ 不執行 | 技術調研、方案比較、腦力激盪 |
| `code`（預設） | `openconvene "task"` | N responder → synthesizer（可選）→ **executor 寫碼/改檔** | ✓ 寫碼 | 實作功能、修 bug、重構 |
| `agent` | `openconvene agent "task"` | N responder 出策略 → synthesizer 整合 → **executor agent 長時間執行** | ✓ agent | 複雜多步任務、自動化管線 |

> 三種模式不帶 task 參數時均進入互動式 REPL：`openconvene`、`openconvene ask`、`openconvene agent`。
> synthesizer 為可選：不指定時，executor 兼任 synthesizer（直接讀 N 份回應後執行）。

---

## 5. 文件索引

| 文件 | 內容 |
|------|------|
| `00-Overview.md` | 本文——概覽、動機、與 Fusion/MoA 比較 |
| `01-Architecture.md` | 系統架構圖、Go module 結構、資料流、adapter/convene 介面、CLI 介面、config schema、跨平台說明 |
| `02-Usage-Guide.md` | 安裝、初始設定、基本用法、互動式 REPL 模式、完整參數參考、模式說明、Config 說明 |
| `03-Model-Adapters.md` | 9 個 CLI adapter 設計、read_only 能力矩陣、各 CLI 呼叫方式與限制 |
| `04-Configuration.md` | models.yaml 完整 schema + 範例 config |
| `05-Examples.md` | 各模式實際使用範例（ask / code / agent / stdin / verbose / timeout） |
| `06-Troubleshooting.md` | 常見問題、config 疑難排解、adapter 問題、警告說明、Go 編譯問題 |

> `Docs/Agents/` 存放設計 Session 的 Agent prompt，非 OpenConveneCLI 使用者文件。
