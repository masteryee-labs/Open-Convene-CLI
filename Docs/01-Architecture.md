# 01 — 系統架構

> **版本**：v1.0（S1 架構 Session 產出）
> **Module**：`github.com/masteryee-labs/open-convene-cli`
> **語言**：Go >= 1.24

---

## 1. 系統架構圖

```
                            ┌─────────────────────────────────────────────┐
                            │            openconvene (cobra)              │
                            │           cmd/openconvene/main.go            │
                            └────────────────────┬────────────────────────┘
                                                 │ parse flags + load config
                                                 ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         internal/convene (引擎層)                            │
│                                                                              │
│   ┌──────────────┐    fan-out (goroutines)    ┌───────────────────────────┐  │
│   │ ConveneEngine │───────┬──────────┬────────►│  Adapter A (agy)         │  │
│   │  .Run()       │       │          │         │  Adapter B (grok)        │  │
│   │               │       │   ...    │         │  Adapter C (codex)       │  │
│   │              ◄───────┴──────────┴─────────┤  ... N responders         │  │
│   │  (collect)   │   AdapterResult.Stdout     │  (read-only / respond)   │  │
│   └──────┬───────┘                            └───────────────────────────┘  │
│          │ responses: map[name]string                                       │
│          ▼                                                                   │
│   ┌──────────────┐    synthesizer (optional)   ┌───────────────────────────┐ │
│   │ prompts.go   │──── BuildSynthesisPrompt ──►│  Synthesizer Adapter      │ │
│   │              │◄── synthesis text ──────────┤  (read-only / respond)    │ │
│   └──────┬───────┘                            └───────────────────────────┘ │
│          │ synthesis: *string  (nil = executor 兼任)                        │
│          ▼                                                                   │
│   ┌──────────────┐    executor                ┌───────────────────────────┐ │
│   │ prompts.go   │──── BuildExecPrompt ──────►│  Executor Adapter          │ │
│   │              │◄── execution result ───────┤  (agentic / execute)       │ │
│   └──────┬───────┘                            └───────────────────────────┘ │
│          │                                                                   │
│          ▼                                                                   │
│   ┌──────────────┐                                                            │
│   │ ConveneResult│  → FormatOutput (internal/mode) → stdout                  │
│   └──────────────┘                                                            │
└──────────────────────────────────────────────────────────────────────────────┘

                            ┌─────────────────────┐
                            │  internal/config     │
                            │  models.yaml → struct│
                            │  ConveneConfig       │
                            └─────────────────────┘
```

### 分層說明

| 層 | 套件 | 職責 |
|----|------|------|
| **CLI 入口** | `cmd/openconvene` | cobra 命令解析、flag 綁定、載入 config、呼叫 ConveneEngine、格式化輸出 |
| **引擎層** | `internal/convene` | 多模型協作流程編排：fan-out responders → synthesis → execution → 結果收集 |
| **介面卡層** | `internal/adapter` | 各 CLI 的 adapter（實作 Adapter interface）、factory、detect |
| **配置層** | `internal/config` | models.yaml 解析（struct 定義 + 載入邏輯）、驗證、範例產生 |
| **模式層** | `internal/mode` | Mode 型別、輸出格式化、模式驗證 |

---

## 2. Go Module 結構與套件職責

```
module: github.com/masteryee-labs/open-convene-cli

open-convene-cli/
├── go.mod                       # Go module 定義
├── go.sum                       # 依賴校驗（go mod tidy 產生）
├── cmd/
│   └── openconvene/
│       ├── main.go              # CLI entry point（cobra root + subcommands）
│       ├── repl.go              # Interactive REPL + slash commands + usage tracking
│       └── main_test.go         # CLI tests
├── internal/
│   ├── config/
│   │   ├── config.go            # LoadConfig / ValidateConfig / GenerateExampleConfig / InitConfig
│   │   └── models.go            # ModelConfig / DefaultsConfig / ConveneConfig struct（★S1 已產出）
│   ├── adapter/
│   │   ├── adapter.go           # Adapter interface + AdapterResult struct
│   │   ├── agy.go               # Antigravity CLI adapter
│   │   ├── codex.go             # Codex CLI adapter
│   │   ├── devin.go             # Devin CLI adapter
│   │   ├── grok.go              # Grok CLI adapter
│   │   ├── cursor.go            # Cursor CLI adapter
│   │   ├── kimi.go              # Kimi Code CLI adapter
│   │   ├── hermes.go            # Hermes Agent CLI adapter
│   │   ├── aider.go             # Aider adapter
│   │   ├── opencode.go          # OpenCode CLI adapter
│   │   ├── factory.go           # GetAdapter factory function
│   │   └── detect.go            # DetectAvailableAdapters（偵測 9 個 CLI）
│   ├── convene/
│   │   ├── engine.go            # ConveneEngine（goroutines fan-out + synthesis + execution）
│   │   ├── result.go            # ConveneResult struct
│   │   └── prompts.go           # BuildSynthesisPrompt / BuildExecPrompt + 模板常數
│   └── mode/
│       └── mode.go              # Mode type + FormatOutput + ValidateModeConfig
├── config/
│   └── models.yaml.example      # 範例 config
└── Docs/
    ├── 00-Overview.md
    ├── 01-Architecture.md        # ← 本文
    ├── 02-Usage-Guide.md
    ├── 03-Model-Adapters.md
    ├── 04-Configuration.md
    ├── 05-Examples.md
    └── 06-Troubleshooting.md
```

### 各套件職責

| 套件 | 檔案 | 職責 | 實作 Session |
|------|------|------|-------------|
| `config` | `models.go` | struct 定義 + yaml tags（不含方法） | **S1（已完成）** |
| `config` | `config.go` | LoadConfig / ValidateConfig / GenerateExampleConfig / InitConfig | S5 |
| `adapter` | `adapter.go` | Adapter interface + AdapterResult struct | S2 |
| `adapter` | `*.go`（9 個） | 各 CLI adapter 實作 | S2 |
| `adapter` | `factory.go` | `GetAdapter(name, cfg) → Adapter` | S2 |
| `adapter` | `detect.go` | `DetectAvailableAdapters() → []string`（exec.LookPath） | S2 |
| `convene` | `engine.go` | ConveneEngine.Run：fan-out + synthesis + execution | S3 |
| `convene` | `result.go` | ConveneResult struct | S3 |
| `convene` | `prompts.go` | BuildSynthesisPrompt / BuildExecPrompt + 模板常數 | S3 |
| `mode` | `mode.go` | Mode 型別 + FormatOutput + ValidateModeConfig | S4 |
| `cmd` | `main.go` | cobra CLI entry point | S4 |

### Go 慣例說明

- **`internal/`**：Go 編譯器確保此目錄下的套件不被外部 module import（封裝）。只有本 module 內的 `cmd/` 可 import `internal/*`。
- **`cmd/<binary>/main.go`**：Go CLI 慣例的 entry point 位置。`<binary>` 名稱即編譯產出的執行檔名。
- **測試檔 `*_test.go`**：與源碼同目錄（Go 慣例），非 `tests/` 子目錄。例如 `adapter_test.go` 與 `adapter.go` 同放在 `internal/adapter/`。
- **Go 不需要 `__init__.py`**：每個目錄本身就是一個 package。目錄名 = package 名（慣例）。
- **package 命名**：`config`、`adapter`、`convene`、`mode`——簡短、小寫、無底線。

---

## 3. 資料流圖

### 3.1 通用流程（所有模式）

```
User
 │
 │  openconvene [ask|agent] "<task>" --responders <list> --executor <name> [--synthesizer <name>]
 │
 ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│ CLI (cobra)                                                                      │
│  1. parse flags                                                                  │
│  2. LoadConfig("config/models.yaml")  →  ConveneConfig                           │
│  3. resolve responders/executor/synthesizer (CLI flag > config defaults)         │
│  4. ConveneEngine{Config, adapterFactory}.Run(ctx, task, mode, ...)              │
└──────────────────────────────┬──────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│ ConveneEngine.Run                                                                │
│                                                                                  │
│  Phase 1: FAN-OUT (parallel)                                                     │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  for each responder in responders:                                       │   │
│  │    go func:                                                              │   │
│  │      adapter := adapterFactory(responder, cfg.Models[responder])         │   │
│  │      result := adapter.Respond(ctx, task, timeout)   ← read-only         │   │
│  │      responses[responder] = result.Stdout                                │   │
│  │  WaitGroup.Wait()                                                        │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                               │                                                  │
│                               ▼ responses: map[string]string                    │
│  Phase 2: SYNTHESIS (optional)                                                   │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  if synthesizer != nil:                                                  │   │
│  │    prompt := BuildSynthesisPrompt(task, responses)                       │   │
│  │    adapter := adapterFactory(synthesizer, ...)                           │   │
│  │    result := adapter.Respond(ctx, prompt, timeout)                       │   │
│  │    synthesis = &result.Stdout                                            │   │
│  │  else:                                                                   │   │
│  │    synthesis = nil  (executor 兼任)                                       │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                               │                                                  │
│                               ▼ synthesis: *string                              │
│  Phase 3: EXECUTION (code / agent 模式；research 跳過)                           │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  if mode != "research":                                                  │   │
│  │    execPrompt := BuildExecPrompt(task, synthesis, responses)             │   │
│  │    adapter := adapterFactory(executor, ...)                              │   │
│  │    result := adapter.Execute(ctx, execPrompt, timeout, synthesisContext) │   │
│  │    execution = &result.Stdout                                            │   │
│  │  else:                                                                   │   │
│  │    execution = nil  (research 不執行)                                     │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                               │                                                  │
│                               ▼                                                  │
│  ConveneResult{Task, Mode, Responses, Synthesis, Execution, Metadata}            │
└──────────────────────────────┬──────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│ FormatOutput (internal/mode)                                                     │
│  research  → 印出 synthesis（或 responses 若無 synthesizer）                     │
│  code      → 印出 execution 結果摘要                                             │
│  agent     → 印出 execution 結果摘要                                             │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 各模式差異

| 階段 | research | code | agent |
|------|----------|------|-------|
| Phase 1: Fan-out responders | ✓ | ✓ | ✓ |
| Phase 2: Synthesis | ✓（可選） | ✓（可選） | ✓（可選） |
| Phase 3: Execution | ✗（跳過） | ✓ executor.Execute | ✓ executor.Execute（agentic） |
| 輸出 | synthesis / responses | execution 摘要 | execution 摘要 |

> **synthesizer = nil 時**：跳過 Phase 2，executor 在 Phase 3 直接讀 N 份 responses（透過 BuildExecPrompt 將 responses 傳入）。

---

## 4. Adapter 介面定義（Go interface）

> 位於 `internal/adapter/adapter.go`（S2 實作）

```go
package adapter

import (
	"context"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// AdapterResult 是每次 adapter 呼叫的回傳值。
type AdapterResult struct {
	Stdout     string // CLI 標準輸出（純文字回應）
	Stderr     string // CLI 標準錯誤（診斷用）
	ReturnCode int    // CLI 退出碼
	Success    bool   // 是否成功（ReturnCode == 0 且 Stdout 非空）
}

// Adapter 是每個 model CLI adapter 必須實作的介面。
type Adapter interface {
	// Respond: read-only 模式，呼叫 CLI 回傳純文字，不執行工具。
	// 用於 responder 與 synthesizer。
	Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error)

	// Execute: 執行模式，呼叫 CLI agentic 模式，可使用工具（改檔、跑指令）。
	// synthesisContext 為 synthesizer 整合後的結論（可為空字串）。
	// 用於 executor。
	Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error)

	// SupportsReadOnly 回傳此 CLI 是否真正支援 read-only。
	// 基於 ModelConfig.ReadOnly（"true" → true, 其餘 → false）。
	SupportsReadOnly() bool

	// GetCommand 組裝 CLI 命令字串。
	// mode = "respond" | "execute"
	// ★回傳完整命令字串（含引號），不拆參數——RunCommand 用 shell 執行。
	// respond 模式使用 ModelConfig.Command；
	// execute 模式使用 ModelConfig.ExecuteCommand（空則 fallback 到 Command）。
	GetCommand(prompt string, mode string) string
}
```

### Adapter 設計要點

1. **GetCommand 回傳完整命令字串**：不拆成 `[]string` 參數，而是回傳含引號的完整命令，交由 `RunCommand`（shell 執行器）透過 shell 跑。原因是各 CLI 的引號規則差異大（如 `agy -p "{prompt}"` 的雙引號），用 shell 統一處理更安全。
2. **{prompt} 佔位符替換**：`GetCommand` 將 `ModelConfig.Command` 中的 `{prompt}` 替換為實際 prompt（需做 shell 跳脫）。
3. **Respond vs Execute**：Respond 走 read-only 命令；Execute 走 agentic 命令。兩者共用同一個 Adapter 實例，差異在 GetCommand 的 mode 參數。
4. **上層取值**：ConveneEngine 從 `AdapterResult.Stdout` 取純文字回應。Stdout 即各 CLI 非互動模式的標準輸出。

### Factory 函式

```go
// GetAdapter 根據模型名建立對應的 adapter 實例。
// name = "agy" | "codex" | "devin" | "grok" | "cursor" | "kimi" | "hermes" | "aider" | "opencode"
// cfg = 該模型的 ModelConfig（Name 欄位由 factory 從 map key 填入）
func GetAdapter(name string, cfg config.ModelConfig) (Adapter, error)
```

---

## 5. ConveneEngine 介面定義（Go struct/method）

> 位於 `internal/convene/engine.go`（S3 實作）+ `internal/convene/result.go`

```go
package convene

import (
	"context"
	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// ConveneResult 是 Convene 執行的完整結果。
type ConveneResult struct {
	Task      string                 // 原始任務描述
	Mode      string                 // "research" | "code" | "agent"
	Responses map[string]string      // {modelName: responseText}——N 個 responder 的回應
	Synthesis *string                // synthesizer 整合結果；nil = 無 synthesizer 或失敗
	Execution *string                // executor 執行結果；nil = research 模式（不執行）或失敗
	Metadata  map[string]interface{} // timing, errors, adapter status 等
}

// AdapterFactory 是建立 adapter 的函式型別（依賴注入）。
// 預設 = adapter.GetAdapter；測試可替換為 mock（S6 用）。
type AdapterFactory func(name string, cfg config.ModelConfig) (adapter.Adapter, error)

// ConveneEngine 多模型協作引擎。
type ConveneEngine struct {
	Config         *config.ConveneConfig
	adapterFactory AdapterFactory // ★預設 = adapter.GetAdapter，測試可替換（S6 用）
}

// Run 執行 Convene 流程。
//
// responders: 模型名列表，如 []string{"agy", "grok"}
// executor:   模型名，如 "codex"
// synthesizer: 模型名指標，nil = executor 兼任 synthesizer
//
// 回傳 ConveneResult（含所有階段的回應 / 整合 / 執行結果）。
func (e *ConveneEngine) Run(
	ctx context.Context,
	task string,
	mode string,
	responders []string,
	executor string,
	synthesizer *string,
) (ConveneResult, error)
```

### ConveneEngine 設計要點

1. **AdapterFactory 依賴注入**：`ConveneEngine.adapterFactory` 是函式型別，預設指向 `adapter.GetAdapter`。S6 測試時可替換為 mock factory，回傳假的 Adapter，不需真實 CLI。這讓引擎邏輯可被單元測試。
2. **goroutines fan-out**：Phase 1 用 `sync.WaitGroup` 或 `golang.org/x/sync/errgroup` 平行呼叫 N 個 responder。單個 responder 失敗不中斷其他（收集 error 到 Metadata）。
3. **context 取消**：所有 adapter 呼叫接受 `ctx`，超時或使用者中斷（Ctrl+C）時取消所有 goroutine。
4. **Synthesis nil 語義**：`Synthesis == nil` 表示「無 synthesizer 或 synthesizer 失敗」。executor 兼任時，BuildExecPrompt 直接用 responses。
5. **Execution nil 語義**：`Execution == nil` 表示「research 模式（不執行）」或「執行失敗」。
6. **Responder 匿名化**：`BuildSynthesisPrompt` 和 `buildResponsesSection` 將 responder 回應標記為匿名編號（Response A、B、C...），而非 model 名稱。這防止 synthesizer/executor 在與某個 responder 使用相同模型時產生偏心。model 名稱僅在 `--verbose` 的 metadata 中出現（供使用者除錯），不會進入 LLM prompt。

---

## 6. CLI 介面定義（cobra）

> 位於 `cmd/openconvene/main.go`（S4 實作）

### 6.1 命令結構

```
openconvene
├── ask          # research 模式（read-only，不執行）
├── agent        # agent 模式（長時間 agent 任務）
├── models       # 列出 config 中已配置的模型 + 偵測可用 CLI
├── detect       # 偵測本機已安裝的 9 個 CLI
├── init         # 產生 models.yaml 範例
├── check        # 驗證 models.yaml 語法 + 參照完整性
├── (default)    # code 模式（預設）— openconvene "<task>"
└── (no task)    # 進入互動式 REPL — openconvene / openconvene ask / openconvene agent
```

### 6.2 核心命令（ask / default / agent）

```bash
# research 模式（ask — read-only，不執行）
openconvene ask "<task>" \
  --responders <name1,name2,...>     # 覆蓋 defaults.responders
  --executor <name>                  # 覆蓋 defaults.executor（ask 模式會被忽略）
  --synthesizer <name>               # 覆蓋 defaults.synthesizer（不指定 = executor 兼任）
  --timeout <seconds>                # 覆蓋 defaults.timeout
  --config <path>                    # config 檔路徑（預設 config/models.yaml）

# code 模式（預設 — 寫碼改檔）
openconvene "<task>" \
  --responders <name1,name2,...>     # 覆蓋 defaults.responders
  --executor <name>                  # 覆蓋 defaults.executor
  --synthesizer <name>               # 覆蓋 defaults.synthesizer（不指定 = executor 兼任）
  --timeout <seconds>                # 覆蓋 defaults.timeout
  --config <path>                    # config 檔路徑（預設 config/models.yaml）

# agent 模式（長時間 agent 任務）
openconvene agent "<task>" \
  --responders <name1,name2,...>     # 覆蓋 defaults.responders
  --executor <name>                  # 覆蓋 defaults.executor
  --synthesizer <name>               # 覆蓋 defaults.synthesizer（不指定 = executor 兼任）
  --timeout <seconds>                # 覆蓋 defaults.timeout
  --config <path>                    # config 檔路徑（預設 config/models.yaml）
```

> 不帶 task 參數時，三個命令均進入互動式 REPL（互動模式），可用 slash 指令切換模式、模型、查看使用量。詳見 02-Usage-Guide.md。

| Flag | 類型 | 預設 | 說明 |
|------|------|------|------|
| `--responders` | string (comma-sep) | config defaults | responder 模型名列表 |
| `--executor` | string | config defaults | executor 模型名 |
| `--synthesizer` | string | config defaults | synthesizer 模型名（空 = executor 兼任） |
| `--timeout` | int | config defaults | 每次呼叫 timeout 秒數 |
| `--config` | string | `config/models.yaml` | config 檔路徑 |
| `--model`, `-m` | string | config defaults | executor 模型名（`--executor` 的別名，對齊 Codex/agy/Grok） |
| `--json` | bool | false | JSON 輸出格式（對齊 Grok `--output-format json`） |
| `--verbose` | bool | false | 顯示各 responder 原始回應 + metadata 到 stderr |
| `-p`, `--print` | bool | false | 單輪模式（非互動，適合腳本） |

### 6.3 `models` 命令

```bash
openconvene models [--config <path>]
```

輸出 config 中所有已配置的模型，並標記哪些 CLI 已安裝（透過 `exec.LookPath` 偵測）：

```
MODEL       COMMAND                    READ_ONLY  EXECUTOR_CAPABLE  INSTALLED
agy         agy -p "{prompt}"          maybe      true              ✓
codex       codex exec ...             true       true              ✓
grok        grok -p "{prompt}"         maybe      true              ✗
...
```

### 6.4 `detect` 命令

```bash
openconvene detect
```

偵測本機 9 個支援的 CLI 是否已安裝（`exec.LookPath`），輸出：

```
CLI         DETECT NAME  INSTALLED  PATH
Devin       devin        ✓          /usr/local/bin/devin
Grok        grok         ✗          -
Codex       codex        ✓          C:\Users\...\codex.cmd
...
```

### 6.5 `init` / `check` 命令

```bash
# 產生範例 config
openconvene init [--path <path>]   # 預設 config/models.yaml

# 驗證 config 語法 + 參照完整性
openconvene check [--config <path>]
```

---

## 7. Config Schema（models.yaml）

> 完整 schema + 範例見 `04-Configuration.md`。此處為摘要。

Config 路徑搜尋順序：

1. `--config` flag
2. `OPENCONVENE_CLI_CONFIG` env var
3. `~/.config/openconvene/models.yaml`
4. `./config/models.yaml`

```yaml
defaults:
  timeout: 120                    # int — 預設每次呼叫 timeout 秒數
  responders: ["agy", "grok"]     # []string — 預設 responder 列表
  executor: "codex"               # string — 預設 executor
  synthesizer: null               # *string — 預設 synthesizer（null = executor 兼任）

models:
  <name>:                          # map key = adapter 名（填入 ModelConfig.Name）
    command: str                   # respond（read-only）命令模板，含 {prompt}
    execute_command: str           # execute 命令模板（可選，空 = 用 command）
    read_only: str                 # "true" | "false" | "maybe"
    timeout: int                   # 該模型 timeout 秒數
    executor_capable: bool         # 是否能當 executor
    extra_args: list[str]          # 額外 CLI 參數（如 --model）
```

### 對應 Go struct（internal/config/models.go — S1 已產出）

```go
type ModelConfig struct {
	Name            string   `yaml:"-"`
	Command         string   `yaml:"command"`
	ExecuteCommand  string   `yaml:"execute_command"`
	ReadOnly        string   `yaml:"read_only"`
	Timeout         int      `yaml:"timeout"`
	ExecutorCapable bool     `yaml:"executor_capable"`
	ExtraArgs       []string `yaml:"extra_args"`
}

type DefaultsConfig struct {
	Timeout     int      `yaml:"timeout"`
	Responders  []string `yaml:"responders"`
	Executor    string   `yaml:"executor"`
	Synthesizer *string  `yaml:"synthesizer"`
}

type ConveneConfig struct {
	Models   map[string]ModelConfig `yaml:"models"`
	Defaults DefaultsConfig         `yaml:"defaults"`
}
```

> ★重要：`Defaults` 是 `DefaultsConfig` struct（強型別），不是 `map[string]interface{}`。
> S3 存取方式：`cfg.Defaults.Responders`（struct 欄位），不是 `cfg.Defaults["responders"]`（map 索引）。

---

## 8. 跨平台說明

OpenConveneCLI 支援 **Windows / Linux / macOS** 三平台。

### 8.1 跨平台機制

| 機制 | 說明 |
|------|------|
| **Go 交叉編譯** | `GOOS=windows/linux/darwin GOARCH=amd64/arm64 go build` 一鍵產出三平台二進位。Go 原生支援，無需 CI 矩陣。 |
| **os/exec** | `exec.LookPath(name)` 跨平台偵測 CLI 是否安裝——Windows 查 `.exe`/`.cmd`/`.bat`，Linux/macOS 查 PATH。`exec.Command` 跨平台啟動子進程。 |
| **context.WithTimeout** | 跨平台一致的 timeout / 取消機制。超時自動 kill 子進程。 |
| **filepath.Join** | 路徑拼接用 `filepath.Join`（自動處理 `\` vs `/`），不硬編碼分隔符。 |
| **無平台特定系統呼叫** | 不使用 syscall / CGO，純標準庫 + cobra + yaml.v3，確保跨平台一致。 |

### 8.2 shell 執行差異

`RunCommand`（adapter 層的命令執行器）需處理跨平台 shell 差異：

| 平台 | shell | 執行方式 |
|------|-------|---------|
| Linux / macOS | `/bin/sh` | `exec.Command("sh", "-c", commandString)` |
| Windows | `cmd.exe` | `exec.Command("cmd", "/c", commandString)` |

> GetCommand 回傳完整命令字串（含引號），RunCommand 透過 shell 執行，避免各平台引號 / 參數解析差異。
> 平台判斷用 `runtime.GOOS`（`"windows"` vs `"linux"` vs `"darwin"`）。

### 8.3 CLI 偵測（detect.go）

`DetectAvailableAdapters` 對 9 個 CLI 逐一呼叫 `exec.LookPath`：

```go
var cliNames = []string{"devin", "grok", "codex", "agy", "cursor", "kimi", "hermes", "aider", "opencode"}

func DetectAvailableAdapters() map[string]bool {
	available := make(map[string]bool)
	for _, name := range cliNames {
		_, err := exec.LookPath(name)
		available[name] = (err == nil)
	}
	return available
}
```

- Windows 上 `exec.LookPath("codex")` 會自動找 `codex.exe` / `codex.cmd` / `codex.bat`。
- Linux/macOS 上找 PATH 中的 `codex`（具可執行權限）。
- 未安裝的 CLI 回傳 `false`，不影響其他 CLI 偵測。

---

## 9. Go 技術選型

| 領域 | 選擇 | 原因 |
|------|------|------|
| **CLI 框架** | `github.com/spf13/cobra` | Go 生態最成熟的 CLI 框架：subcommand、flag、help 自動產生 |
| **YAML 解析** | `gopkg.in/yaml.v3` | 標準選擇；支援 struct tag + map |
| **並發** | goroutines + `sync.WaitGroup` / `golang.org/x/sync/errgroup` | 原生 fan-out；errgroup 支援 error 傳播 |
| **subprocess** | `os/exec` + `context.WithTimeout` | 管理外部 CLI 進程的標準方案，跨平台一致 |
| **錯誤處理** | `errors.New` / `fmt.Errorf` + 自訂 `ConveneError` type | Go 慣例；ConveneError 攜帶 phase / model / cause |
| **日誌** | `log/slog`（Go 1.21+ 標準庫） | 結構化日誌，無外部依賴 |
| **REPL readline** | `github.com/reeflective/readline` v1.1.4 | fish-style menu-complete（Tab 列選單 + 上下鍵導航）、增量歷史搜尋、Vim/Emacs 模式、`.inputrc` 支援 |
| **Go 版本** | >= 1.24 | 須支援 reeflective/readline v1.1.4（要求 Go 1.23.6+）、context 取消、log/slog |

### go.mod（目前版本）

```go
module github.com/masteryee-labs/open-convene-cli

go 1.23.6

toolchain go1.24.8

require (
	github.com/reeflective/readline v1.1.4
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/sync v0.7.0
	gopkg.in/yaml.v3 v3.0.1
)
```

> `go.sum` 由 `go mod tidy` 產生。reeflective/readline 取代了早期的 chzyer/readline，提供 fish-style menu-complete 補全體驗。

---

## 10. 錯誤處理設計

### ConveneError 自訂錯誤型別

```go
// ConveneError 攜帶 Convene 流程的上下文資訊。
type ConveneError struct {
	Phase  string // "fanout" | "synthesis" | "execution"
	Model  string // 出錯的模型名
	Cause  error  // 底層錯誤
}

func (e *ConveneError) Error() string
func (e *ConveneError) Unwrap() error
```

### 錯誤策略

| 情境 | 策略 |
|------|------|
| 單個 responder 失敗 | 不中斷其他 responder；記錄到 `ConveneResult.Metadata["errors"]`；至少 1 個成功即繼續 |
| 全部 responder 失敗 | 回傳 `ConveneError{Phase: "fanout"}` |
| Synthesizer 失敗 | `Synthesis = nil`，executor 兼任（若 mode 需執行）；research 模式則印出 responses |
| Executor 失敗 | `Execution = nil`；回傳 `ConveneError{Phase: "execution"}`（code/agent 模式） |
| Config 載入失敗 | CLI 層直接報錯退出，不進入 ConveneEngine |
| Timeout | context 取消 → adapter 回傳 `context.DeadlineExceeded` → 記錄為該模型失敗 |
