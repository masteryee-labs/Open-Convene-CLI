# OpenConveneCLI — 使用指南

> **版本**：v1.0（S7 User Docs Session 產出）
> **Module**：`github.com/masteryee-labs/open-convene-cli`
> **語言**：Go >= 1.24

---

## 安裝

### 前置條件

- **Go 1.24+**（若從源碼安裝或編譯）
- **至少 1 個 AI CLI 工具已安裝**——OpenConveneCLI 支援以下 9 個可選 CLI，你至少需要安裝其中一個：

| CLI | 安裝指令（僅供參考） |
|-----|---------------------|
| Devin | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| Grok | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| Codex | `npm install -g @openai/codex` |
| Antigravity (agy) | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| Cursor | `curl https://cursor.com/install -fsS \| bash` |
| Kimi Code | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| Hermes | `hermes setup --portal` |
| Aider | `python -m pip install aider-install && aider-install` |
| OpenCode | 見 https://opencode.ai/docs/cli/ |

> 這些 CLI 各自連線到各自的模型後端，OpenConveneCLI 本身不依賴任何雲端服務。

### 方式一：從源碼安裝（推薦）

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go install ./cmd/openconvene
```

### 方式二：直接 go install

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### 方式三：從 release 下載預編譯二進位

從 GitHub Releases 頁面下載對應平台的二進位檔，放入你的 `PATH` 即可。Go 編譯為靜態二進位，無 runtime 依賴。

### 驗證安裝

```bash
openconvene --help
```

若看到 root command 說明文字，表示安裝成功。

> **找不到命令？** 確認 `GOPATH/bin` 在你的 `PATH` 中：
> - **Linux/macOS**：`export PATH=$PATH:$(go env GOPATH)/bin`
> - **Windows**：`%USERPROFILE%\go\bin`

---

## 初始設定

### 步驟 1：偵測系統已安裝的 CLI

```bash
openconvene detect
```

此命令**不需要 config 檔**，直接掃描 `PATH` 偵測 9 個 CLI 的安裝狀態。輸出包含：

- 各 CLI 的安裝狀態（已安裝 / 未安裝）
- 完整路徑（若已安裝）
- read_only 能力（`true` / `false` / `maybe`）
- 是否適合當 responder 或 executor
- 未安裝 CLI 的安裝指令（僅供參考，不自動安裝）

範例輸出：

```
CLI          INSTALLED  PATH                                      READ_ONLY  CAN_RESPOND  CAN_EXECUTE
------------ ---------- ---------------------------------------- ---------- ------------ ------------
agy          yes        /usr/local/bin/agy                       maybe      yes          yes
aider        no         -                                        false      no           yes
codex        yes        /usr/local/bin/codex                     true       yes          yes
cursor       no         -                                        true       yes          yes
devin        yes        /usr/local/bin/devin                     maybe      yes          yes
grok         yes        /usr/local/bin/grok                      maybe      yes          yes
hermes       no         -                                        maybe      yes          yes
kimi         no         -                                        true       yes          yes
opencode     no         -                                        maybe      yes          yes

Installed: 4 / 9
Available responders: agy, codex, devin, grok
Available executors:  agy, codex, devin, grok

Missing (install to use):
  aider         python -m pip install aider-install && aider-install
  cursor        curl https://cursor.com/install -fsS | bash
  ...
```

> 偵測結果用來決定 `models.yaml` 要設定哪些模型。你只需配置已安裝的 CLI。

> **捷徑**：如果你只想快速開始，可以跳過步驟 2-5，直接執行 `openconvene` 或 `openconvene ask`。若系統中沒有 `models.yaml`，CLI 會自動生成一份預設 config（使用 `devin:glm-5.2`、`devin:swe-1.7`、`devin:kimi-k2.7` 三個動態模型名），寫到 `~/.config/openconvene/models.yaml`，然後直接進入 REPL。

### 步驟 2：產生範例 config

```bash
openconvene init --path ~/.config/openconvene/models.yaml
```

此命令會在指定路徑產生一份包含全部 9 個 adapter 的範例 `models.yaml`。若檔案已存在則拒絕覆蓋。

### 步驟 3：編輯 models.yaml

用編輯器開啟 `models.yaml`，根據 `detect` 結果調整：

- 確認各 CLI 的 `command` 和 `execute_command` 模板正確
- 確認 `read_only` 值與實際 CLI 行為一致（參考 `detect` 輸出）
- 設定 `defaults` 中的 `responders`、`executor`、`synthesizer`、`timeout`

> 詳細欄位說明見 [Config 說明](#config-說明) 章節及 [04-Configuration.md](04-Configuration.md)。

### 步驟 4：驗證 config

```bash
openconvene check
```

確認 config 語法正確、參照完整性通過。若有 ERROR 級問題會中止並列出；WARNING 級問題僅提示不阻擋。

### 步驟 5：確認模型列表

```bash
openconvene models
```

列出 config 中所有已配置的模型，並顯示安裝狀態、read_only 能力、executor 能力。

---

## 基本用法

### Ask 模式（研究分析）

```bash
openconvene ask "你的問題" --responders agy,grok
```

N 個 responder 平行回答問題，synthesizer（若有設定）整合回應，最後印出結論。**不執行任何工具、不改檔**。

### Code 模式（寫碼改檔，預設模式）

```bash
openconvene "你的寫碼任務" --responders agy,grok --executor codex
```

N 個 responder 平行提供建議，synthesizer（可選）整合，executor 根據整合結果實際寫碼改檔。code 為預設模式，不需指定子命令。

### Agent 模式（Agent 任務）

```bash
openconvene agent "你的 agent 任務" --responders agy,grok --executor devin --synthesizer agy
```

N 個 responder 平行提供調查策略，synthesizer 整合，executor 以 agentic 模式長時間執行（研究、檔案操作、多步驟工作流）。

### 從 stdin 讀取 task

```bash
echo "分析這段程式碼的效能瓶頸" | openconvene ask - --responders agy
```

位置參數值為 `-` 時從 stdin 讀取任務描述，適合 pipe 管線使用。

---

## Interactive REPL Mode

When you run `openconvene`, `openconvene ask`, or `openconvene agent` WITHOUT a task argument, the CLI enters an interactive REPL (Read-Eval-Print Loop), similar to codex, grok, agy, and devin.

### Entering the REPL

```bash
openconvene           # enters REPL in code mode (default)
openconvene ask       # enters REPL in ask mode
openconvene agent     # enters REPL in agent mode
```

### Using the REPL

In the REPL, you can:
1. Type any text → it runs as a prompt through the Convene pipeline in the current mode
2. Use slash commands (starting with `/`) to inspect state, switch modes, change models, and view usage
3. Press **Up/Down arrows** to browse command history (persists across sessions)
4. Press **Tab** after typing `/` to open a fish-style completion menu — use **Up/Down arrows** to navigate candidates, **Enter** to confirm, **Shift-Tab** to cycle backward
5. Press **Ctrl-R** for reverse incremental history search

```
openconvene(code)> fix the bug in main.go     # direct prompt
openconvene(code)> /mode ask                  # switch to ask mode
openconvene(ask)> /responders agy:Gemini 3.5 Flash (High),grok:grok-4.5  # 動態模型名
openconvene(ask)> /executor devin:glm-5.2     # 切換 executor（可用動態模型名）
openconvene(ask)> /synthesizer grok:grok-4.5  # 切換 synthesizer
openconvene(ask)> /language zh-TW             # 設定模型回應語言為繁體中文
openconvene(ask)> /usage                      # view per-CLI usage stats
openconvene(ask)> /status                     # 查看當前 session 狀態
openconvene(ask)> /new                        # 清除 session 重新開始
openconvene(ask)> /models                     # list configured models
openconvene(ask)> /detect                     # detect installed CLIs
openconvene(ask)> /config                     # show current config
openconvene(ask)> /help                       # show all commands
openconvene(ask)> /exit                       # exit REPL
```

> **REPL 特色**：
> - **fish-style menu-complete** — 按 Tab 列出補全選單，上下鍵導航候選項，Enter 確認，Shift-Tab 反向循環。支援兩階段補全：命令名（`/ex` → `/executor`, `/exit`）和參數（`/executor d` → `devin`, `devin:glm-5.2`）
> - **增量歷史搜尋** — Ctrl-R 反向搜尋、Ctrl-S 正向搜尋，邊打字邊過濾
> - **上下鍵歷史** — 命令歷史儲存到 `~/.openconvene_history`，跨 session 保留
> - **無參數提示** — 輸入 `/executor` 不帶參數直接 Enter，列出所有可用模型
> - **自動生成 config** — 若沒有 `models.yaml`，第一次啟動 REPL 時自動生成預設 config（使用 `devin:glm-5.2` 等動態模型名），無需先跑 `openconvene init`
>
> **readline 引擎**：使用 [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4，支援完整 `.inputrc` 設定、Vim/Emacs 模式、語法高亮、bracketed paste。

### 語言設定

`/language` 命令（或 `--language` flag）控制**模型輸出語言**，不影響 CLI 介面（slash 命令、help、error messages 保持英文）。

```
# 在 REPL 中設定
openconvene(ask)> /language zh-TW          # 設定為繁體中文
openconvene(ask)> /language 繁體中文        # 也接受完整語言名稱
openconvene(ask)> /language English        # 切換回英文
openconvene(ask)> /language none           # 清除（使用模型預設語言）
openconvene(ask)> /lang                    # 查看當前語言

# 或用 CLI flag（單輪模式）
openconvene ask "what is CRDT?" --language zh-TW
openconvene agent "deploy the app" --language 繁體中文
```

設定後語言會**持久化到 `models.yaml`**，跨 session 保留。引擎會在 task 前注入 `[Please respond in <lang>.]` 指令，所有 responders、synthesizer、executor 都會以該語言回應。

### Agentic Outer Loop（v1.2）

code/agent 模式下，引擎不再「執行一步就停」。`ConveneLoop` 包住單趟 MoA pipeline，自動重派直到任務完成：

```bash
# 預設：自動 loop，最多 5 趟
openconvene "fix the failing tests in pkg/foo"

# 限制最多 3 趟
openconvene "refactor the auth module" --max-iterations 3

# 單趟（回到 v1.1 行為，適合 research 或明確單步任務）
openconvene "explain this function" --max-iterations 1
```

**完成度判斷（雙機制）**：
1. **顯式 `[[DONE]]` marker**：executor 在輸出中放 `[[DONE]]` 區塊表示任務完成，loop 立即停止。
2. **隱式 judge**：無 marker 時，judge 模型（synthesizer，或 executor 若無 synthesizer）被問「任務完成了嗎？若否，下一步是什麼？」。判完成→停；判未完成→把下一步當新 task 重派。

**停止條件**（任一）：`[[DONE]]` marker / judge 判完成 / 到達 `--max-iterations` / Ctrl+C。

Loop 結束後 stderr 印摘要：
```
=== Agentic Loop Summary ===
Iterations: 3
Stop reason: done-marker
Total elapsed: 2m15s
```

`--verbose` 會額外印每趟 iteration 的 mode 與 response 數量。

### Lane 分類路由（v1.2）

執行前引擎先用一個輕量分類器（第一個 responder，或 executor 若無 responder）把任務歸入 6 個 lane，再依 lane 選最適的 responders/executor：

| Lane | 適用任務 |
|------|---------|
| `hardest-coding` | 深度根因 debug、正確性關鍵改動 |
| `bulk-mechanical` | 重構、遷移、寫測試、review sweep |
| `triage` | 大量掃描、首輪過濾 |
| `taste-final` | 面向使用者的文句、文件潤飾、prompt 工程 |
| `long-context` | 大上下文綜合（唯讀分析） |
| `live-search` | 即時網頁/社群搜尋 |

```bash
# 預設：lane routing 開啟
openconvene "migrate the DB schema"   # → lane: bulk-mechanical

# 關閉 lane routing，用靜態 responders/executor
openconvene "fix the bug" --no-lane

# REPL 內查看 lane 設定
/lane
```

分類失敗時 fallback 到 `hardest-coding`（最能力的超集）。每個 lane 可在 `models.yaml` 的 `lanes:` 段覆寫模型選擇（見 04-Configuration）。

### Arbitrate 投票面板（v1.2）

synthesizer 階段可選兩種綜合策略：

- **`reasoning`（預設）**：單一 synthesizer 做推理式整合（傳統 MoA 行為）。
- **`vote`**：問題派給 1-4 個 voter 各自獨立回答，可選辯論回合（round 2），再由 chair 模型統整判決。

```bash
# CLI flag 覆寫
openconvene "design the API contract" --vote

# config 永久設定
# models.yaml:
#   defaults:
#     synthesis_mode: vote
#     vote_voters: [agy, grok, codex]
#     vote_rounds: 2
```

投票面板容錯：單一 voter 失敗不中斷，chair 用收到的意見統整。全部 voter 失敗才 fallback 到 nil synthesis（executor 讀原始回應）。

### Fallback Chain 與 Self-execute 去重（v1.2）

- **Fallback chain**：模型沒裝或 adapter 建立失敗時，自動走 `fallback:` 鏈中下一個。配置見 04-Configuration。
- **Self-execute 去重**：若某 responder 同時是 executor，在 code/agent 模式跳過該 responder 的冗餘 Respond 呼叫（executor 會在 Phase 3 產出，不必先 Respond 一次）。
- **No-nested-dispatch guard**：引擎透過 `OPENCONVENE_DEPTH` 環境變數防止 executor CLI 再叫 openconvene 形成無限遞迴。若偵測到 nested dispatch，exit 86。

### Slash Commands

| Command | Aliases | Description | Aligned with |
|---------|---------|-------------|--------------|
| `/help` | `/h`, `/?` | Show available commands | Devin, Codex, agy |
| `/status` | | Show session status (mode, models, run count) | Codex |
| `/mode [ask\|code\|agent]` | | Show or switch current mode | Devin, Codex |
| `/models` | `/m` | List all configured models | OpenConvene unique |
| `/responders [a,b,c]` | | Show or set responders（可用動態模型名） | OpenConvene unique |
| `/executor [name]` | | Show or set executor（可用動態模型名） | OpenConvene unique |
| `/synthesizer [name]` | | Show or set synthesizer (`none` to clear；可用動態模型名) | OpenConvene unique |
| `/language [lang]` | `/lang` | Show or set output language（如 `zh-TW`、`繁體中文`、`English`；`none` 清除） | OpenConvene unique |
| `/lane` | | Show lane routing configuration and built-in lanes | OpenConvene v1.2 |
| `/usage` | `/u` | Show session usage statistics (per-CLI calls) | agy |
| `/config` | `/c`, `/settings` | Show current configuration summary | agy |
| `/detect` | `/d` | Detect installed CLIs | OpenConvene unique |
| `/clear` | `/new` | Clear screen and reset session | Devin, Codex |
| `/compact` | | (stub) Summarize conversation to free tokens | Devin, Codex |
| `/resume` | `/continue` | (stub) Resume a previous session | Devin, agy |
| `/update` | | (stub) Check and install updates | Devin |
| `/exit` | `/quit`, `/q` | Exit REPL | Devin, agy |

### CLI Flags (pre-REPL)

| Flag | Description | Aligned with |
|------|-------------|--------------|
| `-p`, `--print` | Non-interactive single-shot mode | Devin, agy, Grok |
| `-m`, `--model <name>` | Specify model (alias for `--executor`) | Codex, agy, Grok |
| `--json` | JSON output format | Grok (`--output-format json`) |
| `--responders <a,b,c>` | Specify responders | OpenConvene unique |
| `--executor <name>` | Specify executor | OpenConvene unique |
| `--synthesizer <name>` | Specify synthesizer | OpenConvene unique |
| `--language <lang>` | Output language for model responses (e.g. `zh-TW`, `繁體中文`) | OpenConvene unique |
| `--config <path>` | Specify config path | OpenConvene unique |
| `--timeout <sec>` | Override timeout | OpenConvene unique |
| `--verbose` | Show raw responses and metadata | OpenConvene unique |
| `--max-iterations <N>` | Agentic loop cap (0=default 5, 1=single-shot) | OpenConvene v1.2 |
| `--no-lane` | Disable task-classification lane routing | OpenConvene v1.2 |
| `--vote` | Use multi-model voting panel for synthesis | OpenConvene v1.2 |
| `--` | Separator (add before prompt) | Devin |

### Usage Tracking

The `/usage` command shows per-CLI statistics accumulated during the current REPL session:
- Number of calls per CLI (as responder, synthesizer, or executor)
- Success and failure counts
- Total elapsed time per CLI
- Total runs and session duration

When you exit the REPL, a session summary is printed automatically.

### CLI Flag Overrides in REPL

You can pass CLI flags when entering the REPL to override config defaults:

```bash
openconvene ask --responders agy,grok,codex    # start REPL with custom responders
openconvene agent --executor devin             # start REPL with custom executor
openconvene --config /path/to/models.yaml      # start REPL with custom config
openconvene --language zh-TW                   # start REPL with Chinese output
```

These overrides apply as the initial state of the REPL session. You can still change them with slash commands inside the REPL.

---

## 完整參數參考

### Root command: `openconvene`

| Flag | 類型 | 說明 |
|------|------|------|
| `-h, --help` | bool | 顯示說明 |

### 核心命令（ask / default / agent）

| Flag | 類型 | 必填 | 預設 | 說明 |
|------|------|------|------|------|
| `<task>` | string (positional) | ✓ | — | 任務描述（值為 `-` 時從 stdin 讀取） |
| `--responders` | string | ✗ | config defaults | 逗號分隔的 responder 模型名（覆蓋 config） |
| `--executor` | string | ✗ | config defaults | executor 模型名（code/agent 模式必填） |
| `--synthesizer` | string | ✗ | config defaults | synthesizer 模型名（空 = executor 兼任） |
| `--language` | string | ✗ | config defaults | 模型回應語言（如 `zh-TW`、`繁體中文`、`English`；空 = 不指定） |
| `--config` | string | ✗ | 搜尋預設路徑 | `models.yaml` 路徑 |
| `--timeout` | int | ✗ | config defaults | 覆蓋每次呼叫 timeout（秒，>0 生效） |
| `--verbose` | bool | ✗ | false | 顯示各 responder 原始回應 + metadata 到 stderr |
| `--model`, `-m` | string | ✗ | config defaults | executor 模型名（`--executor` 的別名，對齊 Codex/agy/Grok） |
| `--json` | bool | ✗ | false | JSON 輸出格式（適合腳本/自動化，對齊 Grok） |
| `-p`, `--print` | bool | ✗ | false | 單輪模式（非互動，適合腳本） |

```bash
# ask 模式（research — read-only）
openconvene ask "<task description or - for stdin>" \
  --responders <name1,name2,...> \
  --executor <name> \
  --synthesizer <name> \
  --language <lang> \
  --config <path> \
  --timeout <seconds> \
  --verbose \
  --model <name> \
  --json

# code 模式（預設）
openconvene "<task description or - for stdin>" \
  --responders <name1,name2,...> \
  --executor <name> \
  --synthesizer <name> \
  --language <lang> \
  --config <path> \
  --timeout <seconds> \
  --verbose \
  --model <name> \
  --json

# agent 模式
openconvene agent "<task description or - for stdin>" \
  --responders <name1,name2,...> \
  --executor <name> \
  --synthesizer <name> \
  --language <lang> \
  --config <path> \
  --timeout <seconds> \
  --verbose \
  --model <name> \
  --json
```

### `models` 子命令

| Flag | 類型 | 說明 |
|------|------|------|
| `--config` | string | `models.yaml` 路徑（預設搜尋標準位置） |

```bash
openconvene models [--config <path>]
```

### `detect` 子命令

```bash
openconvene detect
```

無額外 flags。掃描 `PATH` 偵測 9 個 CLI，不需要 config 檔。

### `models-info` 子命令

```bash
openconvene models-info
```

無額外 flags。查詢每個已安裝 CLI 的可用模型清單與預設模型。

- 對於有 `models` 子命令的 CLI（agy、grok），執行該命令並解析輸出。
- 對於沒有 `models` 子命令的 CLI（devin、codex），顯示已知模型提示。
- 幫助你決定在 `models.yaml` 中設定哪些模型，特別是單一 CLI 多模型的 MoA 配置。

輸出範例：
```
--- agy [INSTALLED] ---
  Available models (8):
    - Gemini 3.5 Flash (High)
    - Gemini 3.5 Flash (Medium)
    ...

--- devin [INSTALLED] ---
  Default model: (account-dependent)
  Available models (7):
    - glm-5.2
    - swe-1.7
    ...
```

### `init` 子命令

| Flag | 類型 | 預設 | 說明 |
|------|------|------|------|
| `--path` | string | `config/models.yaml` | 輸出路徑（已存在則拒絕覆寫） |

```bash
openconvene init [--path <path>]
```

### `check` 子命令

| Flag | 類型 | 說明 |
|------|------|------|
| `--config` | string | `models.yaml` 路徑（預設搜尋標準位置） |

```bash
openconvene check [--config <path>]
```

### Config 路徑解析順序

| 優先序 | 來源 | 說明 |
|--------|------|------|
| 1（最高） | `--config` flag | 明確指定路徑 |
| 2 | `OPENCONVENE_CLI_CONFIG` 環境變數 | 未傳 flag 時檢查此環境變數 |
| 3 | `~/.config/openconvene/models.yaml` | 使用者目錄（XDG 風格） |
| 4 | `./config/models.yaml` | 當前工作目錄 |

> 若全部不存在，`LoadConfig` 回傳 error 並提示執行 `openconvene init`。

---

## 模式說明

| 模式 | 命令 | 流程 | 執行？ | 典型用途 |
|------|------|------|--------|---------|
| `ask`（內部 research） | `openconvene ask` | N responder → synthesizer → 印出結論 | ✗ 不執行 | 技術調研、方案比較、腦力激盪 |
| `code`（預設） | `openconvene` | N responder → synthesizer（可選）→ executor 寫碼/改檔 | ✓ 寫碼 | 實作功能、修 bug、重構 |
| `agent` | `openconvene agent` | N responder → synthesizer → executor agent 長時間執行 | ✓ agent | 複雜多步任務、自動化管線 |

### ask 模式

- **Phase 1（Fan-out）**：N 個 responder 平行以 read-only 模式回答問題
- **Phase 2（Synthesis）**：synthesizer（可選）整合 N 份回應成一份結論
- **Phase 3（Execution）**：跳過——ask 模式不執行任何工具、不改檔
- **輸出**：synthesis 結論（若無 synthesizer 則印出各 responder 的原始回應）
- **注意**：若指定了 `--executor`，會收到警告（executor 將被忽略）

### code 模式

- **Phase 1（Fan-out）**：N 個 responder 平行提供建議（read-only）
- **Phase 2（Synthesis）**：synthesizer（可選）整合建議
- **Phase 3（Execution）**：executor 根據整合結果實際寫碼、改檔
- **輸出**：executor 的執行結果摘要 + synthesis + responder 回應摘要
- **executor 為必填**：若未指定 executor 且 config defaults 也沒有，會報錯

### agent 模式

- **Phase 1（Fan-out）**：N 個 responder 平行提供調查策略（read-only）
- **Phase 2（Synthesis）**：synthesizer 整合策略
- **Phase 3（Execution）**：executor 以 agentic 模式執行，可使用所有工具（研究、檔案操作、命令執行、多步驟工作流）
- **輸出**：executor 的執行結果摘要
- **executor 為必填**：同 code 模式

### synthesizer 的角色

synthesizer 為**可選**：

- **指定 synthesizer**：Phase 2 由獨立 synthesizer 整合 N 份回應，Phase 3 的 executor 拿到整合後的結論
- **不指定 synthesizer**（`--synthesizer` 為空且 config `defaults.synthesizer` 為 `null`）：跳過 Phase 2，executor 在 Phase 3 直接讀 N 份原始回應（透過 `BuildExecPrompt` 將回應傳入）

> 獨立 synthesizer 的優勢：executor 拿到的是經過推理整合的結論，品質更穩定。
> executor 兼任 synthesizer 的優勢：少一次 API 呼叫，延遲更低。

> **匿名化設計**：responder 的回應在傳給 synthesizer（和 executor）時，會以匿名編號標記（Response A、B、C...），而非 model 名稱。這防止 synthesizer 在與某個 responder 使用相同模型時產生偏心。model 名稱僅在 `--verbose` 的 metadata 中顯示，不會進入 LLM prompt。

---

## 模型角色

| 角色 | 職責 | 適合的模型 |
|------|------|-----------|
| responder | 平行回答問題（read-only） | `read_only=true` 的模型（如 codex、cursor、kimi） |
| synthesizer | 整合 N 份回應 | `read_only=true` 的模型 |
| executor | 執行（寫碼/agent） | `executor_capable=true` 的模型 |

### responder（回應者）

- 以 **read-only** 模式平行回答問題，不執行工具、不改檔
- N 個 responder 同時執行（goroutines fan-out），互不影響
- **建議使用 `read_only=true` 的模型**：codex（`--sandbox read-only`）、cursor、kimi
- `read_only=maybe` 的模型（agy、devin、grok、hermes、opencode）也可當 responder，但 `-p`/`--print` 模式只是軟約束，本質上仍具 agentic 能力，不保證絕對 read-only
- `read_only=false` 的模型（aider）**不應**當 responder——aider 本質是 code editor，預設會改檔

### synthesizer（整合者）

- 讀取 N 份 responder 回應，進行**推理式整合**（不是平均、不是多數表決）
- 識別各模型在各子論點上的正確性，標記幻覺或事實錯誤，組合出比任何單一回應更強的結論
- **建議使用 `read_only=true` 的模型**，避免 synthesis 期間意外執行工具
- 若 synthesizer 同時也是 responder，會收到警告（synthesis 可能偏袒自身回應）

### executor（執行者）

- 根據整合後的結論（或原始回應）實際執行——寫碼、改檔、跑指令、多步驟 agent 任務
- **必須是 `executor_capable=true` 的模型**
- code 模式：executor 寫乾淨、可運作的程式碼
- agent 模式：executor 使用所有可用工具完成任務
- 若 executor 同時也是 responder，會收到警告（execution 可能偏袒自身回應）

### 9 個 CLI 的角色適配矩陣

| CLI | read_only | 適合 responder | 適合 synthesizer | 適合 executor | 實測狀態 |
|-----|-----------|---------------|-----------------|--------------|---------|
| agy | maybe | ✓（軟約束） | ✓（軟約束） | ✓ | ✓ `--help` 已驗證 |
| codex | true | ✓（最推薦） | ✓（最推薦） | ✓ | ✓ `--help` 已驗證 |
| devin | maybe | ✓（軟約束） | ✓（軟約束） | ✓ | ✓ `--help` 已驗證 |
| grok | maybe | ✓（軟約束） | ✓（軟約束） | ✓ | ✓ `--help` 已驗證 |
| cursor | true | ✓（推薦） | ✓（推薦） | ✓ | ✗ 未安裝 |
| kimi | true | ✓（推薦） | ✓（推薦） | ✓ | ✗ 未安裝 |
| hermes | maybe | ✓（軟約束） | ✓（軟約束） | ✓ | ✗ 未安裝 |
| aider | false | ✗（會改檔） | ✗ | ✓ | ✗ 未安裝 |
| opencode | maybe | ✓（軟約束） | ✓（軟約束） | ✓ | ✗ 未安裝 |

> `read_only=maybe` 表示 CLI 的 `-p`/`--print` 模式是非互動的，但本質上仍具 agentic 能力。作為 responder 時是軟約束，不保證絕對 read-only。

---

## Config 說明

OpenConveneCLI 透過 `models.yaml` 配置各 CLI 的命令模板和能力標記。完整 schema 見 [04-Configuration.md](04-Configuration.md)。

### 檔案位置

| 優先序 | 路徑 | 說明 |
|--------|------|------|
| 1 | `--config <path>` flag | 明確指定 |
| 2 | `OPENCONVENE_CLI_CONFIG` 環境變數 | 環境變數 |
| 3 | `~/.config/openconvene/models.yaml` | 使用者目錄（XDG 風格） |
| 4 | `./config/models.yaml` | 當前工作目錄 |

### 完整 Schema

```yaml
# --- 預設值（CLI flag 未指定時使用）---
defaults:
  timeout: 120                      # int — 預設每次呼叫 timeout（秒）
  responders: ["agy", "grok"]       # []string — 預設 responder 模型名列表
  executor: "codex"                 # string — 預設 executor 模型名
  synthesizer: null                 # *string — 預設 synthesizer（null = executor 兼任）

# --- 模型配置 ---
models:
  <name>:                           # adapter 偵測名（map key）
    command: str                    # respond（read-only）命令模板，必須含 {prompt}
    execute_command: str            # execute 命令模板（可選，空 = 用 command）
    read_only: str                  # "true" | "false" | "maybe"
    timeout: int                    # 該模型 timeout（秒），覆蓋 defaults.timeout
    executor_capable: bool          # 是否能當 executor
    extra_args: list[str]           # 額外 CLI 參數
```

### 各欄位說明

| 欄位 | 型別 | 必填 | 說明 |
|------|------|------|------|
| `defaults.timeout` | int | 否 | 預設每次 CLI 呼叫的 timeout（秒） |
| `defaults.responders` | []string | 否 | 預設 responder 列表（可用動態模型名） |
| `defaults.executor` | string | 否 | 預設 executor 模型名（可用動態模型名） |
| `defaults.synthesizer` | *string | 否 | 預設 synthesizer（`null` = executor 兼任；可用動態模型名） |
| `models.<name>.command` | string | ✓ | respond 命令模板，必須含 `{prompt}` 或 `{prompt_file}` 佔位符 |
| `models.<name>.execute_command` | string | 否 | execute 命令模板（空 = 用 `command`） |
| `models.<name>.read_only` | string | 否 | `"true"` / `"false"` / `"maybe"` |
| `models.<name>.timeout` | int | 否 | 該模型專屬 timeout（0 = 繼承 defaults） |
| `models.<name>.executor_capable` | bool | 否 | 是否能當 executor（預設 false） |
| `models.<name>.extra_args` | []string | 否 | 額外 CLI 參數 |

### 動態模型名（免定義 models 區段）

除了在 `models` 區段手動定義每個模型，你也可以直接使用**動態模型名**格式：
`CLI名稱:模型名稱`

系統內建 9 個 CLI 的標準命令模板，會自動根據動態名稱生成完整命令。

**最小 config（不需要 models 區段）**：

```yaml
defaults:
  timeout: 120
  responders:
    - agy:Gemini 3.5 Flash (High)
    - grok:grok-4.5
    - codex:gpt-5.5
  executor: devin:glm-5.2
  synthesizer: devin:glm-5.2
```

**動態名稱格式規則**：
- 分隔符為 `:`（冒號）——不用 `-`，因為模型名常含連字號（如 `glm-5.2`），用 `-` 會造成解析歧義
- `CLI名稱` 必須是 9 個已知 CLI 之一：`devin`、`grok`、`codex`、`agy`、`cursor`、`kimi`、`hermes`、`aider`、`opencode`
- `模型名稱` 是 CLI 真正的模型名（從 `openconvene models-info` 查詢），可以是任何值
- 模型名稱可包含空格、括號、連字號（如 `Gemini 3.5 Flash (High)`、`glm-5.2`）

**範例**：

| 動態名稱 | CLI | 模型名 | 自動生成的命令 |
|----------|-----|--------|---------------|
| `agy:Gemini 3.5 Flash (High)` | agy | Gemini 3.5 Flash (High) | `agy --model "Gemini 3.5 Flash (High)" --print "{prompt}"` |
| `devin:glm-5.2` | devin | glm-5.2 | `devin --model "glm-5.2" --print --prompt-file {prompt_file}` |
| `grok:grok-4.5` | grok | grok-4.5 | `grok --model "grok-4.5" --prompt-file {prompt_file}` |
| `codex:gpt-5.5` | codex | gpt-5.5 | `type {prompt_file} \| codex exec -m gpt-5.5 --sandbox read-only ...` |

**何時用動態模型名 vs 手動定義**：

| 場景 | 建議 |
|------|------|
| 快速測試、CLI 預設命令即可 | 動態模型名（免定義） |
| 需要自訂命令模板（如加 extra_args） | 手動定義在 models 區段 |
| CLI 更新模型清單 | 動態模型名不需改 config |
| 需要非標準的 read_only 或 timeout | 手動定義 |

### `{prompt}` 與 `{prompt_file}` 佔位符

`command` 和 `execute_command` 中的佔位符會在執行時被替換為實際的 prompt 內容。有兩種佔位符可選：

**`{prompt}`** — inline 模式，prompt 直接嵌入命令字串（shell-escaped）：

```yaml
agy:
  command: 'agy -p "{prompt}"'
```

執行 respond 時實際命令為：`agy -p "你的問題內容"`

**`{prompt_file}`** — 檔案模式，prompt 寫入臨時檔案，路徑替換進命令：

```yaml
devin-glm52:
  command: 'devin --model glm-5.2 --print --prompt-file {prompt_file}'
```

執行 respond 時實際命令為：`devin --model glm-5.2 --print --prompt-file C:\...\Temp\openconvene-prompt-xxx.txt`

`{prompt_file}` 適用場景：
- CLI 支援 `--prompt-file` flag（如 devin、grok）
- prompt 包含多行內容或特殊字元（如 synthesis prompt）
- Windows 上 `{prompt}` 的 shell 引號處理有問題時

### `read_only` 值語義

| 值 | 含義 | `SupportsReadOnly()` | 說明 |
|----|------|----------------------|------|
| `"true"` | CLI 強制 read-only | true | 如 codex 的 `--sandbox read-only` |
| `"false"` | CLI 預設會改檔 | false | 如 aider，不應當 responder |
| `"maybe"` | 非互動但本質 agentic | false | 如 `-p`/`--print` 模式，軟約束 |

> `SupportsReadOnly()` 只有 `read_only="true"` 時回傳 `true`。`"maybe"` 和 `"false"` 都回傳 `false`。

### 驗證規則

`openconvene check` 會檢查：

| 規則 | 等級 | 條件 |
|------|------|------|
| 至少一個 `executor_capable=true` model | ERROR | 無任何 executor_capable 模型 |
| `read_only` 合法值 | ERROR | 非空且非 `"true"`/`"false"`/`"maybe"` |
| `timeout >= 0` | ERROR | timeout < 0（0 = 繼承 defaults，僅 WARNING） |
| `command` 含 `{prompt}` 或 `{prompt_file}` | ERROR | command 非空時必須含其中一個佔位符 |
| `execute_command` 含 `{prompt}` 或 `{prompt_file}` | ERROR | execute_command 非空時必須含其中一個佔位符 |
| `defaults.executor` 存在且 `executor_capable=true` | ERROR | 引用未知 model 或非 executor_capable |
| `defaults.responders` 非空且每個 name 存在 | ERROR | 空 / 引用未知 model |
| `defaults.synthesizer` 存在（若非 nil） | ERROR | 引用未知 model |
| responder `read_only=false` | WARNING | 可能會改檔（不阻擋） |

### 範例 config

完整的範例 config 由 `openconvene init` 產生，包含全部 9 個 adapter。以下為摘要：

```yaml
defaults:
  timeout: 120
  responders:
    - agy
    - grok
  executor: codex
  synthesizer: null

models:
  agy:
    command: 'agy -p "{prompt}"'
    execute_command: 'agy -p "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []

  codex:
    command: 'codex exec --sandbox read-only "{prompt}"'
    execute_command: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: "true"
    timeout: 180
    executor_capable: true
    extra_args: []

  devin:
    command: 'devin -p "{prompt}"'
    # ★注意：devin 的 --permission-mode bypass 不是有效值
    # S2 實測有效值為 auto/accept-edits/smart/dangerous
    # 建議使用 dangerous（最接近全自動批准）
    execute_command: 'devin --permission-mode dangerous "{prompt}"'
    read_only: "maybe"
    timeout: 300
    executor_capable: true
    extra_args: []
  # ... 其餘模型見 config/models.yaml.example
```

> ★**Devin permission-mode 注意事項**：`config/models.yaml.example` 中 devin 的 `execute_command` 預設為 `devin --permission-mode bypass "{prompt}"`，但 S2 實測發現 `bypass` 不是有效的 Devin permission mode。有效值為 `auto`、`accept-edits`、`smart`、`dangerous`。建議將 `bypass` 改為 `dangerous`（最接近全自動批准所有工具的行為）。

---

## 為什麼選 Go

| 決策點 | Go 的優勢 |
|--------|----------|
| **單一靜態二進位** | 編譯成靜態二進位，無 runtime/依賴地獄。Python 需要 venv + pip，Node 需要 node_modules。`curl + chmod` 即可用 |
| **goroutines 並發** | 原生支援 fan-out（N 個 responder 平行呼叫），比 Python asyncio 更輕量、更安全 |
| **快速啟動** | 編譯型語言，啟動快（~5ms），適合 CLI 場景。Python 啟動 + import 已 200ms+ |
| **subprocess 管理** | `os/exec` + `context.WithTimeout` 是管理外部 CLI 進程的標準方案，跨平台一致 |
| **跨平台** | `GOOS=windows/linux/darwin` 一鍵交叉編譯，無需 CI 矩陣 |
| **類型安全** | 強型別 struct（`DefaultsConfig`、`ModelConfig`）取代 `map[string]interface{}`，重構時編譯器抓錯 |
| **CLI 生態** | cobra（spf13）是 Go 最成熟的 CLI 框架；yaml.v3 是標準 YAML 解析 |
| **內建工具** | `go test` / `go vet` / `go fmt` / `go build` 零配置即用 |
