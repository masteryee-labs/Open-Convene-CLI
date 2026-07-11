# S1 — 架構文件

> 類型：A (Architecture) | 依賴：S0 | 並行限制：normal
> 本 Session 寫 OpenConveneCLI 的架構文件，定義模組邊界、adapter 介面、資料流。
> 後續 S2~S5 的程式碼 Session 都依賴本文件定義的介面。
> ★本專案使用 Go 語言——所有介面定義用 Go struct/interface，不是 Python class。

---

## === S1 PROMPT（複製以下 code block 內容）===

```
你是 S1 Architecture SubAgent。你的任務是寫 OpenConveneCLI 的架構文件。

═══════════════════════════════════════════════════════════════
【OpenConveneCLI 概念】
═══════════════════════════════════════════════════════════════

獨立 Go CLI，實現多模型協作（概念對齊 OpenRouter Fusion + Mixture-of-Agents arXiv:2406.04692）：
- N 個 responder 模型平行回答同一問題（read-only，不執行）
- synthesizer 整合 N 份回應（可選；不指定則 executor 兼任）
- executor 根據整合結果執行

三種模式：
- research: N responder → synthesizer → 印出結論（不執行）
- code: N responder → synthesizer（可選）→ executor 寫碼/改檔
- agent: N responder 出策略 → synthesizer 整合 → executor agent 長時間執行

═══════════════════════════════════════════════════════════════
【要產出的文件】
═══════════════════════════════════════════════════════════════

1. Docs/00-Overview.md
   - 一段話描述 OpenConveneCLI 是什麼
   - 設計動機（為什麼不用 Devin Skill、為什麼獨立 CLI、為什麼選 Go）
   - 與 OpenRouter Fusion / MoA 的異同

2. Docs/01-Architecture.md
   - 系統架構圖（ASCII art）
   - Go module 結構與套件職責
   - 資料流圖（task → responders → synthesizer → executor → output）
   - adapter 介面定義（Go interface）
   - convene core 的介面定義（Go struct/method）
   - CLI 介面定義（cobra 參數：run / list-models / detect / config）
   - config schema（models.yaml 的欄位定義）
   - ★跨平台說明：CLI 支援 Windows / Linux / macOS（Go 跨平台編譯 + os/exec + exec.LookPath）

3. Docs/03-Model-Adapters.md
   - 每個 adapter 的設計
   - read_only 能力矩陣
   - ★9 個支援的 CLI 清單 + 各自的非互動模式命令 + 安裝指令（僅供參考，不自動安裝）：

| CLI | 偵測名 | 非互動模式 | Read-Only | 安裝指令（僅顯示） |
|-----|--------|-----------|-----------|------------------|
| Devin | devin | devin -p "{prompt}" | maybe | curl -fsSL https://cli.devin.ai/install.sh \| bash |
| Grok | grok | grok -p "{prompt}" | maybe | curl -fsSL https://x.ai/cli/install.sh \| bash |
| Codex | codex | codex exec --sandbox read-only "{prompt}" | true | npm install -g @openai/codex |
| Antigravity | agy | agy -p "{prompt}" | maybe | curl -fsSL https://antigravity.google/cli/install.sh \| bash |
| Cursor | cursor | cursor agent -p "{prompt}" | true | curl https://cursor.com/install -fsS \| bash |
| Kimi Code | kimi | kimi -p "{prompt}" | true | curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash |
| Hermes | hermes | hermes chat -q "{prompt}" | maybe | hermes setup --portal |
| Aider | aider | aider --yes --model {model} "{prompt}" | false | python -m pip install aider-install && aider-install |
| OpenCode | opencode | opencode run "{prompt}" | maybe | 見 opencode.ai/docs/cli/ |
   - 各 CLI 的呼叫方式與限制

4. Docs/04-Configuration.md
   - models.yaml 完整 schema
   - 範例 config

═══════════════════════════════════════════════════════════════
【★也要產出的骨架檔（不是只有文件）】
═══════════════════════════════════════════════════════════════

★S1 不只寫文件——也產出專案骨架，讓後續 S2/S5 可直接 import：
- go.mod（定義 module path + go 版本）
- internal/config/models.go（核心 struct 定義，含 yaml tags，不含方法）

這是「中空模板」設計：S1 定義骨架，S2/S5 填實作邏輯。
若 S1 不產出這些骨架，S2 與 S5 並行時會因 import 路徑不存在而編譯失敗。

5. go.mod
   module github.com/masteryee-labs/open-convene-cli
   go 1.22  ★與 S0 安裝的 Go 1.22.5 對齊（go.mod 的版本是最低要求）

6. internal/config/models.go
   - ModelConfig struct（含 yaml tags）
   - DefaultsConfig struct（含 yaml tags）
   - ConveneConfig struct（含 yaml tags，Defaults 用 DefaultsConfig 型別）
   - ★只定義 struct + yaml tags，不含方法（IsReadOnly 等方法由 S5 實作）
   - ★這樣 S2 的 factory.go 可 import config.ModelConfig，不依賴 S5 DONE

═══════════════════════════════════════════════════════════════
【架構要求 — 文件中必須定義】
═══════════════════════════════════════════════════════════════

Go module 結構（文件中定義，後續 Session 實作）：

module: github.com/masteryee-labs/open-convene-cli

open-convene-cli/
├── go.mod                       # Go module 定義
├── go.sum                       # 依賴校驗
├── cmd/
│   └── openconvene-cli/
│       └── main.go              # CLI entry point（cobra）
├── internal/
│   ├── config/
│   │   ├── config.go            # LoadConfig / ValidateConfig / GenerateExampleConfig / InitConfig
│   │   └── models.go            # ModelConfig / ConveneConfig struct
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

★Go 慣例說明（文件中必須標註）：
- internal/ 確保套件不被外部 import（封裝）
- cmd/<binary>/main.go 是 Go CLI 慣例的 entry point 位置
- 測試檔 *_test.go 與源碼同目錄（Go 慣例，非 tests/ 子目錄）
- Go 不需要 __init__.py，每個目錄本身就是 package

Adapter 介面（Go interface）必須定義：

```go
// AdapterResult 是每次 adapter 呼叫的回傳值
type AdapterResult struct {
    Stdout     string
    Stderr     string
    ReturnCode int
    Success    bool
}

// Adapter 是每個 model CLI adapter 必須實作的介面
type Adapter interface {
    // Respond: read-only 模式，呼叫 CLI 回傳純文字，不執行工具
    Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error)

    // Execute: 執行模式，呼叫 CLI agentic 模式，可使用工具
    Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error)

    // SupportsReadOnly 回傳此 CLI 是否真正支援 read-only
    SupportsReadOnly() bool

    // GetCommand 組裝 CLI 命令字串。mode = "respond" | "execute"
    // ★回傳完整命令字串（含引號），不拆參數——RunCommand 用 shell 執行
    GetCommand(prompt string, mode string) string
}
```

→ 上層（ConveneEngine）從 AdapterResult.Stdout 取純文字回應

ConveneEngine 介面必須定義：

```go
// ConveneResult 是 Convene 執行的完整結果
type ConveneResult struct {
    Task      string
    Mode      string                    // "research" | "code" | "agent"
    Responses map[string]string         // {modelName: responseText}
    Synthesis *string                   // nil = 無 synthesizer 或 synthesizer 失敗
    Execution *string                   // nil = research 模式（不執行）或執行失敗
    Metadata  map[string]interface{}    // timing, errors, etc.
}

// ConveneEngine 多模型協作引擎
type ConveneEngine struct {
    Config         *config.ConveneConfig
    adapterFactory AdapterFactory  // ★預設 = adapter.GetAdapter，測試可替換（S6 用）
}

// AdapterFactory 是建立 adapter 的函式型別（★依賴注入，讓 S6 測試可 mock）
type AdapterFactory func(name string, cfg config.ModelConfig) (adapter.Adapter, error)

// Run 執行 Convene 流程
func (e *ConveneEngine) Run(
    ctx context.Context,
    task string,
    mode string,
    responders []string,        // 模型名列表，如 []string{"agy", "grok"}
    executor string,            // 模型名，如 "codex"
    synthesizer *string,        // 模型名指標，nil = executor 兼任
) (ConveneResult, error)
```

Config schema (models.yaml) 必須定義：

```yaml
defaults:
  timeout: 120
  responders: ["agy", "grok"]
  executor: "codex"
  synthesizer: null

models:
  <name>:
    command: str           # respond（read-only）模式命令模板，含 {prompt} 佔位符
    execute_command: str   # execute 模式命令模板（可選，空=用 command；如 codex 需 --sandbox workspace-write）
    read_only: str         # "true" / "false" / "maybe"
    timeout: int           # 預設 timeout 秒
    executor_capable: bool # 是否能當 executor
    extra_args: list[str]  # 額外 CLI 參數（如 --model）
```

對應 Go struct（★S1 產出 internal/config/models.go 骨架，含這些定義）：

```go
package config

// ModelConfig 單一模型的配置
type ModelConfig struct {
    Name            string   `yaml:"-"`                 // ★不從 YAML 解析——由 factory 用 map key 填入
    Command         string   `yaml:"command"`           // respond（read-only）模式命令模板，含 {prompt}
    ExecuteCommand  string   `yaml:"execute_command"`   // execute 模式命令模板（可選，空=用 Command）
    ReadOnly        string   `yaml:"read_only"`         // "true" | "false" | "maybe"
    Timeout         int      `yaml:"timeout"`
    ExecutorCapable bool     `yaml:"executor_capable"`
    ExtraArgs       []string `yaml:"extra_args"`
}

// DefaultsConfig 預設值（★強型別 struct，不是 map[string]interface{}）
type DefaultsConfig struct {
    Timeout     int      `yaml:"timeout"`
    Responders  []string `yaml:"responders"`
    Executor    string   `yaml:"executor"`
    Synthesizer *string  `yaml:"synthesizer"`  // nil = executor 兼任
}

// ConveneConfig 全域配置
type ConveneConfig struct {
    Models   map[string]ModelConfig `yaml:"models"`
    Defaults DefaultsConfig         `yaml:"defaults"`
}
```

★注意：Defaults 用 `DefaultsConfig` struct（強型別），不是 `map[string]interface{}`。
後續 S3 存取用 `cfg.Defaults.Responders`（struct 欄位），不是 `cfg.Defaults["responders"]`（map 索引）。

═══════════════════════════════════════════════════════════════
【read_only 能力矩陣 — 文件中必須標記】
═══════════════════════════════════════════════════════════════

| CLI | read_only | 原因 |
|-----|-----------|------|
| agy | maybe | Antigravity CLI，-p 非互動但本質 agentic |
| codex | true | --sandbox read-only 確保 read-only |
| devin | maybe | -p print mode 但本質 agentic |
| grok | maybe | -p 非互動但本質 agentic |
| cursor | true | 無 --force 時 read-only |
| kimi | true | read-only ops 自動批准 |
| hermes | maybe | chat -q single query 但本質 agentic |
| aider | false | 本質是 code editor，預設會改檔 |
| opencode | maybe | run 子命令但本質 agentic |

→ ★read_only 值已從官方文件研究預填，S2 實作時由 SubAgent 實測驗證。

═══════════════════════════════════════════════════════════════
【Go 技術選型 — 文件中必須標註】
═══════════════════════════════════════════════════════════════

- CLI 框架：cobra（github.com/spf13/cobra）—— Go 生態最成熟的 CLI 框架
- YAML 解析：gopkg.in/yaml.v3 —— 標準選擇
- 並發模型：goroutines + sync.WaitGroup / golang.org/x/sync/errgroup
- subprocess：os/exec + context.WithTimeout
- 錯誤處理：errors.New / fmt.Errorf + 自訂 error type（ConveneError）
- Go 版本：>= 1.22（須支援 context 取消、log/slog；與 S0 安裝的 1.22.5 對齊）

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- Docs/00-Overview.md 存在
- Docs/01-Architecture.md 存在且含：架構圖、Go module 結構、adapter 介面（Go interface）、config schema
- Docs/03-Model-Adapters.md 存在且含 read_only 矩陣
- Docs/04-Configuration.md 存在且含 models.yaml 範例
- ★go.mod 存在且 module path = github.com/masteryee-labs/open-convene-cli
- ★internal/config/models.go 存在且含 ModelConfig + DefaultsConfig + ConveneConfig struct（含 yaml tags）
- ★exec("go build ./internal/config/...") 不報錯（骨架 struct 可獨立編譯）
- git commit: feat(S1): write architecture docs + Go project skeleton for Go CLI

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S1.md，內容：
- 產出文件清單 + 骨架檔清單（go.mod, internal/config/models.go）
- 架構決策摘要（為什麼選 Go、為什麼這樣分套件）
- adapter 介面的最終 Go 定義（給 S2 用）
- ★config struct 的最終定義（ModelConfig/DefaultsConfig/ConveneConfig，給 S5 用）
  ★告知 S5：models.go 已由 S1 產出，S5 只需產 config.go（載入邏輯）+ models.yaml.example
- ConveneEngine + ConveneResult 介面的最終 Go 定義（給 S3 用）
- ★DefaultsConfig 的欄位（給 S3 用：存取方式是 cfg.Defaults.Responders，不是 map 索引）
- prompts.go 的職責說明（BuildSynthesisPrompt / BuildExecPrompt，給 S3 用）
- git commit hash
```
