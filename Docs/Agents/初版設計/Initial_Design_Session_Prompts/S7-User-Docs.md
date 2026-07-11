# S7 — User Docs（中空提示詞）

> 類型：DOC (Docs) | 依賴：S4,S5 | 並行限制：normal
> 本 Session 寫使用者文件。中空模板——文件結構固定，具體內容依實作結果填。
> ★Go 專案：安裝/範例用 go install / go build，不是 pip install。

---

## === S7 PROMPT（複製以下 code block 內容）===

```
你是 S7 User Docs SubAgent。你的任務是寫 OpenConveneCLI 的使用者文件。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/00-Overview.md") → 概覽
2. read("Docs/01-Architecture.md") → Go 架構
3. read(".agent/handoff/S2.md") → adapter 實測結果（read_only 矩陣）
4. read(".agent/handoff/S3.md") → 三種模式行為差異 + prompt 模板
5. read(".agent/handoff/S4.md") → CLI 完整參數（cobra flags）
6. read(".agent/handoff/S5.md") → config 系統 + models.yaml.example
7. read("cmd/openconvene-cli/main.go") → 確認實際 CLI 參數
8. read("config/models.yaml.example") → 確認實際 config 格式

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

Docs/
├── 02-Usage-Guide.md       # 完整使用指南
├── 05-Examples.md          # 各模式使用範例
├── 06-Troubleshooting.md   # 常見問題
README.md                   # 專案 README（若 S0 的 harness 未生成或需更新）

═══════════════════════════════════════════════════════════════
【02-Usage-Guide.md — 結構（中空）】
═══════════════════════════════════════════════════════════════

```markdown
# OpenConveneCLI — 使用指南

## 安裝

<INSTALL_SECTION>
← 填入（Go 安裝方式）：
  - ★從源碼安裝：
    git clone https://github.com/masteryee-labs/open-convene-cli.git
    cd open-convene-cli
    go install ./cmd/openconvene-cli
  - ★或直接 go install：
    go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene-cli@latest
  - ★或從 release 下載預編譯二進位：
    從 GitHub Releases 頁下載對應平台的二進位，放入 PATH
  - 前置條件：Go 1.22+（若從源碼安裝）、各 CLI 工具已安裝（★9 個可選：agy/codex/devin/grok/cursor/kimi/hermes/aider/opencode，至少裝 1 個）
  - 驗證安裝：openconvene-cli --help

## 初始設定

<SETUP_SECTION>
← 填入：
  - ★先偵測系統已安裝哪些 CLI：
    openconvene-cli detect
    → 顯示 9 個 CLI（agy/codex/devin/grok/cursor/kimi/hermes/aider/opencode）的安裝狀態 + read_only 能力 + 適合角色
    → 未安裝的 CLI 顯示安裝指令供參考
  - openconvene-cli config init --path ~/.config/openconvene-cli/models.yaml
  - 編輯 models.yaml 填入各 CLI 的 command + read_only（參考 detect 結果）
  - openconvene-cli config validate 確認

## 基本用法

<BASIC_USAGE_SECTION>
← 填入：
  - openconvene-cli run --task "..." --mode research --responders agy,grok
  - openconvene-cli run --task "..." --mode code --responders agy,grok --executor codex
  - openconvene-cli run --task "..." --mode agent --responders agy,grok --executor devin --synthesizer agy

## 完整參數參考

<ARGS_REFERENCE_SECTION>
← 填入：從 main.go 整理所有 cobra flags 表格

## 模式說明

| 模式 | 流程 | 適用 |
|------|------|------|
| research | N responder → synthesizer → 印結論 | 研究分析 |
| code | N responder → synthesizer（可選）→ executor 寫碼 | 寫碼改檔 |
| agent | N responder → synthesizer → executor agent | Agent 任務 |

<MODE_DETAILS_SECTION>
← 填入各模式的詳細行為說明

## 模型角色

| 角色 | 職責 | 適合的模型 |
|------|------|-----------|
| responder | 平行回答問題（read-only） | read_only=true 的模型 |
| synthesizer | 整合 N 份回應 | read_only=true 的模型 |
| executor | 執行（寫碼/agent） | executor_capable=true 的模型 |

<MODEL_ROLE_DETAILS_SECTION>
← 填入各角色的選擇建議

## Config 説明

<CONFIG_SECTION>
← 填入：models.yaml 各欄位説明，引用 Docs/04-Configuration.md

## 為什麼選 Go

<WHY_GO_SECTION>
← 填入：
  - 單一靜態二進位分發（無 runtime 依賴）
  - goroutines 天生並行 fan-out
  - 快速啟動（~5ms）
  - 靜態型別，重構安全
```

═══════════════════════════════════════════════════════════════
【05-Examples.md — 結構（中空）】
═══════════════════════════════════════════════════════════════

```markdown
# OpenConveneCLI — 使用範例

## 範例 1：Research 模式 — 多模型分析

<EXAMPLE_1>
← 填入：
  - 場景：分析某技術方案的優劣
  - 命令：openconvene-cli run --task "分析 Rust vs Go 的 async 效能" --mode research --responders agy,grok,codex --synthesizer agy
  - 預期輸出：3 個模型的分析 + synthesizer 整合結論

## 範例 2：Code 模式 — 多模型建議 + 單一 executor 寫碼

<EXAMPLE_2>
← 填入：
  - 場景：重構一個函式
  - 命令：openconvene-cli run --task "重構 auth.go 的 login 函式" --mode code --responders agy,grok --executor codex
  - 預期輸出：2 個模型的重構建議 + codex 實際改檔

## 範例 3：Agent 模式 — 多模型策略 + executor agent 執行

<EXAMPLE_3>
← 填入：
  - 場景：修一個 bug
  - 命令：openconvene-cli run --task "修 issue #42 的記憶體洩漏" --mode agent --responders agy,grok --executor devin --synthesizer agy
  - 預期輸出：2 個模型的調查策略 + agy 整合 + devin agent 執行修復

## 範例 4：可配置 synthesizer

<EXAMPLE_4>
← 填入：
  - 場景：用獨立 synthesizer 整合
  - 命令：openconvene-cli run --task "..." --mode code --responders agy,grok,codex --synthesizer agy --executor devin
  - 説明 synthesizer 獨立 vs executor 兼任的差異

## 範例 5：從 stdin 讀 task

<EXAMPLE_5>
← 填入：
  - echo "分析這段程式碼" | openconvene-cli run --task - --mode research --responders agy

## 範例 6：偵測系統已安裝的 CLI

<EXAMPLE_6>
← 填入：
  - 場景：初次使用，想知道系統上裝了哪些 CLI
  - 命令：openconvene-cli detect
  - 預期輸出：表格顯示 9 個 CLI（devin/grok/codex/agy/cursor/kimi/hermes/aider/opencode）的安裝狀態 + read_only 能力 + 適合角色
  - 未安裝的 CLI 會顯示安裝指令供參考（不自動安裝）
  - 說明：detect 不需 config，直接掃描 PATH；結果用來決定 models.yaml 要設定哪些模型

## 範例 7：使用新 CLI adapter（cursor/kimi 等）

<EXAMPLE_7>
← 填入：
  - 場景：系統上裝了 cursor + kimi + codex，用它們做 research
  - 命令：openconvene-cli run --task "比較 React vs Vue 的效能" --mode research --responders cursor,kimi,codex --synthesizer kimi
  - 預期輸出：3 個模型的分析 + kimi 整合結論
  - 說明：9 個 CLI 都可當 responder/executor/synthesizer，依 config 的 read_only 和 executor_capable 決定適合角色
```

═══════════════════════════════════════════════════════════════
【06-Troubleshooting.md — 結構（中空）】
═══════════════════════════════════════════════════════════════

```markdown
# OpenConveneCLI — 疑難排解

## config 相關

<TROUBLESHOOT_CONFIG>
← 填入：
  - config 不存在 → openconvene-cli config init
  - command 不含 {prompt} → 編輯 models.yaml
  - read_only 值錯誤 → 只接受 true/false/maybe

## adapter 相關

<TROUBLESHOOT_ADAPTER>
← 填入：
  - 各 CLI 呼叫失敗 → 先跑 `openconvene-cli detect` 確認已安裝
  - agy 呼叫失敗 → 確認 agy 已安裝 + API key
  - codex read_only 不可靠 → 避免用 codex 當 responder
  - devin timeout → 增加 --timeout
  - grok 介面不明 → 確認 Grok CLI 版本
  - ★各 CLI 安裝指令（僅供參考，需手動執行）：
    - Devin:    curl -fsSL https://cli.devin.ai/install.sh | bash
    - Grok:     curl -fsSL https://x.ai/cli/install.sh | bash
    - Codex:    npm install -g @openai/codex
    - Antigravity: curl -fsSL https://antigravity.google/cli/install.sh | bash
    - Cursor:   curl https://cursor.com/install -fsS | bash
    - Kimi Code: curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash
    - Hermes:   hermes setup --portal
    - Aider:    python -m pip install aider-install && aider-install
    - OpenCode: 見 https://opencode.ai/docs/cli/

## convene 相關

<TROUBLESHOOT_CONVENE>
← 填入：
  - 全部 responder 失敗 → 檢查各 CLI 是否可用
  - executor 失敗 → 查看 --verbose 的詳細錯誤
  - synthesis 品質差 → 嘗試不同 synthesizer 或增加 responder 數量

## 常見警告

<COMMON_WARNINGS>
← 填入：
  - "responder X is not read-only, may execute unexpectedly"
  - "research mode with executor specified, executor will be ignored"
  - "synthesizer not specified, executor will self-synthesize"

## Go 相關

<TROUBLESHOOT_GO>
← 填入：
  - go install 失敗 → 確認 Go 1.22+ 已安裝
  - 二進位找不到 → 確認 GOPATH/bin 在 PATH 中
  - 編譯錯誤 → go mod tidy + go build ./...
```

═══════════════════════════════════════════════════════════════
【README.md — 結構（中空）】
═══════════════════════════════════════════════════════════════

若 S0 部署的 harness 已生成 README.md，則更新它；否則新建。

```markdown
# OpenConveneCLI

<ONE_LINE_DESCRIPTION>
← 一句話描述

## Quick Start

<QUICK_START>
← 3 步快速上手（Go 版）：
  1. go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene-cli@latest
  2. openconvene-cli detect  # 偵測系統已安裝的 9 個 CLI adapter
  3. openconvene-cli config init --path ~/.config/openconvene-cli/models.yaml
  4. openconvene-cli run --task "你的問題" --mode research --responders agy,grok
  ★responders 可用任何已安裝的 CLI：agy/codex/devin/grok/cursor/kimi/hermes/aider/opencode

## What it does

<WHAT_IT_DOES>
← 引用 Docs/00-Overview.md 的概念説明

## Why Go

<WHY_GO>
← 一段話：為什麼選 Go（單二進位、goroutines、快啟動）

## Documentation

- [Overview](Docs/00-Overview.md)
- [Architecture](Docs/01-Architecture.md)
- [Usage Guide](Docs/02-Usage-Guide.md)
- [Examples](Docs/05-Examples.md)
- [Configuration](Docs/04-Configuration.md)
- [Troubleshooting](Docs/06-Troubleshooting.md)

## Build from Source

<BUILD_FROM_SOURCE>
← 填入：
  git clone https://github.com/masteryee-labs/open-convene-cli.git
  cd open-convene-cli
  go build -o openconvene-cli ./cmd/openconvene-cli

## License

MIT
```

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. 所有 <PLACEHOLDER> 必須填入實際內容（從 handoff + 程式碼整理）
2. 範例命令必須與 cmd/openconvene-cli/main.go 的實際 cobra flags 一致
3. config 説明必須與 models.yaml.example 一致
4. read_only 矩陣必須與 S2 handoff 的實測結果一致
5. 文件用 Markdown，程式碼範例用 fenced code block
6. ★安裝指令用 Go 生態（go install / go build），不是 pip
7. 中文撰寫（與 Yee-World-Life 風格一致）

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- Docs/02-Usage-Guide.md 存在且含安裝/設定/用法/參數/模式/config 六段 + 為什麼選 Go
- Docs/05-Examples.md 存在且含 ≥7 個範例（含 detect 命令範例 + 新 CLI adapter 範例）
- Docs/06-Troubleshooting.md 存在且含 config/adapter/convene/警告/Go 五段
- README.md 存在且含 Quick Start + 文件連結 + Build from Source
- 所有 <PLACEHOLDER> 已填入
- 範例命令與 main.go 參數一致
- git commit: docs(S7): write user guide, examples, troubleshooting, README for Go CLI

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S7.md，內容：
- 產出文件清單
- 各文件的段落摘要
- git commit hash
```
