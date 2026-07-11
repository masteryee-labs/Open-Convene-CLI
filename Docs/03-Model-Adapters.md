# 03 — Model Adapters 設計

> **版本**：v1.0（S1 架構 Session 產出）
> **套件**：`internal/adapter`
> **實作 Session**：S2

---

## 1. Adapter 設計概覽

每個 adapter 是 `internal/adapter/` 下的一個 `.go` 檔，實作 `Adapter` interface（見 `01-Architecture.md` §4）。adapter 的職責：

1. **組裝命令**：將 `ModelConfig.Command`（或 `ExecuteCommand`）模板中的 `{prompt}` 替換為實際 prompt，產出完整 shell 命令字串。
2. **執行 CLI**：透過 `RunCommand`（shell 執行器）跑命令，捕捉 stdout / stderr / return code。
3. **回傳結果**：包裝成 `AdapterResult`，上層（ConveneEngine）從 `.Stdout` 取純文字回應。

### 共用基底

所有 adapter 共用一個 `BaseAdapter` struct（內嵌 `ModelConfig`），避免重複程式碼：

```go
// BaseAdapter 提供所有 adapter 共用的邏輯。
type BaseAdapter struct {
	Config ModelConfig
}

// GetCommand 組裝 CLI 命令字串。mode = "respond" | "execute"
func (b *BaseAdapter) GetCommand(prompt string, mode string) string {
	tmpl := b.Config.Command
	if mode == "execute" && b.Config.ExecuteCommand != "" {
		tmpl = b.Config.ExecuteCommand
	}
	// 替換 {prompt} 佔位符（shell 跳脫）
	return strings.ReplaceAll(tmpl, "{prompt}", shellEscape(prompt))
}

// SupportsReadOnly 回傳此 CLI 是否真正支援 read-only。
func (b *BaseAdapter) SupportsReadOnly() bool {
	return b.Config.ReadOnly == "true"
}
```

各 adapter 只需覆寫 `Respond` / `Execute`（差異在 timeout 處理或特殊後處理）：

```go
// AgyAdapter — Antigravity CLI
type AgyAdapter struct {
	BaseAdapter
}

func (a *AgyAdapter) Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error) {
	cmd := a.GetCommand(prompt, "respond")
	return RunCommand(ctx, cmd, timeout)
}

func (a *AgyAdapter) Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error) {
	cmd := a.GetCommand(prompt, "execute")
	return RunCommand(ctx, cmd, timeout)
}
```

---

## 2. read_only 能力矩陣

| CLI | 偵測名 | read_only | 原因 |
|-----|--------|-----------|------|
| Antigravity | `agy` | `maybe` | `-p` 非互動模式但本質 agentic，不保證不執行工具 |
| Codex | `codex` | `true` | `--sandbox read-only` 確保 read-only（respond 模式專用 flag） |
| Devin | `devin` | `maybe` | `-p` print mode 非互動但本質 agentic |
| Grok | `grok` | `maybe` | `-p` 非互動但本質 agentic |
| Cursor | `cursor` | `true` | 無 `--force` 時為 read-only（不自動改檔） |
| Kimi Code | `kimi` | `true` | read-only ops 自動批准，不修改檔案 |
| Hermes | `hermes` | `maybe` | `chat -q` single query 但本質 agentic |
| Aider | `aider` | `false` | 本質是 code editor，預設會改檔 |
| OpenCode | `opencode` | `maybe` | `run` 子命令非互動但本質 agentic |

### read_only 值語義

| 值 | 語義 | respond 模式行為 |
|----|------|-----------------|
| `true` | CLI 強制 read-only | 安全用於 responder（不會改檔） |
| `false` | CLI 預設會改檔 | ★不應用於 responder；僅用於 executor |
| `maybe` | CLI 有非互動模式但本質 agentic | 可用於 responder，但需注意可能執行工具；ConveneEngine 可加警告 |

> ★read_only 值已從官方文件研究預填。S2 實作時由 SubAgent 實測驗證各 CLI 的實際行為，必要時修正 config。

---

## 3. 支援的 9 個 CLI 清單

### 3.1 總覽表

| # | CLI | 偵測名 | 非互動模式命令 | Read-Only | 安裝指令（僅顯示，不自動安裝） |
|---|-----|--------|---------------|-----------|---------------------------|
| 1 | Devin | `devin` | `devin -p "{prompt}"` | maybe | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| 2 | Grok | `grok` | `grok -p "{prompt}"` | maybe | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| 3 | Codex | `codex` | `codex exec --sandbox read-only "{prompt}"` | true | `npm install -g @openai/codex` |
| 4 | Antigravity | `agy` | `agy -p "{prompt}"` | maybe | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| 5 | Cursor | `cursor` | `cursor agent -p "{prompt}"` | true | `curl https://cursor.com/install -fsS \| bash` |
| 6 | Kimi Code | `kimi` | `kimi -p "{prompt}"` | true | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| 7 | Hermes | `hermes` | `hermes chat -q "{prompt}"` | maybe | `hermes setup --portal` |
| 8 | Aider | `aider` | `aider --yes --model {model} "{prompt}"` | false | `python -m pip install aider-install && aider-install` |
| 9 | OpenCode | `opencode` | `opencode run "{prompt}"` | maybe | 見 opencode.ai/docs/cli/ |

> ★安裝指令僅供參考顯示，OpenConveneCLI 不自動安裝任何 CLI。使用者需自行安裝所需 CLI。
> `detect` 命令偵測到未安裝的 CLI 時，會顯示對應安裝指令提示。

---

### 3.2 各 CLI 詳細說明

#### 1. Devin (`devin`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `devin` |
| respond 命令模板 | `devin -p "{prompt}"` |
| execute 命令模板 | `devin "{prompt}"`（agentic 模式，無 `-p`） |
| read_only | `maybe` |
| executor_capable | `true` |
| 呼叫方式 | `-p` flag = print mode（非互動，輸出純文字到 stdout）；無 `-p` = agentic session |
| 限制 | 本質是 agentic AI，`-p` 模式仍可能觸發工具；需登入 Devin 帳號 |
| 安裝 | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| adapter 檔 | `internal/adapter/devin.go` |

#### 2. Grok (`grok`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `grok` |
| respond 命令模板 | `grok -p "{prompt}"` |
| execute 命令模板 | `grok "{prompt}"`（agentic 模式） |
| read_only | `maybe` |
| executor_capable | `true` |
| 呼叫方式 | `-p` flag = 非互動查詢模式；預設模式為 agentic |
| 限制 | 本質 agentic，`-p` 模式非嚴格 read-only |
| 安裝 | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| adapter 檔 | `internal/adapter/grok.go` |

#### 3. Codex (`codex`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `codex` |
| respond 命令模板 | `codex exec --sandbox read-only "{prompt}"` |
| execute 命令模板 | `codex exec --sandbox workspace-write "{prompt}"` |
| read_only | `true` |
| executor_capable | `true` |
| 呼叫方式 | `exec` 子命令 = 非互動執行；`--sandbox read-only` 確保不改檔；`--sandbox workspace-write` 允許寫入 |
| 限制 | 需 OpenAI API key；npm 全域安裝 |
| 安裝 | `npm install -g @openai/codex` |
| adapter 檔 | `internal/adapter/codex.go` |
| ★特殊 | respond 與 execute 用不同命令模板（sandbox flag 不同），是唯一必須設 `execute_command` 的 adapter |

#### 4. Antigravity (`agy`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `agy` |
| respond 命令模板 | `agy -p "{prompt}"` |
| execute 命令模板 | `agy "{prompt}"`（agentic 模式） |
| read_only | `maybe` |
| executor_capable | `true` |
| 呼叫方式 | `-p` flag = 非互動 print mode；無 `-p` = agentic session |
| 限制 | 本質 agentic，`-p` 非嚴格 read-only |
| 安裝 | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| adapter 檔 | `internal/adapter/agy.go` |

#### 5. Cursor (`cursor`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `cursor` |
| respond 命令模板 | `cursor agent -p "{prompt}"` |
| execute 命令模板 | `cursor agent --force "{prompt}"`（`--force` 允許自動改檔） |
| read_only | `true` |
| executor_capable | `true` |
| 呼叫方式 | `agent` 子命令 = 非互動 agent 模式；無 `--force` 時 read-only；`--force` 時自動執行 |
| 限制 | 需 Cursor 帳號 / 授權 |
| 安裝 | `curl https://cursor.com/install -fsS \| bash` |
| adapter 檔 | `internal/adapter/cursor.go` |

#### 6. Kimi Code (`kimi`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `kimi` |
| respond 命令模板 | `kimi -p "{prompt}"` |
| execute 命令模板 | `kimi "{prompt}"`（agentic 模式） |
| read_only | `true` |
| executor_capable | `true` |
| 呼叫方式 | `-p` flag = 非互動；read-only ops 自動批准 |
| 限制 | 需 Kimi 授權 |
| 安裝 | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| adapter 檔 | `internal/adapter/kimi.go` |

#### 7. Hermes (`hermes`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `hermes` |
| respond 命令模板 | `hermes chat -q "{prompt}"` |
| execute 命令模板 | `hermes "{prompt}"`（agentic 模式） |
| read_only | `maybe` |
| executor_capable | `true` |
| 呼叫方式 | `chat` 子命令 + `-q` flag = single query（非互動）；無 `-q` = agentic |
| 限制 | 需透過 `hermes setup --portal` 設定入口 |
| 安裝 | `hermes setup --portal` |
| adapter 檔 | `internal/adapter/hermes.go` |

#### 8. Aider (`aider`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `aider` |
| respond 命令模板 | `aider --yes --model {model} "{prompt}"` |
| execute 命令模板 | 同 respond（aider 本質就是執行） |
| read_only | `false` |
| executor_capable | `true` |
| 呼叫方式 | `--yes` = 自動同意所有操作（不互動詢問）；`--model` 指定 LLM |
| 限制 | ★read_only=false——本質是 code editor，預設會改檔。不應用於 responder（除非使用者明知風險） |
| 安裝 | `python -m pip install aider-install && aider-install` |
| adapter 檔 | `internal/adapter/aider.go` |
| ★特殊 | `--model` 參數透過 `extra_args` 配置，如 `extra_args: ["--model", "gpt-4o"]` |

#### 9. OpenCode (`opencode`)

| 項目 | 內容 |
|------|------|
| 偵測名 | `opencode` |
| respond 命令模板 | `opencode run "{prompt}"` |
| execute 命令模板 | `opencode "{prompt}"`（agentic 模式） |
| read_only | `maybe` |
| executor_capable | `true` |
| 呼叫方式 | `run` 子命令 = 非互動執行單一 prompt；無 `run` = agentic |
| 限制 | 本質 agentic，`run` 子命令非嚴格 read-only |
| 安裝 | 見 opencode.ai/docs/cli/ |
| adapter 檔 | `internal/adapter/opencode.go` |

---

## 4. Factory 與 Detect

### 4.1 GetAdapter（factory.go）

```go
// GetAdapter 根據模型名建立對應的 adapter 實例。
// name 必須是 9 個支援的 CLI 偵測名之一。
// cfg 的 Name 欄位由 factory 從 map key 填入。
func GetAdapter(name string, cfg config.ModelConfig) (Adapter, error) {
	cfg.Name = name  // ★factory 填入 Name（yaml:"-" 不解析）
	base := BaseAdapter{Config: cfg}

	switch name {
	case "agy":
		return &AgyAdapter{base}, nil
	case "codex":
		return &CodexAdapter{base}, nil
	case "devin":
		return &DevinAdapter{base}, nil
	case "grok":
		return &GrokAdapter{base}, nil
	case "cursor":
		return &CursorAdapter{base}, nil
	case "kimi":
		return &KimiAdapter{base}, nil
	case "hermes":
		return &HermesAdapter{base}, nil
	case "aider":
		return &AiderAdapter{base}, nil
	case "opencode":
		return &OpenCodeAdapter{base}, nil
	default:
		return nil, fmt.Errorf("unknown adapter: %s", name)
	}
}
```

### 4.2 DetectAvailableAdapters（detect.go）

```go
var supportedCLIs = []string{
	"devin", "grok", "codex", "agy", "cursor",
	"kimi", "hermes", "aider", "opencode",
}

// DetectAvailableAdapters 偵測本機已安裝的 CLI。
// 回傳 {cliName: isInstalled} map。
func DetectAvailableAdapters() map[string]bool {
	available := make(map[string]bool)
	for _, name := range supportedCLIs {
		_, err := exec.LookPath(name)
		available[name] = (err == nil)
	}
	return available
}
```

> ★跨平台：`exec.LookPath` 在 Windows 自動找 `.exe`/`.cmd`/`.bat`，在 Linux/macOS 找 PATH 中可執行檔。

---

## 5. RunCommand（shell 執行器）

> 位於 `internal/adapter/adapter.go`（S2 實作）

```go
// RunCommand 透過 shell 執行完整命令字串，回傳 AdapterResult。
// timeout 秒數透過 context.WithTimeout 控制；超時自動 kill 子進程。
func RunCommand(ctx context.Context, command string, timeout int) (AdapterResult, error) {
	// 設定 timeout context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 跨平台 shell 選擇
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	returnCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			returnCode = exitErr.ExitCode()
		} else {
			returnCode = -1
		}
	}

	return AdapterResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ReturnCode: returnCode,
		Success:    returnCode == 0 && stdout.Len() > 0,
	}, err
}
```

---

## 6. shellEscape（prompt 跳脫）

GetCommand 替換 `{prompt}` 時需對 prompt 做 shell 跳脫，避免命令注入：

```go
// shellEscape 對 prompt 做雙引號跳脫，防止 shell 注入。
func shellEscape(s string) string {
	// 將 " 替換為 \"，將 $ 替換為 \$，將 ` 替換為 \`
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `$`, `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}
```

> ★Windows cmd.exe 跳脫規則與 sh 不同。S2 實作時需按 `runtime.GOOS` 分支處理，或統一用單引號（Windows cmd 不支援單引號，需另案處理）。
> 此為已知跨平台複雜點，S2 需實測驗證。

---

## 7. Adapter 檔案清單

| 檔案 | 內容 | 實作 Session |
|------|------|-------------|
| `adapter.go` | Adapter interface + AdapterResult + BaseAdapter + RunCommand + shellEscape | S2 |
| `agy.go` | AgyAdapter | S2 |
| `codex.go` | CodexAdapter | S2 |
| `devin.go` | DevinAdapter | S2 |
| `grok.go` | GrokAdapter | S2 |
| `cursor.go` | CursorAdapter | S2 |
| `kimi.go` | KimiAdapter | S2 |
| `hermes.go` | HermesAdapter | S2 |
| `aider.go` | AiderAdapter | S2 |
| `opencode.go` | OpenCodeAdapter | S2 |
| `factory.go` | GetAdapter | S2 |
| `detect.go` | DetectAvailableAdapters | S2 |
