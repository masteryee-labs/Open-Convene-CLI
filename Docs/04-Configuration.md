# 04 — Configuration（models.yaml）

> **版本**：v1.0（S1 架構 Session 產出）
> **套件**：`internal/config`
> **struct 定義**：`internal/config/models.go`（S1 已產出）
> **載入邏輯**：`internal/config/config.go`（S5 實作）

---

## 1. Config 檔位置

| 優先序 | 路徑 | 說明 |
|--------|------|------|
| 1（最高） | `--config <path>` CLI flag | 明確指定 |
| 2 | `OPENCONVENE_CLI_CONFIG` 環境變數 | 環境變數 |
| 3 | `~/.config/openconvene/models.yaml` | 使用者目錄（XDG 風格） |
| 4 | `./config/models.yaml` | 當前工作目錄 |

> 載入邏輯在 `config.go` 的 `LoadConfig` 實作。搜尋順序為先 `--config` flag，再 `OPENCONVENE_CLI_CONFIG` env，再 `~/.config/openconvene/models.yaml`，最後 `./config/models.yaml`（first match wins）。

---

## 2. 完整 Schema

```yaml
# ============================================================
# OpenConveneCLI — models.yaml
# ============================================================

# --- 預設值（CLI 未明確指定時使用）---
defaults:
  timeout: 120                      # int  — 預設每次呼叫 timeout（秒）
  responders: ["agy", "grok"]       # []string — 預設 responder 模型名列表
  executor: "codex"                 # string — 預設 executor 模型名
  synthesizer: null                 # *string — 預設 synthesizer 模型名
                                    #   null = executor 兼任 synthesizer
  language: ""                      # string — 模型回應語言（空 = 不指定）
                                    #   例: "zh-TW", "繁體中文", "English"
                                    #   只影響模型輸出，CLI UI 保持英文

# --- 模型配置（map: name → ModelConfig）---
models:
  <name>:                           # string — adapter 偵測名（map key）
    command: str                    # string — respond（read-only）命令模板
                                    #   ★必須含 {prompt} 佔位符
                                    #   例: 'agy -p "{prompt}"'
    execute_command: str            # string — execute（agentic）命令模板
                                    #   可選；空字串或省略 = 用 command
                                    #   例: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: str                  # string — "true" | "false" | "maybe"
                                    #   true  = CLI 強制 read-only
                                    #   false = CLI 預設會改檔（如 aider）
                                    #   maybe = 非互動但本質 agentic
    timeout: int                    # int — 該模型 timeout（秒），覆蓋 defaults.timeout
    executor_capable: bool          # bool — 是否能當 executor（agentic 執行）
    extra_args: list[str]           # []string — 額外 CLI 參數
                                    #   例: ["--model", "gpt-4o"]（aider 用）
```

### 欄位型別對應

| YAML 欄位 | Go struct 欄位 | Go 型別 | yaml tag | 必填？ |
|-----------|---------------|---------|----------|--------|
| `defaults.timeout` | `DefaultsConfig.Timeout` | `int` | `timeout` | 否（有零值） |
| `defaults.responders` | `DefaultsConfig.Responders` | `[]string` | `responders` | 否 |
| `defaults.executor` | `DefaultsConfig.Executor` | `string` | `executor` | 否 |
| `defaults.synthesizer` | `DefaultsConfig.Synthesizer` | `*string` | `synthesizer` | 否（nil = 兼任） |
| `defaults.language` | `DefaultsConfig.Language` | `string` | `language` | 否（空 = 不指定） |
| `models.<name>`（map key） | `ModelConfig.Name` | `string` | `-`（不解析） | ★由 factory 填入 |
| `models.<name>.command` | `ModelConfig.Command` | `string` | `command` | ★是（必須含 {prompt}） |
| `models.<name>.execute_command` | `ModelConfig.ExecuteCommand` | `string` | `execute_command` | 否（空 = 用 command） |
| `models.<name>.read_only` | `ModelConfig.ReadOnly` | `string` | `read_only` | 否（預設空 = maybe 語義） |
| `models.<name>.timeout` | `ModelConfig.Timeout` | `int` | `timeout` | 否（用 defaults） |
| `models.<name>.executor_capable` | `ModelConfig.ExecutorCapable` | `bool` | `executor_capable` | 否（預設 false） |
| `models.<name>.extra_args` | `ModelConfig.ExtraArgs` | `[]string` | `extra_args` | 否 |

> ★`Name` 欄位 `yaml:"-"`：不從 YAML 解析。`LoadConfig` 解析後，factory 用 map key 填入 `cfg.Models["agy"].Name = "agy"`。

---

## 3. 範例 Config（models.yaml.example）

> 此檔由 S5 產出至 `config/models.yaml.example`。以下為完整內容供參考。

```yaml
# ============================================================
# OpenConveneCLI — models.yaml
# 範例配置。複製為 models.yaml 後依需求修改。
# 產生方式: openconvene init
# ============================================================

defaults:
  timeout: 120
  responders: ["agy", "grok"]
  executor: "codex"
  synthesizer: null              # null = executor 兼任 synthesizer
  # language: "zh-TW"            # 取消註解以設定模型回應語言

models:
  # --- Antigravity (AGY) ---
  agy:
    command: 'agy -p "{prompt}"'
    execute_command: 'agy "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Grok ---
  grok:
    command: 'grok -p "{prompt}"'
    execute_command: 'grok "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Codex ---
  # ★respond 與 execute 用不同命令（sandbox flag 不同）
  codex:
    command: 'codex exec --sandbox read-only "{prompt}"'
    execute_command: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: "true"
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Devin ---
  devin:
    command: 'devin -p "{prompt}"'
    execute_command: 'devin "{prompt}"'
    read_only: "maybe"
    timeout: 300
    executor_capable: true
    extra_args: []

  # --- Cursor ---
  cursor:
    command: 'cursor agent -p "{prompt}"'
    execute_command: 'cursor agent --force "{prompt}"'
    read_only: "true"
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Kimi Code ---
  kimi:
    command: 'kimi -p "{prompt}"'
    execute_command: 'kimi "{prompt}"'
    read_only: "true"
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Hermes ---
  hermes:
    command: 'hermes chat -q "{prompt}"'
    execute_command: 'hermes "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Aider ---
  # ★read_only=false，不應用於 responder
  aider:
    command: 'aider --yes --model gpt-4o "{prompt}"'
    execute_command: 'aider --yes --model gpt-4o "{prompt}"'
    read_only: "false"
    timeout: 300
    executor_capable: true
    extra_args: ["--model", "gpt-4o"]

  # --- OpenCode ---
  opencode:
    command: 'opencode run "{prompt}"'
    execute_command: 'opencode "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []
```

---

## 4. 驗證規則

`ValidateConfig`（S5 實作）需檢查：

| 規則 | 條件 | 失敗行為 |
|------|------|---------|
| defaults.executor 存在 | `cfg.Defaults.Executor` 必須是 `cfg.Models` 的 key | 報錯：unknown executor model |
| defaults.responders 非空 | `cfg.Defaults.Responders` 至少 1 個 | 報錯：no responders configured |
| defaults.synthesizer 存在（若非 nil） | `*cfg.Defaults.Synthesizer` 必須是 `cfg.Models` 的 key | 報錯：unknown synthesizer model |
| 每個 model.command 含 {prompt} | `strings.Contains(cfg.Models[name].Command, "{prompt}")` | 報錯：command missing {prompt} placeholder |
| executor 模型 executor_capable=true | `cfg.Models[executor].ExecutorCapable == true` | 報錯：executor not executor_capable |
| responder 模型 read_only≠false | `cfg.Models[responder].ReadOnly != "false"` | 警告：responder is not read-only（不中斷） |

---

## 5. Config 存取方式（給 S3 / S4 用）

### ★DefaultsConfig 是 struct，不是 map

```go
// ✅ 正確（struct 欄位存取）
cfg.Defaults.Responders      // []string
cfg.Defaults.Executor        // string
cfg.Defaults.Timeout         // int
*cfg.Defaults.Synthesizer    // string（需 nil check）

// ❌ 錯誤（map 索引——會編譯失敗）
cfg.Defaults["responders"]   // compile error
```

### 遍歷 models

```go
for name, modelCfg := range cfg.Models {
	// name = "agy", "codex", ...
	// modelCfg = ModelConfig{Name: "", Command: "...", ...}
	// ★注意：YAML 解析後 modelCfg.Name 為空（yaml:"-"）
	//   factory 或 LoadConfig 需手動填入: modelCfg.Name = name
	fmt.Println(name, modelCfg.Command)
}
```

### Factory 填入 Name

```go
// LoadConfig（S5）解析 YAML 後，應遍歷填入 Name：
for name := range cfg.Models {
	model := cfg.Models[name]
	model.Name = name           // ★填入 map key 作為 Name
	cfg.Models[name] = model    // 寫回 map
}
```

> 或由 `GetAdapter`（S2 factory）填入：`cfg.Name = name`（見 03-Model-Adapters.md §4.1）。
> 兩處擇一即可，避免重複。建議在 LoadConfig 統一填入。

---

## 6. InitConfig（產生範例 config）

`InitConfig`（S5 實作）透過 `GenerateExampleConfig` 產出 `models.yaml.example` 內容，複製到目標路徑：

```bash
# 產生範例 config
openconvene init
# → 寫入 ./config/models.yaml（若已存在則拒絕覆蓋）

# 指定輸出路徑
openconvene init --path ~/.config/openconvene/models.yaml
```

> `GenerateExampleConfig` 回傳上 §3 範例的完整 YAML 字串。

---

## 7. 語言設定（`defaults.language`）

`defaults.language` 控制模型輸出語言。設定後，引擎在每次 task 前注入 `[Please respond in <lang>.]` 指令，所有 responders、synthesizer、executor 都會以該語言回應。

### 設定方式

| 方式 | 範例 | 持久化 |
|------|------|--------|
| config 檔 | `language: "zh-TW"` | ✓ 寫在 models.yaml |
| CLI flag | `--language zh-TW` | ✗ 僅當次 |
| REPL 命令 | `/language zh-TW` | ✓ 寫回 models.yaml |

### 範例值

| 值 | 效果 |
|----|------|
| `""`（空） | 不指定（模型用預設語言） |
| `"zh-TW"` | 繁體中文 |
| `"繁體中文"` | 繁體中文（完整名稱也接受） |
| `"English"` | 英文 |
| `"日本語"` | 日文 |

### 注意事項

- **只影響模型輸出** — CLI 介面（slash 命令、help text、error messages）保持英文
- **REPL 持久化** — `/language` 命令會即時寫回 config 檔，跨 session 保留
- **flag 覆寫** — `--language` flag 優先於 config 預設值，但不寫回 config
