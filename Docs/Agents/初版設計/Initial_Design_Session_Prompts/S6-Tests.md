# S6 — Tests（中空提示詞）

> 類型：T (Test) | 依賴：S2,S3,S4,S5 | 並行限制：normal
> 本 Session 寫測試。中空模板——測試結構固定，具體 test case 留空。
> ★Go 實作：用 go test + testify（github.com/stretchr/testify），測試檔 *_test.go 與源碼同目錄。

---

## === S6 PROMPT（複製以下 code block 內容）===

```
你是 S6 Tests SubAgent。你的任務是寫 OpenConveneCLI 的測試（Go）。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/01-Architecture.md") → 了解 Go module 結構
2. read(".agent/handoff/S2.md") → adapter 實測結果 + Adapter interface
3. read(".agent/handoff/S3.md") → ConveneEngine 介面 + prompts.go 模板
4. read(".agent/handoff/S4.md") → CLI 介面（cobra）
5. read(".agent/handoff/S5.md") → config 系統
6. read("internal/adapter/adapter.go") → Adapter interface + AdapterResult
7. read("internal/convene/engine.go") → ConveneEngine
8. read("internal/convene/prompts.go") → BuildSynthesisPrompt / BuildExecPrompt
9. read("internal/config/config.go") → LoadConfig
10. read("internal/config/models.go") → ModelConfig / ConveneConfig

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

★Go 測試慣例：測試檔 *_test.go 與源碼同目錄（不是 tests/ 子目錄）

internal/config/
└── config_test.go         # config 載入/驗證測試

internal/adapter/
└── adapter_test.go        # adapter 骨架測試（mock os/exec）

internal/convene/
├── prompts_test.go        # BuildSynthesisPrompt / BuildExecPrompt 測試
└── engine_test.go         # ConveneEngine 邏輯測試（mock adapters）

internal/mode/
└── mode_test.go           # mode 驗證/格式化測試

cmd/openconvene-cli/
└── main_test.go           # CLI 整合測試（cobra + mock engine）

★測試依賴：
  exec("go get github.com/stretchr/testify@latest")
  exec("go mod tidy")

═══════════════════════════════════════════════════════════════
【測試策略】
═══════════════════════════════════════════════════════════════

★ 重要：測試不真正呼叫 agy/codex/devin/grok CLI（需要外部服務 + 花費）。
所有 adapter 測試用 mock——Go 的 mock 方式：
  1. 實作一個 MockAdapter struct，實作 adapter.Adapter interface
  2. 在測試中注入 MockAdapter 取代真實 adapter
  3. ★ConveneEngine 已有 SetAdapterFactory() 方法（S3 實作）——測試時呼叫
     engine.SetAdapterFactory(mockFactory) 注入 mock，不需修改 engine.go
只有 CLI 的 --help 測試可以真正跑（exec("go run ./cmd/openconvene-cli --help")）。

═══════════════════════════════════════════════════════════════
【MockAdapter 骨架】
═══════════════════════════════════════════════════════════════

★跨 package 問題：Go test 檔只在所屬 package 編譯。MockAdapter 定義在
  adapter_test.go（package adapter）後，engine_test.go（package convene）
  無法存取它。解決方案：每個需要 mock 的 package 各自定義自己的 MockAdapter。

```go
// ★adapter_test.go 中（package adapter）——測試 adapter 本身用
type MockAdapter struct {
    Name          string
    ResponseText  string
    ShouldFail    bool
    IsReadOnly    bool
}

// ★engine_test.go 中（package convene）——測試 engine 用，獨立定義
// ★因為 Go test 檔不跨 package 共享，convene 測試需自己的 mock
type mockAdapter struct {
    name         string
    responseText string
    shouldFail   bool
    isReadOnly   bool
}

func (m *mockAdapter) Respond(ctx context.Context, prompt string, timeout int) (adapter.AdapterResult, error) {
    if m.shouldFail {
        return adapter.AdapterResult{Success: false, Stderr: "mock failure"}, nil
    }
    return adapter.AdapterResult{Stdout: m.responseText, Success: true, ReturnCode: 0}, nil
}

func (m *mockAdapter) Execute(ctx context.Context, prompt string, timeout int, synthCtx string) (adapter.AdapterResult, error) {
    if m.shouldFail {
        return adapter.AdapterResult{Success: false, Stderr: "mock failure"}, nil
    }
    return adapter.AdapterResult{Stdout: "executed: " + m.responseText, Success: true}, nil
}

func (m *mockAdapter) SupportsReadOnly() bool { return m.isReadOnly }
func (m *mockAdapter) GetCommand(prompt string, mode string) string {
    return "mock " + prompt
}

// ★mockFactory 回傳 mockAdapter，注入 engine.SetAdapterFactory()
func mockFactory(responses map[string]string, failNames []string) convene.AdapterFactory {
    return func(name string, cfg config.ModelConfig) (adapter.Adapter, error) {
        resp := responses[name]
        fail := false
        for _, fn := range failNames {
            if fn == name { fail = true; break }
        }
        return &mockAdapter{name: name, responseText: resp, shouldFail: fail, isReadOnly: true}, nil
    }
}
```

═══════════════════════════════════════════════════════════════
【config_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package config

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadConfigValid(t *testing.T) {
    // 合法 config 載入成功
    // <IMPLEMENT>
    ...
}

func TestLoadConfigMissingFile(t *testing.T) {
    // config 不存在 → 報錯 + 提示 init
    ...
}

func TestLoadConfigMissingPromptPlaceholder(t *testing.T) {
    // command 不含 {prompt} → validate 報錯
    ...
}

func TestValidateConfigNoExecutor(t *testing.T) {
    // 沒有 executor_capable=true 的模型 → 警告
    ...
}

func TestValidateConfigBadReadOnly(t *testing.T) {
    // read_only 值不是 true/false/maybe → 報錯
    ...
}

func TestGenerateExampleConfig(t *testing.T) {
    // 範例 config 生成且可被 LoadConfig 載入
    ...
}

func TestInitConfig(t *testing.T) {
    // InitConfig 寫檔成功
    ...
}
```

═══════════════════════════════════════════════════════════════
【adapter_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package adapter

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAgyAdapterRespond(t *testing.T) {
    // agy Respond() 回傳 stdout（mock subprocess）
    ...
}

func TestAgyAdapterSupportsReadOnly(t *testing.T) {
    // agy SupportsReadOnly() = false（agy read_only=maybe，IsReadOnly() 只認 "true"）
    ...
}

func TestAgyAdapterExecute(t *testing.T) {
    // agy Execute() = 呼叫 agy -p（agy 可 agentic，不報錯）
    ...
}

func TestCodexAdapterGetCommand(t *testing.T) {
    // codex GetCommand() 組裝正確命令
    ...
}

func TestDevinAdapterGetCommandWithModel(t *testing.T) {
    // devin GetCommand() 含 --model 參數
    ...
}

func TestAdapterTimeout(t *testing.T) {
    // adapter 超時 → AdapterResult.Success=false
    ...
}
```

═══════════════════════════════════════════════════════════════
【prompts_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package convene

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestBuildSynthesisPromptContainsTask(t *testing.T) {
    // synthesis prompt 包含原始 task
    ...
}

func TestBuildSynthesisPromptContainsAllResponses(t *testing.T) {
    // synthesis prompt 包含所有 responder 的回應
    ...
}

func TestBuildSynthesisPromptIncludesNoVoteInstruction(t *testing.T) {
    // synthesis prompt 包含「不要投票、要推理整合」指示
    ...
}

func TestBuildExecPromptCodeMode(t *testing.T) {
    // code 模式的 exec prompt 包含 synthesis + task
    ...
}

func TestBuildExecPromptAgentMode(t *testing.T) {
    // agent 模式的 exec prompt 包含 synthesis + task
    ...
}

func TestBuildExecPromptNoSynthesisFallsBackToResponses(t *testing.T) {
    // 無 synthesis 時，exec prompt 使用原始 responses
    ...
}
```

═══════════════════════════════════════════════════════════════
【engine_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package convene

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/masteryee-labs/open-convene-cli/internal/adapter"
    "github.com/masteryee-labs/open-convene-cli/internal/config"
)

func TestConveneResearchMode(t *testing.T) {
    // research 模式：N responder → synthesizer → 印結論，不執行
    ...
}

func TestConveneCodeMode(t *testing.T) {
    // code 模式：N responder → executor 寫碼
    ...
}

func TestConveneAgentMode(t *testing.T) {
    // agent 模式：N responder → executor agent
    ...
}

func TestConveneResponderPartialFailure(t *testing.T) {
    // 1 個 responder 失敗，其他成功 → 仍繼續
    ...
}

func TestConveneAllRespondersFail(t *testing.T) {
    // 全部 responder 失敗 → error
    ...
}

func TestConveneNoSynthesizer(t *testing.T) {
    // 無 synthesizer → executor 兼任整合
    ...
}

func TestConveneWithSynthesizer(t *testing.T) {
    // 有 synthesizer → synthesizer 整合後交 executor
    ...
}
```

═══════════════════════════════════════════════════════════════
【mode_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package mode

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestValidateModeResearchNoExecutor(t *testing.T) {
    // research 模式不需要 executor → 無警告
    ...
}

func TestValidateModeCodeNoExecutor(t *testing.T) {
    // code 模式無 executor → hard error（errors 非空，不是 warnings）
    // ★ValidateModeConfig 回傳 (errors, warnings)，code/agent 缺 executor 是 error
    ...
}

func TestValidateModeResponderNotReadOnly(t *testing.T) {
    // responder 用 read_only=false 的模型 → 警告
    ...
}

func TestFormatOutputResearch(t *testing.T) {
    // research 輸出格式正確
    ...
}

func TestFormatOutputCode(t *testing.T) {
    // code 輸出格式正確
    ...
}
```

═══════════════════════════════════════════════════════════════
【main_test.go — 測試清單（中空）】
═══════════════════════════════════════════════════════════════

```go
package main

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCLIHelp(t *testing.T) {
    // openconvene-cli --help 不報錯
    // ★用 exec("go run ./cmd/openconvene-cli --help") 或直接呼叫 cobra rootCmd.Execute()
    ...
}

func TestCLIRunHelp(t *testing.T) {
    // openconvene-cli run --help 不報錯
    ...
}

func TestCLIRunMissingTask(t *testing.T) {
    // openconvene-cli run --mode research（缺 --task）→ 報錯
    ...
}

func TestCLIRunResearch(t *testing.T) {
    // openconvene-cli run --mode research --responders agy --task "test" → 正常（mock engine）
    ...
}

func TestCLIListModels(t *testing.T) {
    // openconvene-cli list-models → 列出模型（mock config）
    ...
}

func TestCLIConfigInit(t *testing.T) {
    // openconvene-cli config init --path <tmp> → 生成檔案
    ...
}
```

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. 用 go test + testify（github.com/stretchr/testify/assert + require）
2. 所有 adapter/subprocess 呼叫用 MockAdapter，不真正呼叫外部 CLI
3. CLI --help 測試可以真正跑（用 cobra 的 Execute() 或 os/exec）
4. ★Go 測試慣例：table-driven tests（t.Run 子測試）—— 建議用此模式
5. test function 命名：Test<行為描述>
6. ★測試依賴加入 go.mod：exec("go get github.com/stretchr/testify@latest") + exec("go mod tidy")
7. 所有 <IMPLEMENT> 必須填入實際測試程式碼
8. ★不修改 S2-S5 的源碼檔——只加 *_test.go 檔案
   ★ConveneEngine 已有 SetAdapterFactory()（S3 實作），測試用 mockFactory 注入，不需修改 engine.go

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- 6 個 *_test.go 檔都存在（config, adapter, prompts, engine, mode, main）
- go.mod 含 testify 依賴
- exec("go test ./... -run=^$$ -list='.*'") 不報錯（至少能收集所有 test）
  ★或 exec("go test ./... -count=1") 全部通過
- test 數量 ≥ 20（exec("go test ./... -v 2>&1 | findstr RUN" 或 grep RUN））
- git commit: feat(S6): add Go test suite for config/adapter/prompts/convene/mode/cli

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S6.md，內容：
- 測試檔清單 + 各檔 test 數量
- exec("go test ./... -v") 結果摘要
- 已知 mock 策略摘要
- git commit hash
```
