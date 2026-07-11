# S5 — Config System（中空提示詞）

> 類型：C (Code) | 依賴：S1 | 並行限制：normal
> 本 Session 寫 config 系統（models.yaml 載入、驗證、範例生成）。
> 可與 S2/S3/S4 並行。中空模板——schema 由 S1 定義，實作細節留空。
> ★Go 實作：用 gopkg.in/yaml.v3 解析 YAML，Go struct + yaml tags 做 mapping。

---

## === S5 PROMPT（複製以下 code block 內容）===

```
你是 S5 Config System SubAgent。你的任務是寫 OpenConveneCLI 的 config 系統（Go）。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/04-Configuration.md") → 取得 models.yaml schema 定義
2. read("Docs/01-Architecture.md") → 取得 config 相關架構
3. read(".agent/handoff/S1.md") → 取得 config schema 最終定義
4. ★read("internal/config/models.go") → S1 已產出 struct 骨架（ModelConfig/DefaultsConfig/ConveneConfig）
   ★不要覆蓋 models.go——S5 只產出 config.go（載入邏輯）+ models.yaml.example
   ★若需在 ModelConfig 加方法（IsReadOnly 等），在 config.go 中加（Go 允許同 package 跨檔案定義方法）

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

internal/config/
└── config.go             # LoadConfig / ValidateConfig / GenerateExampleConfig / InitConfig + IsReadOnly 方法
  ★models.go 已由 S1 產出，不產出 models.go

config/
└── models.yaml.example   # 範例 config

═══════════════════════════════════════════════════════════════
【config.go — 載入與驗證 + struct 方法】
═══════════════════════════════════════════════════════════════

★ModelConfig/DefaultsConfig/ConveneConfig struct 已由 S1 定義在 models.go。
S5 在 config.go 中加方法（Go 允許同 package 跨檔案定義方法）：

```go
package config

// IsReadOnly 回傳真正 read_only（maybe 算不可靠 → false）
func (m *ModelConfig) IsReadOnly() bool {
    return m.ReadOnly == "true"
}

// IsMaybeReadOnly 回傳 maybe = prompt 級軟約束，不可靠
func (m *ModelConfig) IsMaybeReadOnly() bool {
    return m.ReadOnly == "maybe"
}
```

═══════════════════════════════════════════════════════════════
【config.go — 載入與驗證】
═══════════════════════════════════════════════════════════════

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

// DefaultConfigPaths 預設搜尋路徑
func DefaultConfigPaths() []string {
    home, _ := os.UserHomeDir()
    return []string{
        filepath.Join(home, ".config", "openconvene-cli", "models.yaml"),
        filepath.Join("config", "models.yaml"),
    }
}

// LoadConfig 載入 models.yaml。path="" 時搜尋預設路徑。
func LoadConfig(path string) (*ConveneConfig, error) {
    // <IMPLEMENTATION> ← 填入
    // 1. 解析路徑：--config > 環境變數 OPENCONVENE_CLI_CONFIG > 預設路徑
    // 2. 讀 YAML 檔案（os.ReadFile）
    // 3. yaml.Unmarshal 解析成 ConveneConfig
    // 4. ★填入 ModelConfig.Name（yaml:"-" 不自動解析，用 map key 填入）：
    //    for name, cfg := range result.Models { cfg.Name = name; result.Models[name] = cfg }
    // 5. 驗證（ValidateConfig）
    // 6. 若 config 不存在 → 報錯 + 提示用 `openconvene-cli config init` 生成
    ...
}

// ValidateConfig 驗證 config 合法性，回傳錯誤/警告清單
func ValidateConfig(cfg *ConveneConfig) []string {
    // <IMPLEMENTATION> ← 填入
    // 檢查：
    // - 每個 model 有 command 含 {prompt}（execute_command 為空時 command 必須有 {prompt}）
    // - executor_capable=true 的 model 若有 execute_command，execute_command 必須含 {prompt}
    // - read_only 值是 true/false/maybe
    // - timeout > 0
    // - 至少有一個 executor_capable=true 的模型
    // ★defaults 引用的 model 必須在 models 中定義：
    //   - 每個 defaults.responders 中的 name 必須存在於 cfg.Models
    //   - defaults.executor 必須存在於 cfg.Models 且 executor_capable=true
    //   - defaults.synthesizer（非 nil）必須存在於 cfg.Models
    //   → 不存在 → 錯誤："defaults.responders references unknown model 'foo'"
    ...
}

// GenerateExampleConfig 生成範例 models.yaml 字串
func GenerateExampleConfig() string {
    // <IMPLEMENTATION> ← 填入
    ...
}

// InitConfig 在指定路徑生成範例 config
func InitConfig(path string) error {
    // <IMPLEMENTATION> ← 填入
    // 1. GenerateExampleConfig() → string
    // 2. os.WriteFile(path, []byte(content), 0644)
    // 3. 確保父目錄存在（os.MkdirAll）
    ...
}
```

═══════════════════════════════════════════════════════════════
【models.yaml.example — 範例 config】
═══════════════════════════════════════════════════════════════

```yaml
# OpenConveneCLI — Model Configuration
# 路徑：~/.config/openconvene-cli/models.yaml
# ★命令模板依官方文件研究，read_only 值由 S2 實測驗證

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
    read_only: maybe              # Antigravity CLI，待 S2 實測
    timeout: 120
    executor_capable: true
    extra_args: []

  codex:
    command: 'codex exec --sandbox read-only "{prompt}"'
    execute_command: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: true               # --sandbox read-only 確保 read-only
    timeout: 180
    executor_capable: true
    extra_args: []

  devin:
    command: 'devin -p "{prompt}"'
    execute_command: 'devin --permission-mode bypass "{prompt}"'
    read_only: maybe              # -p 非互動但本質 agentic
    timeout: 300
    executor_capable: true
    extra_args: []

  grok:
    command: 'grok -p "{prompt}"'
    execute_command: 'grok -p "{prompt}"'
    read_only: maybe              # 待 S2 實測
    timeout: 180
    executor_capable: true
    extra_args: []

  cursor:
    command: 'cursor agent -p "{prompt}"'
    execute_command: 'cursor agent -p --force "{prompt}"'
    read_only: true               # 無 --force 時 read-only
    timeout: 180
    executor_capable: true
    extra_args: []

  kimi:
    command: 'kimi -p "{prompt}"'
    execute_command: 'kimi -p "{prompt}"'
    read_only: true               # read-only ops 自動批准
    timeout: 180
    executor_capable: true
    extra_args: []

  hermes:
    command: 'hermes chat -q "{prompt}"'
    execute_command: 'hermes chat -q "{prompt}"'
    read_only: maybe              # 待 S2 實測
    timeout: 180
    executor_capable: true
    extra_args: []

  aider:
    command: ''                   # aider 不適合當 responder（本質是 code editor）
    execute_command: 'aider --yes --model sonnet "{prompt}"'
    read_only: false              # 預設會改檔
    timeout: 300
    executor_capable: true
    extra_args: []                # 使用者需設定 API key 環境變數（ANTHROPIC_API_KEY 等）

  opencode:
    command: 'opencode run "{prompt}"'
    execute_command: 'opencode run "{prompt}"'
    read_only: maybe              # 待 S2 實測
    timeout: 180
    executor_capable: true
    extra_args: []
```

→ ★命令模板已從官方文件研究填入，S2 需實測驗證 read_only 的 maybe 值。
→ ★aider 的 --model 值（如 sonnet）由使用者依需求修改，需配合 API key 環境變數。

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. 用 gopkg.in/yaml.v3 解析 config（exec("go get gopkg.in/yaml.v3@latest")）
2. 路徑解析順序：--config > OPENCONVENE_CLI_CONFIG 環境變數 > 預設路徑
3. config 不存在時：回傳 error + 提示用 `openconvene-cli config init` 生成
4. ValidateConfig 回傳 []string（空 = 合法）
5. 所有 <PLACEHOLDER> 在程式碼中不需要填——那是給 models.yaml.example 的
6. ★Go struct tag：yaml tags 必須與 YAML 檔的 key 一致
7. config 載入失敗的錯誤訊息要清楚（哪個欄位、哪個 model）
8. ★Synthesizer 用 *string（nil = 不指定 / executor 兼任），不是空字串

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- config.go 含 LoadConfig + ValidateConfig + GenerateExampleConfig + InitConfig + IsReadOnly/IsMaybeReadOnly 方法
- ★不產出 models.go（S1 已產出，不覆蓋）
- config/models.yaml.example 存在
- exec("go build ./internal/config/...") 不報錯
- git commit: feat(S5): implement Go config system with models.yaml loading and validation

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S5.md，內容：
- config.go 的函式簽名（LoadConfig/ValidateConfig/GenerateExampleConfig/InitConfig，給 S4 用）
- config 路徑解析順序（含環境變數名 OPENCONVENE_CLI_CONFIG，給 S4 用）
- models.yaml.example 的完整內容
- ★告知 S4：ModelConfig/ConveneConfig struct 在 internal/config/models.go（S1 產出），方法在 config.go（S5 產出）
- git commit hash
```
