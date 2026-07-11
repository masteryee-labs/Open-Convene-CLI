# S3 — Convene Core（中空提示詞）

> 類型：C (Code) | 依賴：S2 | 並行限制：normal
> 本 Session 寫 convene 核心邏輯：並行 responder → synthesizer → executor。
> 這是中空模板——核心流程骨架固定，但各 mode 的具體行為留空讓 SubAgent 填。
> ★Go 實作：用 goroutines + errgroup 做並行 fan-out，context.Context 做取消/timeout。

---

## === S3 PROMPT（複製以下 code block 內容）===

```
你是 S3 Convene Core SubAgent。你的任務是寫 OpenConveneCLI 的核心邏輯（Go）。

═══════════════════════════════════════════════════════════════
【理論基礎 — Mixture-of-Agents（必讀，影響實作決策）】
═══════════════════════════════════════════════════════════════

本 CLI 的核心架構對齊「Mixture-of-Agents」（MoA）研究（Together AI, arXiv:2406.04692）
與 OpenRouter Fusion 的商業實踐。關鍵洞見（必須反映在程式碼與註解中）：

1. **並行 fan-out 不線性放大延遲**：N 個 responder 並行跑，總延遲 ≈ max(responder_i)，
   不是 sum(responder_i)。用 goroutines + errgroup 確保真正並行而非序列。

2. **Synthesis 不是多數投票**：synthesizer 不是數票數，而是「推理式整合」——
   讀全部回應後辨識：哪個模型在哪個子論點上正確、哪個有幻覺、哪個論證最完整，
   然後組裝一個比任何單一回應更強的答案。prompt 模板必須明確指示 synthesizer
   「不要平均、不要投票、要推理整合」。

3. **隨機性是特性不是缺陷**：模型是 stochastic，單次抽樣不保證最佳。
   並行多模型 = 從回應分佈中取多個樣本，synthesizer 從中挑最優組合。
   這是 MoA 比「單一最強模型」更強的數學基礎。

4. **權衡取捨（必須在 ValidateModeConfig 中警告）**：
   - 延遲 ↑：fan-out + synthesis 比 single-model 多 5-15 秒
   - 成本 ↑：N 個 responder + 1 synthesizer = N+1 次 API 呼叫
   - 輸出可預測性 ↓：synthesis 結果比 single deterministic model 更難預測格式
   - 適用場景：複雜推理、長文生成、多步邏輯、降低單模型盲點
   - 不適用：低延遲聊天、簡單分類、需要嚴格 JSON schema 的場景

5. **Responder 必須 read-only**：responder 只回答，不執行工具、不寫檔、不 side-effect。
   這確保 fan-out 階段不會有 N 個模型同時改檔造成衝突。
   executor 是唯一有 side-effect 的角色，且在 synthesis 之後才跑（單一執行者）。

6. **容錯是設計核心**：單個 responder 失敗不中斷——MoA 的價值在「至少一個模型答對」，
   失敗一個還有 N-1 個。Metadata 必須記錄每個 responder 的成功/失敗/耗時，
   讓使用者能 audit 哪個模型貢獻了什麼。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/01-Architecture.md") → 取得 ConveneEngine 介面定義與資料流圖
2. read(".agent/handoff/S1.md") → 取得架構決策 + DefaultsConfig 欄位定義
3. read(".agent/handoff/S2.md") → 取得 adapter 實測結果與 read_only 矩陣
4. read("internal/adapter/adapter.go") → 確認 Adapter interface + AdapterResult
5. ★read("internal/config/models.go") → 確認 ConveneConfig.Defaults 是 DefaultsConfig struct
   ★存取方式：cfg.Defaults.Responders / cfg.Defaults.Executor / cfg.Defaults.Synthesizer（struct 欄位）
   ★不是 cfg.Defaults["responders"]（map 索引——那是舊設計，已廢棄）

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

internal/convene/
├── engine.go            # ConveneEngine 核心邏輯
├── result.go            # ConveneResult struct
└── prompts.go           # BuildSynthesisPrompt / BuildExecPrompt + 模板常數

internal/mode/
└── mode.go              # Mode type + FormatOutput + ValidateModeConfig

═══════════════════════════════════════════════════════════════
【result.go — ConveneResult struct】
═══════════════════════════════════════════════════════════════

```go
package convene

// ConveneResult 是 Convene 執行的完整結果
type ConveneResult struct {
    Task      string                  // 原始任務
    Mode      string                  // "research" | "code" | "agent"
    Responses map[string]string       // {modelName: responseText}
    Synthesis *string                 // synthesizer 的整合結果（nil = executor 兼任或失敗）
    Execution *string                 // executor 的執行結果（research 模式為 nil）
    Metadata  map[string]interface{}  // timing, errors, warnings, etc.
}

// ConveneError 自訂錯誤類型
type ConveneError struct {
    Phase   string  // "respond" | "synthesize" | "execute"
    Err     error   // 底層錯誤（可為 nil，搭配 Message 用）
    Message string  // 額外訊息（可為空，Err 為主）
}

func (e *ConveneError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("convene error in %s: %v: %s", e.Phase, e.Err, e.Message)
    }
    return fmt.Sprintf("convene error in %s: %s", e.Phase, e.Message)
}
```

═══════════════════════════════════════════════════════════════
【engine.go — ConveneEngine 骨架】
═══════════════════════════════════════════════════════════════

```go
package convene

import (
    "context"
    "fmt"
    "sync"
    "time"
    "golang.org/x/sync/errgroup"
    "github.com/masteryee-labs/open-convene-cli/internal/adapter"
    "github.com/masteryee-labs/open-convene-cli/internal/config"
)

// AdapterFactory 是建立 adapter 的函式型別（★依賴注入，讓 S6 測試可 mock）
type AdapterFactory func(name string, cfg config.ModelConfig) (adapter.Adapter, error)

// ConveneEngine 多模型協作引擎
type ConveneEngine struct {
    Config          *config.ConveneConfig
    adapterFactory  AdapterFactory  // ★預設 = adapter.GetAdapter，測試可替換
}

func NewConveneEngine(cfg *config.ConveneConfig) *ConveneEngine {
    return &ConveneEngine{
        Config:         cfg,
        adapterFactory: adapter.GetAdapter,  // ★預設用真實 factory
    }
}

// ★SetAdapterFactory 供測試注入 mock factory（S6 用）
func (e *ConveneEngine) SetAdapterFactory(f AdapterFactory) {
    e.adapterFactory = f
}

// Run 執行 Convene 流程
func (e *ConveneEngine) Run(
    ctx context.Context,
    task string,
    mode string,               // "research" | "code" | "agent"
    responders []string,       // 模型名列表
    executor string,           // 模型名
    synthesizer *string,       // nil = executor 兼任
) (ConveneResult, error) {
    // <IMPLEMENTATION> ← 你填
    // ★Step 0: 初始化 metadata map + synthesis/execution 為 nil
    //   metadata := map[string]interface{}{}
    //   ★adapter 實例在 Phase 1/2/3 內各自建立（goroutine 內建立，不在此預建）
    //   ★read-only 驗證在 Phase 1 開頭做（需建立 adapter 才能呼叫 SupportsReadOnly）
    ...
}
```

→ 以上為骨架，具體實作由你填。

═══════════════════════════════════════════════════════════════
【核心流程 — 你必須實作的邏輯】
═══════════════════════════════════════════════════════════════

### Phase 1: 並行 Responders（goroutines + errgroup）

```go
// ★Go 並發模式：goroutines + sync.Mutex 保護共享 map + sync.WaitGroup/errgroup
// ★或用 channel 收集結果（更 Go-idiomatic）

type responderResult struct {
    name   string
    result adapter.AdapterResult
    err    error
    elapsed time.Duration
}

// ★read-only 驗證 + 並行 fan-out 整合（避免雙重 adapter 建立）
// ★先建立所有 adapter 並驗證 read-only，再並行 Respond
var warnings []string
type preparedResponder struct {
    name    string
    adapter adapter.Adapter
    timeout int
}
prepared := make([]preparedResponder, 0, len(responders))
for _, name := range responders {
    a, err := e.adapterFactory(name, e.Config.Models[name])
    if err != nil {
        warnings = append(warnings, fmt.Sprintf("responder %s: adapter creation failed: %v", name, err))
        continue  // ★跳過這個 responder，容錯是核心價值
    }
    if !a.SupportsReadOnly() {
        warnings = append(warnings, fmt.Sprintf("responder %s is not truly read-only, may cause side-effects during fan-out", name))
    }
    // ★per-model timeout，fallback 到 defaults
    t := e.Config.Models[name].Timeout
    if t <= 0 {
        t = e.Config.Defaults.Timeout
    }
    prepared = append(prepared, preparedResponder{name: name, adapter: a, timeout: t})
}
// ★將 warnings 存入 metadata
if len(warnings) > 0 {
    metadata["responder_warnings"] = warnings
}

// ★timeout 來源已在上面的 prepared 中計算
// ★S4 的 --timeout flag 會覆寫 cfg.Defaults.Timeout 後再傳入 engine

// ★並行 fan-out：每個 prepared responder 一個 goroutine（adapter 已建立，不重複）
results := make([]responderResult, len(prepared))
var wg sync.WaitGroup
for i, pr := range prepared {
    wg.Add(1)
    go func(idx int, p preparedResponder) {
        defer wg.Done()
        start := time.Now()
        r, err := p.adapter.Respond(ctx, task, p.timeout)
        results[idx] = responderResult{
            name: p.name, result: r, err: err, elapsed: time.Since(start),
        }
    }(i, pr)
}
wg.Wait()

// ★收集結果（容錯：單個失敗不中斷）
responses := make(map[string]string)
for _, r := range results {
    if r.err != nil {
        metadata[fmt.Sprintf("%s_error", r.name)] = r.err.Error()
    } else if r.result.Success {
        responses[r.name] = r.result.Stdout  // ← 取 .Stdout
    } else {
        metadata[fmt.Sprintf("%s_failed", r.name)] = r.result.Stderr
    }
}
// 至少 1 個 responder 成功才繼續；全失敗 → return ConveneResult{}, &ConveneError{Phase: "respond", Message: "all responders failed"}
```

- 每個 responder 用獨立 goroutine 並行跑
- ★先驗證每個 responder 的 SupportsReadOnly()，不支援的加警告到 metadata
- ★Respond() 回傳 (AdapterResult, error)——用 .Stdout 取純文字，用 .Success 判斷成敗
- 單個 responder 失敗不中斷整體（記錄到 metadata）
- 至少 1 個 responder 成功才繼續；全失敗 → return error
- ★可選用 golang.org/x/sync/errgroup 或手動 sync.WaitGroup + channel

### Phase 2: Synthesis（可選）

```go
var synthesis *string
if synthesizer != nil {
    // ★建立 synthesizer adapter（同樣處理 error，不 panic）
    synthAdapter, err := e.adapterFactory(*synthesizer, e.Config.Models[*synthesizer])
    if err != nil {
        metadata["synthesizer_error"] = err.Error()
        synthesis = nil  // fallback 到無 synthesis（不中斷）
    } else {
        synthesisPrompt := BuildSynthesisPrompt(task, responses)
        // ★timeout：per-model > 0 否則 defaults
        t := e.Config.Models[*synthesizer].Timeout
        if t <= 0 {
            t = e.Config.Defaults.Timeout
        }
        synthResult, err := synthAdapter.Respond(ctx, synthesisPrompt, t)
        if err == nil && synthResult.Success {
            s := synthResult.Stdout  // ★取 .Stdout
            synthesis = &s
        } else {
            synthesis = nil  // fallback 到無 synthesis（不中斷）
        }
    }
} else {
    synthesis = nil  // executor 兼任整合
}
```

- BuildSynthesisPrompt() 在 prompts.go 中實作，把 task + N 份回應組裝成 synthesizer 的 prompt
- ★synthAdapter.Respond() 也回傳 (AdapterResult, error)——取 .Stdout 存入 synthesis
- synthesizer 失敗 → synthesis = nil，fallback 到無 synthesis（不中斷）
- <SYNTHESIS_PROMPT_TEMPLATE> ← 填入你設計的 synthesis prompt 模板（放 prompts.go 常數）

### Phase 3: Execution（依 mode）

```go
var execution *string
switch mode {
case "research":
    execution = nil  // 不執行
case "code", "agent":
    // ★建立 executor adapter（同樣處理 error，不 panic）
    execAdapter, err := e.adapterFactory(executor, e.Config.Models[executor])
    if err != nil {
        metadata["executor_error"] = err.Error()
        // executor 建立失敗 → execution 留 nil，記錄錯誤
    } else {
        // ★BuildExecPrompt 簽名：(task, synthesis *string, responses map, mode string)
        execPrompt := BuildExecPrompt(task, synthesis, responses, mode)
        // ★timeout：per-model > 0 否則 defaults
        t := e.Config.Models[executor].Timeout
        if t <= 0 {
            t = e.Config.Defaults.Timeout
        }
        // ★Execute 第 4 參數是 synthesisContext（synthesis 的字串值，nil 時傳空）
        var synthCtx string
        if synthesis != nil {
            synthCtx = *synthesis
        }
        execResult, err := execAdapter.Execute(ctx, execPrompt, t, synthCtx)
        if err == nil && execResult.Success {
            execOut := execResult.Stdout  // ★不用 e（會遮蔽 receiver e *ConveneEngine）
            execution = &execOut
        } else {
            metadata["executor_failed"] = fmt.Sprintf("err=%v stderr=%s", err, execResult.Stderr)
        }
    }
default:
    return ConveneResult{}, &ConveneError{Phase: "execute", Err: fmt.Errorf("unknown mode: %s", mode)}
}

// ★最終 return：組裝 ConveneResult 回傳
return ConveneResult{
    Task:      task,
    Mode:      mode,
    Responses: responses,
    Synthesis: synthesis,
    Execution: execution,
    Metadata:  metadata,
}, nil
```

- <EXEC_PROMPT_TEMPLATE_CODE> ← 填入 code 模式的 executor prompt 模板
- <EXEC_PROMPT_TEMPLATE_AGENT> ← 填入 agent 模式的 executor prompt 模板

═══════════════════════════════════════════════════════════════
【prompts.go — prompt 模板】
═══════════════════════════════════════════════════════════════

```go
package convene

import (
    "fmt"
    "strings"
)

// BuildSynthesisPrompt 組裝 synthesizer 的 prompt：原始 task + N 份 responder 回應
func BuildSynthesisPrompt(task string, responses map[string]string) string {
    // <IMPLEMENTATION> ← 你填
    // ★必須包含指示：「不要平均、不要投票、要推理整合」
    // 格式：task + 各 responder 的回應（標明模型名）
    ...
}

// BuildExecPrompt 組裝 executor 的 prompt：task + synthesis（或原始 responses）
func BuildExecPrompt(task string, synthesis *string, responses map[string]string, mode string) string {
    // <IMPLEMENTATION> ← 你填
    // 若 synthesis != nil → 用 synthesis
    // 若 synthesis == nil → 用原始 responses 組裝
    // mode = "code" | "agent" 影響指示語
    ...
}
```

═══════════════════════════════════════════════════════════════
【mode.go — 三種模式邏輯】
═══════════════════════════════════════════════════════════════

```go
package mode

import (
    "fmt"
    "strings"
    "github.com/masteryee-labs/open-convene-cli/internal/config"
    "github.com/masteryee-labs/open-convene-cli/internal/convene"
)

// Mode 类型
type Mode string

const (
    ModeResearch Mode = "research"
    ModeCode     Mode = "code"
    ModeAgent    Mode = "agent"
)

// FormatOutput 依模式格式化最終輸出給使用者
func FormatOutput(result convene.ConveneResult, mode Mode) string {
    // <IMPLEMENTATION> ← 你填
    ...
}

// ValidateModeConfig 驗證 mode + model 組合是否合理
// ★回傳 (errors, warnings)：errors 是 hard error（S4 需中止），warnings 是軟警告（續行但告知）
func ValidateModeConfig(mode Mode, responders []string, executor string,
    synthesizer *string, models map[string]config.ModelConfig) (errors []string, warnings []string) {
    // <IMPLEMENTATION> ← 你填
    // 必須檢查的項目：
    // 1. research 模式不需要 executor → 若指定了 executor，warnings 加「will be ignored」
    // 2. code/agent 模式缺 executor → errors 加「mode requires executor」（★hard error）
    // 3. code/agent 模式用 read_only=false 的模型當 responder → warnings 加 side-effect 警告
    // 4. ★N=1 時 MoA 價值消失 → warnings 加「only 1 responder, MoA benefit lost」
    // 5. ★synthesizer 同時也是 responder → warnings 加「synthesizer is also a responder,
    //    may bias synthesis toward its own response」
    // 6. ★executor 同時也是 responder → warnings 加「executor is also a responder,
    //    may bias execution toward its own response」
    // 7. ★synthesizer 不是 read_only → warnings 加「synthesizer is not read-only,
    //    may execute tools during synthesis」
    // ★S4 runRun 收到 errors 非空 → 印出 errors + return error（中止）
    // ★S4 runRun 收到 warnings 非空 → 印出 warnings 到 stderr（續行）
    ...
}
```

<MODE_BEHAVIOR_NOTES> ← 填入你對三種模式行為差異的設計筆記

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. 並發用 goroutines + sync.WaitGroup（或 errgroup），不用 Python asyncio
2. responder 並行，不序列——總延遲 ≈ max(responder_i)
3. 至少 1 個 responder 成功才繼續（容錯）
4. timeout 可配置（從 config 讀），用 context.WithTimeout 傳遞
5. 所有 prompt 模板不硬編碼在程式邏輯中——用常數或 config
6. ConveneResult 含完整 Metadata（每個 responder 的耗時、成功/失敗）
7. ★Go 指標語意：Synthesis/Execution 用 *string（nil = 無值），不是空字串
8. 錯誤處理：responder 失敗記錄但不中斷；executor 失敗返回 error；synthesizer 失敗 fallback 到無 synthesis
9. ★golang.org/x/sync 需加入 go.mod：exec("go get golang.org/x/sync/errgroup")

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- engine.go 含 ConveneEngine + Run method
- result.go 含 ConveneResult struct + ConveneError
- prompts.go 含 BuildSynthesisPrompt + BuildExecPrompt + 模板常數
- mode.go 含 Mode type + FormatOutput + ValidateModeConfig
- goroutines + sync.WaitGroup/errgroup 用於並行 responder
- AdapterResult.Stdout 提取邏輯正確
- 所有 <PLACEHOLDER> 已填入
- exec("go build ./internal/...") 不報錯
- git commit: feat(S3): implement Go convene engine core and mode logic

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S3.md，內容：
- 產出檔案清單（engine.go, result.go, prompts.go, mode.go）
- ConveneEngine.Run() 的最終 Go 簽名
- ★ConveneEngine.SetAdapterFactory() 的存在（給 S6 用——測試時注入 mock factory，不需修改 engine.go）
- 三種模式的行為差異摘要
- synthesis/executor prompt 模板（從 prompts.go 摘錄）
- 已知限制
- git commit hash
```
