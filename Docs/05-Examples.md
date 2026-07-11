# OpenConveneCLI — 使用範例

> **版本**：v1.0（S7 User Docs Session 產出）
> **Module**：`github.com/masteryee-labs/open-convene-cli`

本文提供各模式的實際使用範例。所有命令均與 `cmd/openconvene/main.go` 的實際 cobra flags 一致。

---

## 範例 1：Ask 模式 — 多模型分析

### 場景

你想分析某技術方案的優劣，希望多個模型同時提供不同角度的分析，再由 synthesizer 整合出一份全面的結論。

### 命令

```bash
openconvene ask "分析 Rust vs Go 的 async 效能差異，從語言設計、runtime 開銷、生態系三個維度比較" \
  --responders agy,grok,codex \
  --synthesizer agy
```

### 流程

1. **Phase 1（Fan-out）**：agy、grok、codex 三個模型同時以 read-only 模式平行回答問題
2. **Phase 2（Synthesis）**：agy 作為 synthesizer 讀取三份回應，進行推理式整合
3. **Phase 3**：跳過（ask 模式不執行）

### 預期輸出

- stdout：synthesizer 整合後的結論（涵蓋語言設計、runtime 開銷、生態系三個維度的綜合分析）
- 若加 `--verbose`：stderr 顯示三個 responder 的原始回應 + metadata（各模型耗時、成功狀態等）

### 說明

- codex 的 `read_only=true`（`--sandbox read-only` 強制 read-only），是最安全的 responder
- agy 和 grok 的 `read_only=maybe`，`-p` 模式為軟約束，但實務上不會改檔
- 三個模型的多樣性提供不同角度的分析，synthesizer 整合出比任何單一模型更全面的結論

---

## 範例 2：Code 模式 — 多模型建議 + 單一 executor 寫碼

### 場景

你需要重構一個函式，希望多個模型先提供重構建議，再由一個 executor 根據建議實際改檔。

### 命令

```bash
openconvene "重構 auth.go 的 login 函式，改善錯誤處理邏輯，提取重複的 token 驗證碼" \
  --responders agy,grok \
  --executor codex
```

### 流程

1. **Phase 1（Fan-out）**：agy、grok 平行提供重構建議
2. **Phase 2（Synthesis）**：未指定 `--synthesizer`，跳過此階段（executor 兼任 synthesizer）
3. **Phase 3（Execution）**：codex 讀取兩份建議，直接實作重構（`--sandbox workspace-write` 模式可改檔）

### 預期輸出

- stdout：executor 的執行結果摘要（改了哪些檔案、做了什麼）
- codex 實際修改 `auth.go`，改善錯誤處理邏輯

### 說明

- 未指定 `--synthesizer` 時，executor（codex）兼任 synthesizer，直接讀取兩份 responder 建議後執行
- 若想讓 synthesizer 先整合再交給 executor，可加上 `--synthesizer agy`：

```bash
openconvene "重構 auth.go 的 login 函式" \
  --responders agy,grok \
  --synthesizer agy \
  --executor codex
```

---

## 範例 3：Agent 模式 — 多模型策略 + executor agent 執行

### 場景

你需要修一個複雜的 bug，希望多個模型先提供調查策略，再由 executor agent 以 agentic 模式長時間執行修復。

### 命令

```bash
openconvene agent "修 issue #42 的記憶體洩漏：在長時間運行的 WebSocket 連線中，連線關閉後 listener goroutine 未被清理" \
  --responders agy,grok \
  --executor devin \
  --synthesizer agy
```

### 流程

1. **Phase 1（Fan-out）**：agy、grok 平行提供調查策略（可能的洩漏點、排查步驟）
2. **Phase 2（Synthesis）**：agy 整合兩份策略，形成統一的調查+修復計畫
3. **Phase 3（Execution）**：devin 以 agentic 模式執行——研究程式碼、定位洩漏點、修改檔案、驗證修復

### 預期輸出

- stdout：devin agent 的執行結果摘要（做了哪些操作、修改了哪些檔案）
- agy 的整合策略（若 `--verbose`）

### 說明

- agent 模式適合複雜多步任務：研究、檔案操作、命令執行、多步驟工作流
- devin 的 `execute_command` 使用 `--permission-mode dangerous`（S2 實測確認 `bypass` 不是有效值，`dangerous` 最接近全自動批准）
- devin 的 timeout 預設 300 秒（agent 任務通常需要較長時間），可用 `--timeout` 覆蓋

---

## 範例 4：可配置 synthesizer

### 場景

你想比較「獨立 synthesizer 整合」與「executor 兼任 synthesizer」的差異。

### 命令（獨立 synthesizer）

```bash
openconvene "設計一個分散式任務隊列的架構，考慮高可用性和水平擴展" \
  --responders agy,grok,codex \
  --synthesizer agy \
  --executor devin
```

### 流程

1. agy、grok、codex 平行提供架構建議
2. **agy 獨立整合**三份建議成一份統一的架構方案
3. devin 根據整合後的方案實作程式碼

### 命令（executor 兼任 synthesizer）

```bash
openconvene "設計一個分散式任務隊列的架構" \
  --responders agy,grok,codex \
  --executor devin
```

### 流程

1. agy、grok、codex 平行提供架構建議
2. **跳過 synthesis**——devin 直接讀取三份原始建議
3. devin 自己整合並實作

### 差異說明

| 面向 | 獨立 synthesizer | executor 兼任 |
|------|-----------------|--------------|
| **整合品質** | synthesizer 專注整合，推理更深入 | executor 邊整合邊執行，可能不夠全面 |
| **API 呼叫數** | N + 1（synthesis）+ 1（execute）= N+2 | N + 1（execute）= N+1 |
| **延遲** | 多一次 synthesis 呼叫（+5-15s） | 較低 |
| **適用場景** | 複雜任務，需要高品質整合 | 簡單任務，追求速度 |

> 若 config 中 `defaults.synthesizer` 設為 `null`，且 `--synthesizer` flag 未指定，則 executor 自動兼任 synthesizer。

---

## 範例 5：從 stdin 讀 task

### 場景

你想將檔案內容或管線輸出作為 task 傳入 OpenConveneCLI。

### 命令

```bash
# 從 echo 讀取
echo "分析這段程式碼的效能瓶頸" | openconvene ask - --responders agy

# 從檔案讀取
cat bug-report.txt | openconvene agent - --responders agy,grok --executor devin

# 從其他命令管線讀取
(echo "分析最近的 commit 歷史，找出可能的回歸點"; git log --oneline -20) | openconvene ask - --responders agy,grok
```

### 說明

- 位置參數值為 `-` 時，從 stdin 讀取全部內容作為任務描述
- 讀取後會 `TrimSpace` 去除末尾換行
- 適合將長文本（如 bug report、程式碼片段）直接 pipe 進來

---

## 範例 6：偵測系統已安裝的 CLI

### 場景

初次使用 OpenConveneCLI，想知道系統上裝了哪些 CLI，以及哪些適合當 responder 或 executor。

### 命令

```bash
openconvene detect
```

### 預期輸出

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
  hermes        hermes setup --portal
  kimi          curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash
  opencode      # see https://opencode.ai/docs/cli/
```

### 說明

- `detect` **不需要 config 檔**，直接掃描 `PATH`（`exec.LookPath`）
- 輸出 9 個 CLI 的安裝狀態、完整路徑、read_only 能力、是否適合 responder/executor
- 未安裝的 CLI 會顯示安裝指令供參考（**不自動安裝**）
- 偵測結果用來決定 `models.yaml` 要設定哪些模型——只需配置已安裝的 CLI
- Windows 上 `exec.LookPath` 會搜尋 `.exe`/`.cmd`/`.bat` 副檔名

---

## 範例 7：使用新 CLI adapter（cursor/kimi 等）

### 場景

你的系統上安裝了 cursor + kimi + codex，想用它們做 ask 模式的多模型分析。

### 前置：確認已安裝

```bash
openconvene detect
```

確認 cursor、kimi、codex 都顯示 `INSTALLED: yes`。

### 命令

```bash
openconvene ask "比較 React vs Vue 的效能差異，從渲染機制、bundle 大小、學習曲線三個維度分析" \
  --responders cursor,kimi,codex \
  --synthesizer kimi
```

### 流程

1. **Phase 1（Fan-out）**：cursor、kimi、codex 三個模型平行回答問題
   - cursor 使用 `cursor agent -p "{prompt}"`（read_only=true，無 `--force` 時為 read-only）
   - kimi 使用 `kimi -p "{prompt}"`（read_only=true，read-only ops 自動批准）
   - codex 使用 `codex exec --sandbox read-only "{prompt}"`（read_only=true，強制 read-only）
2. **Phase 2（Synthesis）**：kimi 整合三份回應
3. **Phase 3**：跳過（ask 模式）

### 預期輸出

- stdout：kimi 整合後的結論（涵蓋渲染機制、bundle 大小、學習曲線的綜合比較）

### 說明

- 9 個 CLI 都可當 responder/executor/synthesizer，依 config 的 `read_only` 和 `executor_capable` 決定適合角色
- cursor 和 kimi 的 `read_only=true`，是最安全的 responder 選擇
- 這三個模型都是 `read_only=true`，fan-out 期間不會意外改檔
- 也可用 cursor 當 synthesizer：

```bash
openconvene ask "比較 React vs Vue 的效能差異" \
  --responders cursor,kimi,codex \
  --synthesizer cursor
```

### 用新 CLI 當 executor

```bash
openconvene "為這個 React 專案新增一個 useDebounce hook" \
  --responders cursor,kimi \
  --executor cursor
```

- cursor 作為 executor 時使用 `execute_command: 'cursor agent -p --force "{prompt}"'`（加上 `--force` 允許改檔）
- 適合用 cursor 做輕量級的 code 修改

---

## 範例 8：使用 --verbose 偵錯

### 場景

某個 responder 回應異常或為空，你想查看各 responder 的原始回應和 metadata 來排查問題。

### 命令

```bash
openconvene ask "分析這段程式碼的記憶體使用模式" \
  --responders agy,grok,codex \
  --synthesizer agy \
  --verbose
```

### 預期輸出

- **stdout**：synthesizer 整合後的結論（正常輸出）
- **stderr**（`--verbose`）：
  - 各 responder 的原始回應（按模型名排序）
  - metadata（按 key 排序）：
    - `agy_success: true`
    - `agy_elapsed: 3.2s`
    - `grok_success: false`
    - `grok_error: "command not found"`
    - `codex_success: true`
    - `codex_elapsed: 5.1s`
    - `responder_count: 3`
    - `success_count: 2`
    - `synthesizer_success: true`
    - `synthesizer_elapsed: 4.0s`
    - `total_elapsed: 12.3s`

### 說明

- `--verbose` 輸出到 **stderr**，不干擾 stdout 的正常輸出（可安全 pipe 到其他命令）
- metadata 包含各階段的成功狀態、耗時、錯誤訊息
- 適合排查「某個 responder 失敗」、「synthesizer 品質差」等問題

---

## 範例 9：使用 --timeout 控制 agent 任務時限

### 場景

agent 模式的 executor（如 devin）需要較長時間執行，預設 timeout 不夠。

### 命令

```bash
openconvene agent "重構整個 API 層，從 REST 遷移到 GraphQL，包含所有 endpoint 和測試" \
  --responders agy,grok \
  --executor devin \
  --synthesizer agy \
  --timeout 600
```

### 說明

- `--timeout 600` 將每次 CLI 呼叫的 timeout 設為 600 秒（10 分鐘）
- 此 flag 覆蓋 config 中的 `defaults.timeout` 和各模型的 `timeout`
- 適合長時間執行的 agent 任務
- 若 timeout 不足，CLI 呼叫會被中斷並記錄到 metadata

---

## 範例 10：使用 --model / -m flag（對齊其他 CLI）

### 場景

你習慣 codex / agy / grok 的 `--model` flag，想用相同方式指定 executor 模型。

### 命令

```bash
# --model 是 --executor 的別名（對齊 Codex / agy / Grok CLI）
openconvene "重構 auth.go" --responders agy,grok --model codex

# -m 是 --model 的短形式
openconvene ask "分析架構" --responders agy,grok -m codex
```

### 說明

- `--model` / `-m` 是 `--executor` 的語意別名，行為完全相同
- 對齊 codex、agy、grok 等 CLI 的 `--model` flag 慣例，降低學習成本
- 適合從其他 CLI 遷移過來的使用者

---

## 範例 11：使用 --json 輸出（適合腳本/自動化）

### 場景

你想在腳本中呼叫 OpenConveneCLI 並用程式解析輸出結果。

### 命令

```bash
# JSON 輸出模式（對齊 Grok --output-format json）
openconvene ask "列出 TODO 註解" --responders agy,grok --json

# 搭配 jq 解析
openconvene ask "分析架構" --responders agy,grok --json | jq '.synthesis'
```

### 說明

- `--json` 將 ConveneResult 以 JSON 格式輸出到 stdout
- 適合在 CI/CD 管線、腳本、自動化場景中使用
- 對齊 Grok CLI 的 `--output-format json` 慣例

---

## 範例 12：互動式 REPL 模式

### 場景

你想持續與 OpenConveneCLI 互動，隨時切換模式、模型，查看使用量統計。

### 命令

```bash
# 進入 REPL（預設 code 模式）
openconvene

# 進入 REPL（ask 模式）
openconvene ask

# 進入 REPL（agent 模式）
openconvene agent
```

### REPL 操作範例

```
openconvene(code)> fix the bug in main.go     # 直接下提示詞
openconvene(code)> /mode ask                  # 切換到 ask 模式
openconvene(ask)> /model devin                # 切換 executor 模型
openconvene(ask)> /responders agy,grok,codex  # 切換 responders
openconvene(ask)> /status                     # 查看 session 狀態
openconvene(ask)> /usage                      # 查看各 CLI 使用量
openconvene(ask)> /new                        # 清除 session 重新開始
openconvene(ask)> /help                       # 顯示所有指令
openconvene(ask)> /exit                       # 離開 REPL
```

### 說明

- 不帶 task 參數時自動進入互動式 REPL（類似 codex、grok、agy、devin）
- Slash 指令對齊四大 CLI 慣例（`/model`、`/status`、`/new`、`/compact` 等）
- `/usage` 追蹤本次 session 各 CLI 的呼叫次數、成功/失敗、耗時
- 離開時自動印出 session 摘要
- 詳細 slash 指令列表見 02-Usage-Guide.md

---

## 範例 13：同一 CLI 多模型作為 responder（匿名化防偏心）

### 場景

你只有 devin CLI 安裝，但想利用 devin 支援的多個模型（glm-5.2、swe-1.7、kimi-k2.7）做 Mixture-of-Agents。裁判模型也用 glm-5.2——與其中一個 responder 相同。

### 命令

```bash
openconvene ask "分析 Go vs Rust 的 async 效能差異" \
  --responders glm-5.2,swe-1.7,kimi-k2.7 \
  --synthesizer glm-5.2 \
  --config config/test-real.yaml
```

### 為什麼不會偏心？

OpenConveneCLI 的 **responder 匿名化設計**：

- responder 回應在傳給 synthesizer 時，標記為 `Response A`、`Response B`、`Response C`，而非 model 名稱
- synthesizer（glm-5.2）看不到哪份回應來自 glm-5.2、哪份來自 swe-1.7 或 kimi-k2.7
- synthesizer 只能根據回應內容的品質來整合，無法偏袒「自己的」回應
- model 名稱僅在 `--verbose` 的 metadata 中顯示（供使用者除錯），不會進入 LLM prompt

### Config 範例

```yaml
# config/test-real.yaml
defaults:
  responders: [glm-5.2, swe-1.7, kimi-k2.7]
  synthesizer: glm-5.2

models:
  glm-5.2:
    command: 'devin --model glm-5.2 -p "{prompt}"'
    # ...
  swe-1.7:
    command: 'devin --model swe-1.7 -p "{prompt}"'
    # ...
  kimi-k2.7:
    command: 'devin --model kimi-k2.7 -p "{prompt}"'
    # ...
```

### 說明

- 同一個 CLI（devin）可以透過 `--model` flag 指定不同模型，在 config 中建立多個 entry
- 匿名化確保 synthesizer 不會因為「看到自己的名字」而偏心
- 這讓「單 CLI 多模型」的 MoA 設定成為可行的使用模式

---

## 範例 14：設定模型回應語言

### 場景

你想讓所有模型（responders、synthesizer、executor）以繁體中文回應，但 CLI 介面（slash 命令、help）保持英文。

### 方式一：CLI flag（單次使用）

```bash
openconvene ask "what is CRDT and how does it work?" --language zh-TW
```

### 方式二：REPL 中設定（持久化）

```
$ openconvene
openconvene(code)> /language zh-TW
Language set to: zh-TW
  (Model responses will be in this language. CLI commands remain in English.)

openconvene(code)> what is CRDT and how does it work?
# → 模型會以繁體中文回應

openconvene(code)> /status
=== Session Status ===
  Mode:          code
  Model (exec):  devin:glm-5.2
  Responders:    devin:glm-5.2, devin:swe-1.7, devin:kimi-k2.7
  Synthesizer:   devin:glm-5.2
  Language:      zh-TW
  Runs:          1
  Session time:  12s
```

### 方式三：config 檔設定（持久化）

```yaml
# ~/.config/openconvene/models.yaml
defaults:
  timeout: 120
  responders:
    - devin:glm-5.2
    - devin:swe-1.7
    - devin:kimi-k2.7
  executor: devin:glm-5.2
  synthesizer: devin:glm-5.2
  language: "zh-TW"              # 模型回應語言
```

### 清除語言設定

```
openconvene(code)> /language none
Language cleared — models will use their default language.
```

### 說明

- `/language` 命令設定後即時寫回 `models.yaml`，跨 session 保留
- `--language` flag 只影響當次執行，不寫回 config
- 引擎在 task 前注入 `[Please respond in zh-TW.]` 指令
- 接受的值：`zh-TW`、`繁體中文`、`English`、`日本語` 等任意字串
