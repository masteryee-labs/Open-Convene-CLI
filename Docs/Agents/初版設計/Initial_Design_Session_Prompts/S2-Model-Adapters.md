# S2 — Model Adapters（中空提示詞）

> 類型：C (Code) | 依賴：S1 | 並行限制：normal
> 本 Session 寫 model adapter 層。這是中空模板——adapter 介面由 S1 定義，
> 各 CLI 的具體呼叫方式需由執行 SubAgent 實測後填入。
> ★Go 實作：用 interface + struct，os/exec + context.Context 做非同步 + timeout。

---

## === S2 PROMPT（複製以下 code block 內容）===

```
你是 S2 Model Adapters SubAgent。你的任務是寫 OpenConveneCLI 的 model adapter 層（Go）。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/01-Architecture.md") → 取得 Adapter interface 定義 + Go module 結構
2. read("Docs/03-Model-Adapters.md") → 取得 read_only 矩陣與各 CLI 設計
3. read(".agent/handoff/S1.md") → 取得架構決策與介面最終定義

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

internal/adapter/
├── adapter.go           # Adapter interface + AdapterResult struct + BaseAdapter（共用邏輯）
├── agy.go               # Antigravity CLI adapter
├── codex.go             # Codex CLI adapter
├── devin.go             # Devin CLI adapter
├── grok.go              # Grok CLI adapter
├── cursor.go            # Cursor CLI adapter
├── kimi.go              # Kimi Code CLI adapter
├── hermes.go            # Hermes Agent CLI adapter
├── aider.go             # Aider adapter
├── opencode.go          # OpenCode CLI adapter
├── factory.go           # GetAdapter factory function
└── detect.go            # DetectAvailableAdapters（偵測系統已安裝的 CLI）

★go.mod 已由 S1 產出——不要自己建立 go.mod。
★internal/config/models.go 已由 S1 產出——可直接 import config.ModelConfig。

═══════════════════════════════════════════════════════════════
【adapter.go — Adapter interface + AdapterResult】
═══════════════════════════════════════════════════════════════

依照 S1 文件定義的介面實作。核心：

```go
package adapter

import (
    "context"
    "os/exec"
    "bytes"
    "fmt"
    "runtime"
    "syscall"
    "time"
    "strings"
    "github.com/masteryee-labs/open-convene-cli/internal/config"
)

// AdapterResult 是每次 adapter 呼叫的回傳值
type AdapterResult struct {
    Stdout     string
    Stderr     string
    ReturnCode int
    Success    bool
}

// Adapter 是每個 model CLI adapter 必須實作的介面
type Adapter interface {
    Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error)
    Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error)
    SupportsReadOnly() bool
    GetCommand(prompt string, mode string) string  // ★回傳完整命令字串，不拆參數
}

// BaseAdapter 提供共用邏輯：subprocess 執行 + timeout 處理
type BaseAdapter struct {
    Name    string
    Config  config.ModelConfig  // ★強型別，不是 interface{}——各 adapter 直接存取 .Command / .ExecuteCommand / .Timeout
}

// RunCommand 執行組裝好的命令字串，帶 timeout（context.WithTimeout）
// ★用 shell 執行（sh -c / cmd /c）——因為命令含引號包裹的 prompt，不能用 strings.Fields 拆參數
// ★★process group：sh 是直接子進程，CLI 是孫進程。timeout 殺 sh 時，
//   CLI 會變孤兒繼續跑（吃 CPU/API 配額）。用 Setpgid 讓整個 group 一起死。
func RunCommand(ctx context.Context, cmdStr string, timeout int) (AdapterResult, error) {
    // <IMPLEMENTATION> ← 你填
    // 1. context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
    // 2. ★跨平台 shell 執行 + process group：
    //    var cmd *exec.Cmd
    //    if runtime.GOOS == "windows" {
    //        cmd = exec.CommandContext(ctx, "cmd", "/c", cmdStr)
    //        cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200} // CREATE_NEW_PROCESS_GROUP
    //    } else {
    //        cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
    //        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    //    }
    // 3. ★stdin 明確設為 nil（Go 預設接 /dev/null，防止 CLI 等待 stdin 卡住）
    //    cmd.Stdin = nil
    // 4. var stdout, stderr bytes.Buffer; cmd.Stdout = &stdout; cmd.Stderr = &stderr
    // 5. cmd.Run()
    // 6. ★timeout 時殺整個 process group（防孤兒子進程）：
    //    if ctx.Err() == context.DeadlineExceeded && cmd.Process != nil {
    //        if runtime.GOOS != "windows" {
    //            syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // 負 PID = 整個 group
    //        }
    //        // Windows: CREATE_NEW_PROCESS_GROUP + taskkill /T 或靠 context cancel
    //    }
    // 7. 處理 context.DeadlineExceeded → AdapterResult{Success: false, Stderr: "timeout after Ns"}
    // 8. 回傳 AdapterResult{Stdout: stdout.String(), Stderr: stderr.String(), ReturnCode: exitCode, Success: exitCode==0}
    ...
}

// ReplacePrompt 把命令模板中的 {prompt} 替換為實際 prompt
func ReplacePrompt(template, prompt string) string {
    return strings.ReplaceAll(template, "{prompt}", prompt)
}
```

→ 以上為骨架，具體實作由你填。可依 S1 定義調整。

═══════════════════════════════════════════════════════════════
【各 CLI 命令模板 — 依官方文件研究（★SubAgent 必讀，不要猜）】
═══════════════════════════════════════════════════════════════

★每個 CLI 的非互動模式語法不同，以下是官方文件確認的命令模板。
★ModelConfig 有 Command（respond 用）和 ExecuteCommand（execute 用）兩個欄位。
★ExecuteCommand 為空時，adapter 用 Command。

| CLI | respond 命令（Command） | execute 命令（ExecuteCommand） | read_only | 說明 |
|-----|------------------------|------------------------------|-----------|------|
| agy | `agy -p "{prompt}"` | `agy -p "{prompt}"` | maybe | Antigravity CLI，-p = prompt |
| codex | `codex exec --sandbox read-only "{prompt}"` | `codex exec --sandbox workspace-write "{prompt}"` | **true** | Codex 用 exec 子命令，sandbox 控制權限 |
| devin | `devin -p "{prompt}"` | `devin --permission-mode bypass "{prompt}"` | maybe | -p = print mode，bypass = 自動批准 |
| grok | `grok -p "{prompt}"` | `grok -p "{prompt}"` | maybe | Grok CLI，-p = prompt |
| cursor | `cursor agent -p "{prompt}"` | `cursor agent -p --force "{prompt}"` | **true** | cursor agent 子命令，--force 允許改檔 |
| kimi | `kimi -p "{prompt}"` | `kimi -p "{prompt}"` | **true** | Kimi Code，-p = prompt，read-only ops 自動批准 |
| hermes | `hermes chat -q "{prompt}"` | `hermes chat -q "{prompt}"` | maybe | hermes chat 子命令，-q = single query |
| aider | （不適合當 responder） | `aider --yes --model sonnet "{prompt}"` | **false** | 需指定 --model，--yes = 自動批准 |
| opencode | `opencode run "{prompt}"` | `opencode run "{prompt}"` | maybe | opencode run 子命令 |

★實作要點：
1. 各 adapter 的 GetCommand(prompt, mode) 依 mode 回傳 Command 或 ExecuteCommand 的組裝結果
2. Command 中的 {prompt} 用 ReplacePrompt 替換
3. ★Aider 的 --model 值（如 sonnet）由使用者在 config 的 command 欄位指定，不是程式碼硬編碼
4. ★read_only 值（true/false/maybe）已從官方文件預填，S2 仍需實測驗證
   - true = 有明確 read-only 模式（codex --sandbox read-only / cursor 無 --force / kimi read-only ops auto）
   - maybe = 有 -p 非互動模式但本質 agentic，可能產生副作用
   - false = 本質是 code editor，預設會改檔（aider）

═══════════════════════════════════════════════════════════════
【各 adapter 實作指引 — 中空部分由你填】
═══════════════════════════════════════════════════════════════

### agy.go
```go
type AgyAdapter struct {
    BaseAdapter
}
```
- command 模板：`agy -p "{prompt}"`（respond 和 execute 相同）
- read_only = maybe（Antigravity CLI，-p 非互動但本質 agentic）
- SupportsReadOnly() = 依 Config.ReadOnly 值（maybe → false，因為 IsReadOnly() 只認 true）
- ★Respond/Execute 標準實作模式（所有 adapter 參照此模式）：
```go
func (a *AgyAdapter) Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error) {
    cmd := a.GetCommand(prompt, "respond")
    return RunCommand(ctx, cmd, timeout)
}

func (a *AgyAdapter) Execute(ctx context.Context, prompt string, timeout int, synthCtx string) (AdapterResult, error) {
    cmd := a.GetCommand(prompt, "execute")
    return RunCommand(ctx, cmd, timeout)
}

func (a *AgyAdapter) GetCommand(prompt string, mode string) string {
    var template string
    if mode == "execute" && a.Config.ExecuteCommand != "" {
        template = a.Config.ExecuteCommand
    } else {
        template = a.Config.Command
    }
    return ReplacePrompt(template, prompt)  // ★回傳完整命令字串，不拆參數
}

func (a *AgyAdapter) SupportsReadOnly() bool {
    return a.Config.IsReadOnly()  // ★maybe → false
}
```
- ★各 adapter 的 Respond/Execute 邏輯幾乎相同（都用 GetCommand + RunCommand）
- ★差異只在 GetCommand 選 Command 還是 ExecuteCommand，以及 SupportsReadOnly 的判斷
- <AGY_EXTRA_NOTES> ← 填入你實測 agy 時發現的特殊參數或限制

### codex.go
```go
type CodexAdapter struct {
    BaseAdapter
}
```
- ★先實測 Codex CLI 確認命令模板：
  exec("codex --help", timeout=15) → 確認 exec 子命令 + --sandbox 參數
  → ★CLI 實測必須加 timeout（15 秒），避免 CLI 等待登入互動而卡死 SubAgent
  → 若 timeout 超時 → 標記為 <CODEX_UNREACHABLE>，在 handoff 中記錄
- command 模板（respond）：`codex exec --sandbox read-only "{prompt}"`
- execute_command 模板：`codex exec --sandbox workspace-write "{prompt}"`
- SupportsReadOnly() = true（--sandbox read-only 確保 read-only）
- Respond() = 用 Config.Command（含 --sandbox read-only）
- Execute() = 用 Config.ExecuteCommand（含 --sandbox workspace-write）

### devin.go
```go
type DevinAdapter struct {
    BaseAdapter
}
```
- ★先實測 Devin CLI 的參數：
  exec("devin --help", timeout=15) → 確認 -p print mode + --permission-mode
  → ★同樣加 timeout=15，超時標記 <DEVIN_UNREACHABLE>
- command 模板（respond）：`devin -p "{prompt}"`
- execute_command 模板：`devin --permission-mode bypass "{prompt}"`
- SupportsReadOnly() = false（maybe → IsReadOnly() 回傳 false）
- ExecutorCapable = true（Devin 擅長 agentic 執行）

### grok.go
```go
type GrokAdapter struct {
    BaseAdapter
}
```
- ★先實測 Grok CLI 的介面：
  exec("grok --help", timeout=15) → 確認 -p 非互動模式
  → ★同樣加 timeout=15，超時標記 <GROK_UNREACHABLE>
- command 模板：`grok -p "{prompt}"`（respond 和 execute 相同）
- SupportsReadOnly() = 依實測確認 maybe 值是否為 true

### cursor.go
```go
type CursorAdapter struct {
    BaseAdapter
}
```
- ★先實測 Cursor CLI：
  exec("cursor --help", timeout=15) → 確認 agent 子命令 + -p + --force
- command 模板（respond）：`cursor agent -p "{prompt}"`
- execute_command 模板：`cursor agent -p --force "{prompt}"`
- SupportsReadOnly() = true（無 --force 時 read-only）

### kimi.go
```go
type KimiAdapter struct {
    BaseAdapter
}
```
- ★先實測 Kimi Code CLI：
  exec("kimi --help", timeout=15) → 確認 -p 非互動模式
- command 模板：`kimi -p "{prompt}"`（respond 和 execute 相同）
- SupportsReadOnly() = true（read-only ops 自動批准）

### hermes.go
```go
type HermesAdapter struct {
    BaseAdapter
}
```
- ★先實測 Hermes Agent CLI：
  exec("hermes --help", timeout=15) → 確認 chat 子命令 + -q single query
- command 模板：`hermes chat -q "{prompt}"`（respond 和 execute 相同）
- SupportsReadOnly() = false（maybe → IsReadOnly() 回傳 false）

### aider.go
```go
type AiderAdapter struct {
    BaseAdapter
}
```
- ★先實測 Aider：
  exec("aider --help", timeout=15) → 確認 --yes + --model 參數
- command 模板（respond）：空字串（aider 不適合當 responder）
- execute_command 模板：`aider --yes --model sonnet "{prompt}"`
  ★--model 值由使用者在 config 的 execute_command 中指定
- SupportsReadOnly() = false（本質是 code editor）
- Respond() = 回傳 `(adapter.AdapterResult{}, fmt.Errorf("aider does not support respond mode"))`

### opencode.go
```go
type OpenCodeAdapter struct {
    BaseAdapter
}
```
- ★先實測 OpenCode CLI：
  exec("opencode --help", timeout=15) → 確認 run 子命令
- command 模板：`opencode run "{prompt}"`（respond 和 execute 相同）
- SupportsReadOnly() = false（maybe → IsReadOnly() 回傳 false）

### factory.go
```go
// GetAdapter 依模型名回傳對應的 Adapter 實例
func GetAdapter(name string, cfg config.ModelConfig) (Adapter, error) {
    // <IMPLEMENTATION> ← 你填
    // ★cfg.Name 已由 S5 LoadConfig 用 map key 填入（yaml:"-" 不自動解析）
    // ★BaseAdapter 初始化模式：BaseAdapter{Name: name, Config: cfg}
    // switch name {
    // case "agy":       return &AgyAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "codex":     return &CodexAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "devin":     return &DevinAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "grok":      return &GrokAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "cursor":    return &CursorAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "kimi":      return &KimiAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "hermes":    return &HermesAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "aider":     return &AiderAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // case "opencode":  return &OpenCodeAdapter{BaseAdapter{Name: name, Config: cfg}}, nil
    // default:          return nil, fmt.Errorf("unknown adapter: %s", name)
    // }
    ...
}
```

═══════════════════════════════════════════════════════════════
【detect.go — 自動偵測已安裝的 CLI】
═══════════════════════════════════════════════════════════════

★跨平台偵測：用 exec.LookPath() 檢查各 CLI 是否在 PATH 中。
★這讓 `openconvene-cli detect` 命令能告訴使用者「你系統上實際裝了哪些 CLI」，
  與 `list-models`（列 config 中定義的模型）互補。
★★不自動安裝任何 CLI——只偵測並顯示安裝指令供使用者參考。

★支援偵測的 9 個 CLI（依官方文件研究）：

| CLI 名稱       | 偵測命令   | 非互動模式                    | Read-Only | 安裝指令（僅顯示，不自動執行） |
|---------------|-----------|------------------------------|-----------|------------------------------|
| Devin CLI     | devin     | devin -p "prompt"            | maybe     | curl -fsSL https://cli.devin.ai/install.sh \| bash |
| Grok CLI      | grok      | grok -p "prompt"             | maybe     | curl -fsSL https://x.ai/cli/install.sh \| bash |
| Codex CLI     | codex     | codex exec "prompt"          | true      | npm install -g @openai/codex |
| Antigravity   | agy       | agy -p "prompt"              | maybe     | curl -fsSL https://antigravity.google/cli/install.sh \| bash |
| Cursor CLI    | cursor    | cursor agent -p "prompt"     | true      | curl https://cursor.com/install -fsS \| bash |
| Kimi Code     | kimi      | kimi -p "prompt"             | true      | curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash |
| Hermes Agent  | hermes    | hermes chat -q "prompt"      | maybe     | hermes setup --portal（見 hermes-agent.nousresearch.com） |
| Aider         | aider     | aider --yes "prompt"         | false     | python -m pip install aider-install && aider-install |
| OpenCode      | opencode  | opencode run "prompt"        | maybe     | 見 opencode.ai/docs/cli/ |

★Read-Only 能力說明（依官方文件）：
- true：有明確的 read-only 模式（Codex --sandbox read-only / Cursor 無 --force / Kimi read-only ops auto）
- maybe：有 -p 非互動模式但本質是 agentic，可能產生副作用
- false：本質是 code editor，預設會改檔（Aider）

```go
package adapter

import (
    "fmt"
    "os/exec"
    "sort"
)

// DetectResult 單一 CLI 的偵測結果
type DetectResult struct {
    Name         string  // CLI 名稱（devin/grok/codex/agy/cursor/kimi/hermes/aider/opencode）
    Found        bool    // 是否在 PATH 中找到
    Path         string  // 找到的完整路徑（Found=false 時為空）
    ReadOnly     string  // "true" / "false" / "maybe"
    CanRespond   bool    // 是否適合當 responder
    CanExecute   bool    // 是否適合當 executor
    InstallCmd   string  // 安裝指令（僅顯示用，不自動執行）
    HeadlessCmd  string  // 非互動模式命令範例
}

// knownCLIs 是所有支援偵測的 CLI（依官方文件研究）
// ★硬編碼合理：CLI 工具本身是固定的，能力矩陣不常變
var knownCLIs = map[string]struct {
    ReadOnly    string
    CanRespond  bool
    CanExecute  bool
    InstallCmd  string
    HeadlessCmd string
}{
    "devin": {
        ReadOnly: "maybe", CanRespond: true, CanExecute: true,
        InstallCmd:  `curl -fsSL https://cli.devin.ai/install.sh | bash`,
        HeadlessCmd: `devin -p "{prompt}"`,
    },
    "grok": {
        ReadOnly: "maybe", CanRespond: true, CanExecute: true,
        InstallCmd:  `curl -fsSL https://x.ai/cli/install.sh | bash`,
        HeadlessCmd: `grok -p "{prompt}"`,
    },
    "codex": {
        ReadOnly: "true", CanRespond: true, CanExecute: true,  // --sandbox read-only
        InstallCmd:  `npm install -g @openai/codex`,
        HeadlessCmd: `codex exec --sandbox read-only "{prompt}"`,
    },
    "agy": {
        ReadOnly: "maybe", CanRespond: true, CanExecute: true,
        InstallCmd:  `curl -fsSL https://antigravity.google/cli/install.sh | bash`,
        HeadlessCmd: `agy -p "{prompt}"`,
    },
    "cursor": {
        ReadOnly: "true", CanRespond: true, CanExecute: true,  // 無 --force 時 read-only
        InstallCmd:  `curl https://cursor.com/install -fsS | bash`,
        HeadlessCmd: `cursor agent -p "{prompt}"`,
    },
    "kimi": {
        ReadOnly: "true", CanRespond: true, CanExecute: true,  // read-only ops auto-approved
        InstallCmd:  `curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash`,
        HeadlessCmd: `kimi -p "{prompt}"`,
    },
    "hermes": {
        ReadOnly: "maybe", CanRespond: true, CanExecute: true,
        InstallCmd:  `hermes setup --portal  # 見 hermes-agent.nousresearch.com`,
        HeadlessCmd: `hermes chat -q "{prompt}"`,
    },
    "aider": {
        ReadOnly: "false", CanRespond: false, CanExecute: true,  // 本質是 code editor
        InstallCmd:  `python -m pip install aider-install && aider-install`,
        HeadlessCmd: `aider --yes --model {model} "{prompt}"`,
    },
    "opencode": {
        ReadOnly: "maybe", CanRespond: true, CanExecute: true,
        InstallCmd:  `# 見 https://opencode.ai/docs/cli/`,
        HeadlessCmd: `opencode run "{prompt}"`,
    },
}

// DetectAvailableAdapters 偵測系統上已安裝哪些 CLI adapter
// 回傳所有已知 CLI 的偵測結果（Found=true 表示已安裝）
// ★結果按 Name 排序，輸出穩定
func DetectAvailableAdapters() []DetectResult {
    // <IMPLEMENTATION> ← 你填
    // 1. 遍歷 knownCLIs 的 keys（排序後遍歷，確保輸出順序穩定）
    // 2. 對每個 name 呼叫 exec.LookPath(name)
    //    → 找到 → DetectResult{Found: true, Path: path, ...}
    //    → 沒找到 → DetectResult{Found: false, ...}
    // 3. 填入 ReadOnly/CanRespond/CanExecute/InstallCmd/HeadlessCmd 從 knownCLIs
    // 4. sort.Slice(results, func(i,j) bool { return results[i].Name < results[j].Name })
    ...
}
```

→ ★exec.LookPath 是跨平台的（Windows 搜 .exe / Linux/Mac 搜 PATH）
→ ★偵測結果給 S4 的 `detect` 子命令用，顯示給使用者看
→ ★★未安裝的 CLI 顯示 InstallCmd 供使用者參考，但不自動執行安裝

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. 用 os/exec.CommandContext + context.WithTimeout 實作帶 timeout 的 subprocess 呼叫
2. ★Go 沒有 asyncio——並發由 ConveneEngine（S3）用 goroutines 管理，adapter 本身只需提供同步的 Respond/Execute
3. timeout 用 context.WithTimeout 處理，超時返回 AdapterResult{Success: false}（不 panic）
4. 每個 adapter 的 command 組裝從 config/models.yaml 讀取（不硬編碼命令）
5. Go 沒有繼承——用 struct embedding（embedded BaseAdapter）實作共用邏輯
6. 各 adapter 的 godoc 註解必須說明 read_only 能力與限制
7. factory.go 用 switch-case 模式（Go 慣例，非 map of factories）
8. ★error 處理：RunCommand 內部不 panic，所有錯誤包進 AdapterResult 或返回 error

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- adapter.go 含 Adapter interface + AdapterResult struct + RunCommand + BaseAdapter
- 9 個 adapter 檔都存在且實作 Adapter interface：
  agy.go, codex.go, devin.go, grok.go, cursor.go, kimi.go, hermes.go, aider.go, opencode.go
- factory.go 含 GetAdapter function（9 個 case）
- ★detect.go 含 DetectAvailableAdapters + DetectResult struct + knownCLIs（9 個 CLI）
- 所有 <PLACEHOLDER> 已填入實測結果
- exec("go build ./internal/adapter/...") 不報錯（語法編譯通過）
  ★前提：S1 已產出 go.mod + internal/config/models.go，import config 套件可解析
- 各 adapter 的 SupportsReadOnly() 回傳值與 Docs/03-Model-Adapters.md 矩陣一致
- git commit: feat(S2): implement 9 Go model adapters + detect for agy/codex/devin/grok/cursor/kimi/hermes/aider/opencode

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S2.md，內容：
- AdapterResult 的最終欄位定義（給 S3 提取 .Stdout 用）
- Adapter interface 的最終 Go 簽名（給 S3 用）
- 各 adapter 的 read_only 實測結果（填入最終矩陣）
- 各 CLI 的實際 command 模板
- GetAdapter 的簽名（給 S3 用）
- ★DetectAvailableAdapters 的簽名 + DetectResult struct（給 S4 的 detect 子命令用）
- 已知限制（哪個 CLI 的 read_only 不可靠）
- git commit hash
```
