# OpenConveneCLI 初版設計 Session Plan

<!-- 版本：v1.0 | 2026-07-11 | Go 語言版，8 個子 Session（S0-S7） -->
> 對應指揮官：`Docs/Agents/指揮官/Orchestrator_Prompt.md`

---

## 一、管線概述

本管線建造 **OpenConveneCLI**——一個獨立 Go CLI，實現多模型協作（概念對齊 OpenRouter Fusion 與 Mixture-of-Agents 研究 arXiv:2406.04692）：

- N 個 responder 模型平行回答同一問題（read-only，不執行）
- synthesizer 整合 N 份回應（可選；不指定則 executor 兼任）
- executor 模型根據整合結果執行（research=印結論 / code=寫碼 / agent=跑 agent）

### 為什麼選 Go

| 維度 | Go 的優勢 |
|------|----------|
| 分發 | 單一靜態二進位，`go install` 即用，無 runtime 依賴 |
| 並發 | goroutines 天生 fan-out，`sync.WaitGroup` / `errgroup` 極簡潔 |
| 啟動 | ~5ms，CLI 工具體感好 |
| 型別 | 靜態型別，重構安全 |
| 維護 | 語法簡潔，門檻低 |
| ★跨平台 | Go 跨平台編譯（GOOS/GOARCH），同一份程式碼支援 Windows/Linux/macOS |

### 跨平台支援

- **Go 編譯**：`GOOS=windows/linux/darwin` 跨平台編譯，同一份程式碼
- **subprocess**：`os/exec` 跨平台（Windows 搜 .exe / Linux/Mac 搜 PATH）
- **CLI 偵測**：`exec.LookPath()` 跨平台偵測已安裝的 CLI
- **config 路徑**：`os.UserHomeDir()` 跨平台取得 home directory
- **S0 安裝**：Go 安裝步驟支援 Windows（MSI/zip）/ Linux（tar.gz）/ macOS（Homebrew/tar.gz）

### 技術選型

- CLI 框架：cobra（github.com/spf13/cobra）
- YAML 解析：gopkg.in/yaml.v3
- 並發：goroutines + golang.org/x/sync/errgroup
- subprocess：os/exec + context.WithTimeout
- 測試：go test + testify（github.com/stretchr/testify）
- Go 版本：>= 1.21

## 二、Session 總表

| Session | 功能 | 類型 | 依賴 | 並行限制 |
|---------|------|------|------|---------|
| S0 | 部署 Agent Harness + 安裝 Go | D | — | exclusive |
| S1 | 架構文件 + 專案骨架（go.mod + models.go） | A | S0 | normal |
| S2 | Model Adapters（os/exec + interface） | C | S1 | normal |
| S3 | Convene Core（goroutines fan-out） | C | S2 | normal |
| S4 | CLI Interface（cobra） | C | S3,S5 | normal |
| S5 | Config System（yaml.v3） | C | S1 | normal |
| S6 | Tests（go test + testify） | T | S2,S3,S4,S5 | normal |
| S7 | User Docs（go install 範例） | DOC | S4,S5 | normal |

類型：D=Deploy A=Architecture C=Code T=Test DOC=Docs

## 三、依賴圖

```
S0 (exclusive)
  └── S1（文件 + go.mod + internal/config/models.go 骨架）
        ├── S2 ── S3 ── S4（依賴 S3 + S5）
        └── S5 ──────────┘
                      ├── S6 (依賴 S2+S3+S4+S5)
                      └── S7 (依賴 S4+S5)
```

## 四、並行策略

| Wave | 可並行 Session | 說明 |
|------|---------------|------|
| 1 | S0 | exclusive，獨佔（部署 harness + 安裝 Go） |
| 2 | S1 | 架構文件 + 專案骨架，依賴 S0 |
| 3 | S2 + S5 | adapter 層 + config 系統，互不依賴（都依賴 S1） |
| 4 | S3 | convene core，依賴 S2 |
| 5 | S4 | CLI 介面，依賴 S3 + S5 |
| 6 | S6 + S7 | 測試 + 文件，依賴 S2+S3+S4+S5 / S4+S5 |

### 可並行組合示例（MAX_CONCURRENT=2）

- ✅ S2 + S5 = 2（S2 依賴 S1✓，S5 依賴 S1✓，互不依賴）
- ✅ S3 + S5 = 2（S3 依賴 S2✓，S5 依賴 S1✓）
- ❌ S3 + S2 = S3 依賴 S2，不可同波
- ❌ S4 + S5 = S4 依賴 S5，不可同波

## 五、核心設計原則（貫穿所有 Session）

1. 並行 fan-out 不線性放大延遲（總延遲 ≈ max，不是 sum）
2. synthesis 是推理式整合，不是多數投票
3. responder 必須 read-only（fan-out 階段禁止 side-effect）
4. 容錯是核心價值（單個 responder 失敗不中斷，N-1 個仍可整合）
5. 權衡：延遲↑ 成本↑ 可預測性↓ 換取品質↑ 盲點↓

## 六、支援的模型 CLI adapter

★支援偵測 9 個 CLI（依官方文件研究，不自動安裝）：

| CLI | 偵測名 | 非互動模式 | Read-Only | 安裝指令（僅顯示） |
|-----|--------|-----------|-----------|------------------|
| Devin | devin | devin -p "{prompt}" | maybe | curl -fsSL https://cli.devin.ai/install.sh \| bash |
| Grok | grok | grok -p "{prompt}" | maybe | curl -fsSL https://x.ai/cli/install.sh \| bash |
| Codex | codex | codex exec "{prompt}" | true | npm install -g @openai/codex |
| Antigravity | agy | agy -p "{prompt}" | maybe | curl -fsSL https://antigravity.google/cli/install.sh \| bash |
| Cursor | cursor | cursor agent -p "{prompt}" | true | curl https://cursor.com/install -fsS \| bash |
| Kimi Code | kimi | kimi -p "{prompt}" | true | curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash |
| Hermes | hermes | hermes chat -q "{prompt}" | maybe | hermes setup --portal |
| Aider | aider | aider --yes --model {model} "{prompt}" | false | python -m pip install aider-install && aider-install |
| OpenCode | opencode | opencode run "{prompt}" | maybe | 見 opencode.ai/docs/cli/ |

★`openconvene-cli detect` 會偵測這 9 個 CLI 是否已安裝，未安裝的顯示安裝指令供使用者參考。

## 七、CLI 介面範例

```bash
# 偵測系統已安裝的 CLI（不需 config）
openconvene-cli detect

# 執行多模型協作
openconvene-cli run --task "..." --mode {research|code|agent} \
  --responders agy,grok,devin \
  --executor codex \
  [--synthesizer agy]

# 列出 config 中定義的模型
openconvene-cli list-models

# config 管理
openconvene-cli config init [--path ...]
openconvene-cli config validate [--config ...]
```
