# OpenConveneCLI — 疑難排解

> **版本**：v1.0（S7 User Docs Session 產出）
> **Module**：`github.com/masteryee-labs/open-convene-cli`

本文涵蓋常見問題及其解決方案。

---

## config 相關

### config 檔不存在

**症狀**：執行 `ask`、`agent`、`models`、`check` 時報錯 `failed to load config`。

**原因**：`models.yaml` 不存在於任何搜尋路徑。

**解決方案**：

```bash
# 產生範例 config
openconvene init --path ~/.config/openconvene/models.yaml

# 或使用預設路徑
openconvene init
```

然後編輯產生的 `models.yaml`，確認各 CLI 的 `command` 和 `read_only` 設定正確。

### config 路徑搜尋順序

若不指定 `--config`，OpenConveneCLI 按以下順序搜尋：

1. `--config` flag 指定的路徑
2. `OPENCONVENE_CLI_CONFIG` 環境變數
3. `~/.config/openconvene/models.yaml`
4. `./config/models.yaml`

若要固定使用某路徑，可設定環境變數：

```bash
export OPENCONVENE_CLI_CONFIG=/path/to/your/models.yaml
```

### command 不含 {prompt}

**症狀**：`check` 報錯 `ERROR: model <name> command missing {prompt} placeholder`。

**原因**：`command` 或 `execute_command` 模板中缺少 `{prompt}` 佔位符。

**解決方案**：編輯 `models.yaml`，確保 `command` 和 `execute_command` 都包含 `{prompt}`：

```yaml
# ✅ 正確
agy:
  command: 'agy -p "{prompt}"'

# ❌ 錯誤（缺少 {prompt}）
agy:
  command: 'agy -p'
```

### read_only 值錯誤

**症狀**：`check` 報錯 `ERROR: model <name> read_only must be "true", "false", or "maybe"`。

**原因**：`read_only` 欄位值不是合法值。

**解決方案**：只接受以下三個值（加引號）：

```yaml
read_only: "true"    # CLI 強制 read-only（如 codex --sandbox read-only）
read_only: "false"   # CLI 預設會改檔（如 aider）
read_only: "maybe"   # 非互動但本質 agentic（如 -p 模式）
```

> ★YAML 中 `true`/`false` 是 boolean 型別。雖然 yaml.v3 可寬容轉成 string，但加引號更明確，避免歧義。

### defaults.executor 不存在或非 executor_capable

**症狀**：`check` 報錯 `ERROR: default executor "<name>" is not executor_capable` 或 `unknown executor model`。

**解決方案**：

1. 確認 `defaults.executor` 指向的模型名存在於 `models` 中
2. 確認該模型的 `executor_capable: true`

```yaml
defaults:
  executor: codex    # 必須是 models 中已定義且 executor_capable: true 的模型

models:
  codex:
    executor_capable: true    # ← 必須為 true
```

### defaults.responders 引用未知 model

**症狀**：`check` 報錯 `ERROR: default responder "<name>" references unknown model`。

**解決方案**：確認 `defaults.responders` 中的每個名稱都存在於 `models` 中。

### ★Devin permission-mode bypass 問題

**症狀**：devin 作為 executor 時執行失敗，或 `--permission-mode bypass` 被拒絕。

**原因**：`config/models.yaml.example` 中 devin 的 `execute_command` 預設為 `devin --permission-mode bypass "{prompt}"`，但 S2 實測發現 **`bypass` 不是有效的 Devin permission mode**。有效值為：`auto`、`accept-edits`、`smart`、`dangerous`。

**解決方案**：將 `bypass` 改為 `dangerous`（最接近全自動批准所有工具的行為）：

```yaml
devin:
  command: 'devin -p "{prompt}"'
  execute_command: 'devin --permission-mode dangerous "{prompt}"'  # ← 用 dangerous
  read_only: "maybe"
```

---

## adapter 相關

### 各 CLI 呼叫失敗

**排查步驟**：

1. **先跑 `detect` 確認已安裝**：

```bash
openconvene detect
```

確認目標 CLI 顯示 `INSTALLED: yes`。

2. **用 `--verbose` 查看詳細錯誤**：

```bash
openconvene ask "test" --responders agy --verbose
```

stderr 會顯示各 responder 的 `success`、`error`、`failed` 等 metadata。

3. **手動測試 CLI 命令**：

```bash
# 直接執行 config 中的 command 模板（替換 {prompt}）
agy -p "hello"
codex exec --sandbox read-only "hello"
```

確認 CLI 本身可用、API key 已設定。

### agy 呼叫失敗

**可能原因**：
- agy 未安裝 → `curl -fsSL https://antigravity.google/cli/install.sh | bash`
- API key 未設定 → 確認 Antigravity CLI 的認證設定
- agy 版本過舊 → 更新到最新版

### codex read_only 不可靠

**說明**：codex 的 `read_only=true` 是最可靠的——`--sandbox read-only` 在引擎層面強制 read-only，CLI 本身無法繞過。

**建議**：codex 是最安全的 responder 選擇。若其他 CLI 的 read-only 行為不可靠，優先使用 codex 當 responder。

### devin timeout

**症狀**：devin 作為 executor 時超時。

**原因**：agent 任務通常需要較長時間，預設 timeout 可能不足。

**解決方案**：

```bash
# 增加 timeout（秒）
openconvene agent "..." --executor devin --timeout 600
```

或在 `models.yaml` 中為 devin 設定較長 timeout：

```yaml
devin:
  timeout: 600    # 10 分鐘
```

### grok 介面不明

**說明**：grok CLI 的介面可能隨版本更新變化。S2 實測確認 `grok -p "{prompt}"` 有效（`-p` = single-turn 模式）。

**排查**：

```bash
grok --help    # 確認 -p/--single flag 存在
grok -p "hello"  # 手動測試
```

### ★各 CLI 安裝指令（僅供參考，需手動執行）

OpenConveneCLI 不會自動安裝任何 CLI。以下安裝指令僅供參考：

| CLI | 安裝指令 |
|-----|---------|
| Devin | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| Grok | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| Codex | `npm install -g @openai/codex` |
| Antigravity (agy) | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| Cursor | `curl https://cursor.com/install -fsS \| bash` |
| Kimi Code | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| Hermes | `hermes setup --portal` |
| Aider | `python -m pip install aider-install && aider-install` |
| OpenCode | 見 https://opencode.ai/docs/cli/ |

> 安裝後重新執行 `openconvene detect` 確認安裝成功。

### aider 不支援 respond 模式

**說明**：aider 本質是 code editor，預設會改檔（`read_only=false`）。aider 的 `Respond()` 會直接回傳 error：`"aider does not support respond mode"`。

**建議**：
- **不要**將 aider 設為 responder 或 synthesizer
- aider 只適合當 executor（`execute_command: 'aider --yes --model sonnet "{prompt}"'`）
- aider 需要設定 API key 環境變數（如 `ANTHROPIC_API_KEY`），config 不含此資訊

### 未驗證的 CLI（5 個）

以下 CLI 在 S2 實測時未安裝於測試機，命令模板基於官方文件研究，未經實測驗證：

| CLI | 狀態標記 | 說明 |
|-----|---------|------|
| cursor | CURSOR_UNVERIFIED | 命令模板 `cursor agent -p "{prompt}"` 待驗證 |
| kimi | KIMI_UNVERIFIED | 命令模板 `kimi -p "{prompt}"` 待驗證 |
| hermes | HERMES_UNVERIFIED | 命令模板 `hermes chat -q "{prompt}"` 待驗證 |
| aider | AIDER_UNVERIFIED | 命令模板 `aider --yes --model sonnet "{prompt}"` 待驗證 |
| opencode | OPENCODE_UNVERIFIED | 命令模板 `opencode run "{prompt}"` 待驗證 |

> 安裝後若命令模板不正確，請編輯 `models.yaml` 中的 `command`/`execute_command` 欄位。

---

## convene 相關

### 全部 responder 失敗

**症狀**：`convene run failed: all responders failed`。

**原因**：所有 responder 的 CLI 呼叫都失敗了。

**排查步驟**：

1. 用 `--verbose` 查看各 responder 的錯誤詳情：
   ```bash
   openconvene ask "test" --responders agy,grok --verbose
   ```
   stderr 會顯示 `<model>_error` 和 `<model>_failed` 的值。

2. 確認各 CLI 是否可用：
   ```bash
   openconvene detect
   ```

3. 手動測試各 CLI 命令（替換 `{prompt}` 為實際文字）

4. 確認 API key / 認證已設定

### executor 失敗

**症狀**：`--verbose` 顯示 `executor_success: false`，或 executor 的執行結果為空。

**排查步驟**：

1. 查看 `--verbose` 的 `executor_error` 和 `executor_failed` metadata
2. 確認 executor 模型已安裝且可用
3. 確認 `executor_capable: true`
4. 確認 timeout 足夠（agent 任務可能需要更長時間）
5. 手動測試 executor 的 `execute_command`

### synthesis 品質差

**可能原因與解決方案**：

1. **synthesizer 模型能力不足** → 嘗試更強的 synthesizer（如 codex）
2. **responder 數量太少** → 增加 responder 數量（3-5 個效果較好）
3. **responder 回應品質差** → 確認 responder 的 CLI 設定正確
4. **synthesizer 同時是 responder** → 換一個不在 responders 列表中的 synthesizer（避免偏袒）
5. **task 描述太模糊** → 提供更具體的 task 描述

### ask 模式指定了 executor

**症狀**：收到警告 `WARNING: ask mode with executor specified, executor will be ignored`。

**說明**：ask 模式不執行，`--executor` 參數會被忽略。這只是警告，不影響執行。

**解決方案**：若不需要執行，移除 `--executor` flag。若需要執行，改用 code 模式（預設，`openconvene "task"`）或 agent 模式（`openconvene agent "task"`）。

---

## 常見警告

OpenConveneCLI 在執行前會進行模式+模型組合驗證，產生 WARNING（續行）或 ERROR（中止）。以下是常見警告的說明：

### "responder X is not read-only, may execute unexpectedly"

**觸發條件**：code/agent 模式下，某個 responder 的 `read_only=false`（如 aider）。

**風險**：fan-out 期間該 responder 可能執行工具或改檔，破壞 read-only 假設。

**建議**：移除 `read_only=false` 的 responder，或改用 `read_only=true`/`maybe` 的模型。

### "ask mode with executor specified, executor will be ignored"

**觸發條件**：ask 模式下指定了 `--executor`。

**說明**：ask 模式跳過 Phase 3（執行），executor 參數被忽略。

**建議**：移除 `--executor` flag，或改用 code/agent 模式。

### "synthesizer not specified, executor will self-synthesize"

**觸發條件**：`--synthesizer` 為空且 config `defaults.synthesizer` 為 `null`。

**說明**：跳過 Phase 2，executor 直接讀取 N 份原始回應後執行。這是正常行為，不是錯誤。

**影響**：executor 邊整合邊執行，整合品質可能不如獨立 synthesizer。若需更高品質，指定 `--synthesizer`。

### "only 1 responder, MoA value diminishes with single sample"

**觸發條件**：只指定了 1 個 responder。

**說明**：Mixture-of-Agents 的價值來自多模型多樣性。只有 1 個 responder 時，等同於直接問一個模型，失去了多樣性優勢。

**建議**：至少使用 2-3 個 responder。

### "synthesizer is also a responder, synthesis may be biased"

**觸發條件**：synthesizer 同時出現在 responders 列表中。

**風險**：synthesizer 在整合時可能偏袒自己的回應。

**建議**：使用不在 responders 列表中的獨立 synthesizer。

### "executor is also a responder, execution may be biased"

**觸發條件**：executor 同時出現在 responders 列表中。

**風險**：executor 在執行時可能偏袒自己的回應。

**建議**：使用不在 responders 列表中的獨立 executor。

### "synthesizer is not read-only, may execute tools during synthesis"

**觸發條件**：synthesizer 的 `read_only` 不是 `"true"`（即 `"maybe"` 或 `"false"`）。

**風險**：synthesis 期間 synthesizer 可能意外執行工具或改檔。

**建議**：使用 `read_only=true` 的模型當 synthesizer（如 codex、cursor、kimi）。

### "MoA tradeoff: N>=2 responders adds latency and cost"

**觸發條件**：使用 2 個以上 responder。

**說明**：多模型協作會增加延遲（+5-15s 等待最慢的 responder）、成本（N+2 次 API 呼叫）、降低可預測性。這是 MoA 的已知 tradeoff，不是錯誤。

---

## Go 相關

### go install 失敗

**症狀**：`go install` 報錯或編譯失敗。

**排查步驟**：

1. 確認 Go 版本 >= 1.22：
   ```bash
   go version
   ```

2. 確認 GOPATH 設定正確：
   ```bash
   go env GOPATH
   ```

3. 若從源碼安裝，先嘗試 `go build`：
   ```bash
   git clone https://github.com/masteryee-labs/open-convene-cli.git
   cd open-convene-cli
   go build ./cmd/openconvene
   ```

### 二進位找不到

**症狀**：`openconvene: command not found`。

**原因**：`GOPATH/bin` 不在 `PATH` 中。

**解決方案**：

```bash
# Linux/macOS
export PATH=$PATH:$(go env GOPATH)/bin

# Windows (PowerShell)
$env:PATH += ";$env:USERPROFILE\go\bin"

# Windows (CMD)
set PATH=%PATH%;%USERPROFILE%\go\bin
```

或將此設定加入 shell 設定檔（`~/.bashrc`、`~/.zshrc`、PowerShell `$PROFILE`）。

### 編譯錯誤

**排查步驟**：

1. 更新依賴：
   ```bash
   go mod tidy
   ```

2. 重新編譯：
   ```bash
   go build ./...
   ```

3. 若有 cobra 版本問題：
   ```bash
   # 確認 cobra 版本為 v1.8.1（與 go 1.22 相容）
   go get github.com/spf13/cobra@v1.8.1
   ```
   > ★`go get cobra@latest` 會拉取 v1.10.2，該版本要求 go 1.23+，與專案 go 1.22 不相容。

4. 若有 golang.org/x/sync 版本問題：
   ```bash
   # 確認 sync 版本為 v0.7.0（相容 go 1.22）
   go get golang.org/x/sync@v0.7.0
   ```
   > ★`go get golang.org/x/sync@latest` 可能拉取需要 go 1.25+ 的版本。

### 跨平台編譯

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o openconvene-linux ./cmd/openconvene

# macOS
GOOS=darwin GOARCH=amd64 go build -o openconvene-darwin ./cmd/openconvene

# Windows
GOOS=windows GOARCH=amd64 go build -o openconvene.exe ./cmd/openconvene
```

> Go 的交叉編譯零配置，不需要 CI 矩陣。產出的是靜態二進位，無 runtime 依賴。

### Windows process group 限制

**說明**：Windows 上殺進程使用 `taskkill /T /F /PID`，是 best-effort 方式。若 parent process（cmd.exe）已退出，`taskkill /T` 可能找不到孤兒子進程。Unix 版本使用 `Setpgid + kill(-pid, SIGKILL)` 更可靠。

**影響**：Windows 上 timeout 殺進程時，極少數情況下子進程可能殘留。這是 Windows 的已知限制，不影響正常使用。
