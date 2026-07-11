# S4 — CLI Interface（中空提示詞）

> 類型：C (Code) | 依賴：S3,S5 | 並行限制：normal
> 本 Session 寫 CLI 入口（cobra）。中空模板——參數結構固定，help text 留空。
> ★Go 實作：用 cobra（github.com/spf13/cobra）做 CLI，go.mod 管理 module。

---

## === S4 PROMPT（複製以下 code block 內容）===

```
你是 S4 CLI Interface SubAgent。你的任務是寫 OpenConveneCLI 的命令列介面（Go + cobra）。

═══════════════════════════════════════════════════════════════
【前置 — 必讀】
═══════════════════════════════════════════════════════════════

1. read("Docs/01-Architecture.md") → 取得 CLI 介面定義
2. read(".agent/handoff/S3.md") → 取得 ConveneEngine.Run() 簽名 + prompts.go 模板
3. read(".agent/handoff/S2.md") → 取得 DetectAvailableAdapters 簽名 + DetectResult struct
4. read("internal/convene/engine.go") → 確認 ConveneEngine 介面
5. read("internal/mode/mode.go") → 確認 Mode type
6. read("internal/convene/prompts.go") → 確認 prompt 模板結構（--verbose 輸出需引用）
7. read("internal/adapter/detect.go") → 確認 DetectAvailableAdapters 函式

═══════════════════════════════════════════════════════════════
【要產出的檔案】
═══════════════════════════════════════════════════════════════

cmd/
└── openconvene-cli/
    └── main.go              # CLI entry point（cobra root command + subcommands）

★依賴安裝：
  exec("go get github.com/spf13/cobra@latest")
  exec("go mod tidy")
  ★go.mod 已由 S1 產出，yaml.v3 已由 S5 加入 go.mod——不需自己 go get yaml.v3

═══════════════════════════════════════════════════════════════
【CLI 介面設計】
═══════════════════════════════════════════════════════════════

用 cobra（Go 生態最成熟 CLI 框架）。

命令結構：
  openconvene-cli run --task "..." --mode {research|code|agent} \
    --responders agy,grok,devin \
    --executor codex \
    [--synthesizer agy] \
    [--config ~/.config/openconvene-cli/models.yaml] \
    [--timeout 120] \
    [--verbose]

  openconvene-cli list-models [--config ...]
    → 列出 config 中所有可用模型 + read_only 能力

  openconvene-cli detect
    → ★自動偵測系統上已安裝哪些 CLI（9 個：devin/grok/codex/agy/cursor/kimi/hermes/aider/opencode）
    → 顯示各 CLI 的安裝狀態 + read_only 能力 + 適合角色（responder/executor）
    → ★未安裝的 CLI 顯示安裝指令供使用者參考（不自動安裝）
    → 不需 config——直接掃描 PATH

  openconvene-cli config init [--path ...]
    → 生成範例 models.yaml

  openconvene-cli config validate [--config ...]
    → 驗證 config 是否合法

═══════════════════════════════════════════════════════════════
【main.go 骨架】
═══════════════════════════════════════════════════════════════

```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "strings"
    "github.com/spf13/cobra"
    "github.com/masteryee-labs/open-convene-cli/internal/adapter"
    "github.com/masteryee-labs/open-convene-cli/internal/config"
    "github.com/masteryee-labs/open-convene-cli/internal/convene"
    "github.com/masteryee-labs/open-convene-cli/internal/mode"
)

func main() {
    rootCmd := buildRootCmd()
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func buildRootCmd() *cobra.Command {
    rootCmd := &cobra.Command{
        Use:   "openconvene-cli",
        Short: <CLI_SHORT_DESCRIPTION>,  // ← 填入
        Long:  <CLI_LONG_DESCRIPTION>,   // ← 填入
    }

    rootCmd.AddCommand(buildRunCmd())
    rootCmd.AddCommand(buildListModelsCmd())
    rootCmd.AddCommand(buildDetectCmd())
    rootCmd.AddCommand(buildConfigCmd())
    return rootCmd
}

func buildRunCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "run",
        Short: <RUN_SHORT>,  // ← 填入
        RunE:  runRun,
    }
    cmd.Flags().String("task", "", <TASK_HELP>)  // ← 填入
    cmd.MarkFlagRequired("task")
    cmd.Flags().String("mode", "", "模式: research|code|agent")
    cmd.MarkFlagRequired("mode")
    cmd.Flags().String("responders", "", "逗號分隔的 responder 模型名")
    cmd.Flags().String("executor", "", "executor 模型名")
    cmd.Flags().String("synthesizer", "", "整合者模型名（可選）")
    cmd.Flags().String("config", "", "models.yaml 路徑")
    cmd.Flags().Int("timeout", 0, "覆寫預設 timeout（秒）")
    cmd.Flags().Bool("verbose", false, "顯示詳細過程")
    return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
    // <IMPLEMENTATION> ← 填入
    // 0. ★解析 configPath（--config > env > 預設）：
    //    configPath := cmd.Flags().GetString("config")
    //    if configPath == "" {
    //        configPath = os.Getenv("OPENCONVENE_CLI_CONFIG")
    //    }
    //    → configPath 傳給 config.LoadConfig，空字串時 LoadConfig 內部搜尋預設路徑
    // 1. ★config.LoadConfig(configPath) → *config.ConveneConfig（大寫，跨 package）
    // 2. ★合併 CLI 參數與 config.Defaults（CLI 優先）：
    //    a. responders：
    //       flagStr := cmd.Flags().GetString("responders")
    //       if flagStr != "" {
    //           responders = strings.Split(flagStr, ",")
    //           // ★trim 每個 element 的空白
    //       } else {
    //           responders = cfg.Defaults.Responders
    //       }
    //       → 若 responders 仍為空（len==0）→ 報錯「no responders specified」
    //       ★注意：空 string 的 strings.Split 會回傳 [""]（len=1），需先檢查 flagStr != ""
    //    b. executor = flagExecutor（非空）or cfg.Defaults.Executor
    //    c. synthesizer（★string → *string 轉換）：
    //       flagSynth := cmd.Flags().GetString("synthesizer")
    //       var synthesizer *string
    //       if flagSynth != "" {
    //           synthesizer = &flagSynth  // 非空 → 取指標
    //       } else {
    //           synthesizer = cfg.Defaults.Synthesizer  // 已是 *string（可能 nil）
    //       }
    //    d. ★timeout 覆寫（engine.Run() 簽名沒 timeout，透過 config 傳）：
    //       flagTimeout := cmd.Flags().GetInt("timeout")
    //       if flagTimeout > 0 {
    //           cfg.Defaults.Timeout = flagTimeout  // ★覆寫 defaults，engine 會讀 e.Config.Defaults.Timeout
    //       }
    // 3. ★string → mode.Mode 轉換：
    //    modeStr := cmd.Flags().GetString("mode")
    //    m := mode.Mode(modeStr)  // ★typed string 轉換
    //    驗證 mode + model 組合（mode.ValidateModeConfig(m, responders, executor, synthesizer, cfg.Models)）
    // 4. engine := convene.NewConveneEngine(cfg)
    //    result, err := engine.Run(ctx, task, modeStr, responders, executor, synthesizer)
    //    ★engine.Run() 接受 string（不是 mode.Mode），因為 convene package 不 import mode（避免循環依賴）
    // 5. mode.FormatOutput(result, m) → 印出（★FormatOutput 要 mode.Mode）
    // 6. 若 --verbose → 印出各 responder 的原始回應 + metadata
    //    ★metadata 是 map[string]interface{}，用 fmt.Sprintf("%v", v) 或 json.Marshal 格式化
    //    ★建議格式：
    //    for name, resp := range result.Responses {
    //        fmt.Fprintf(os.Stderr, "--- %s ---\n%s\n", name, resp)
    //    }
    //    for k, v := range result.Metadata {
    //        fmt.Fprintf(os.Stderr, "[metadata] %s: %v\n", k, v)
    //    }
    ...
}

func buildListModelsCmd() *cobra.Command {
    // <IMPLEMENT_LIST_MODELS> ← 填入
    // openconvene-cli list-models [--config ...]
    // 列出 config 中所有模型 + read_only 能力
    ...
}

func buildDetectCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "detect",
        Short: <DETECT_SHORT>,  // ← 填入，如 "偵測系統已安裝的 CLI adapter"
        RunE:  runDetect,
    }
    return cmd
}

func runDetect(cmd *cobra.Command, args []string) error {
    // <IMPLEMENTATION> ← 填入
    // 1. results := adapter.DetectAvailableAdapters()
    // 2. 印出表格（9 個 CLI）：
    //    | CLI       | Installed | Path           | Read-Only | Can Respond | Can Execute |
    //    |-----------|-----------|----------------|-----------|-------------|-------------|
    //    | agy       | ✓         | /usr/bin/agy   | maybe     | ✓           | ✓           |
    //    | aider     | ✗         |                | false     | ✗           | ✓           |
    //    | codex     | ✓         | /usr/bin/codex | true      | ✓           | ✓           |
    //    | cursor    | ✓         | /usr/bin/cursor| true      | ✓           | ✓           |
    //    | devin     | ✗         |                | maybe     | ✓           | ✓           |
    //    | grok      | ✓         | /usr/bin/grok  | maybe     | ✓           | ✓           |
    //    | hermes    | ✗         |                | maybe     | ✓           | ✓           |
    //    | kimi      | ✓         | /usr/bin/kimi  | true      | ✓           | ✓           |
    //    | opencode  | ✗         |                | maybe     | ✓           | ✓           |
    // 3. 印出建議：
    //    - "Available responders: agy, codex, cursor, grok, kimi"
    //    - "Available executors: agy, codex, cursor, grok, kimi"
    //    - "Missing (install to use):"
    //      "  aider:    python -m pip install aider-install && aider-install"
    //      "  devin:    curl -fsSL https://cli.devin.ai/install.sh | bash"
    //      "  hermes:   hermes setup --portal"
    //      "  opencode: 見 https://opencode.ai/docs/cli/"
    // ★未安裝的 CLI 顯示 InstallCmd 供使用者參考，但不自動執行安裝
    ...
}

func buildConfigCmd() *cobra.Command {
    // <IMPLEMENT_CONFIG_SUBCOMMANDS> ← 填入
    // ★config 是 parent command，有兩個子命令：
    //
    // config init [--path ...]
    //   → 呼叫 config.InitConfig(path)
    //   → path 預設 = "models.yaml"（當前目錄）或 ~/.config/openconvene-cli/models.yaml
    //   → 成功 → 印出 "Config written to <path>"
    //
    // config validate [--config ...]
    //   → 呼叫 config.LoadConfig(configPath) → *ConveneConfig
    //   → 呼叫 config.ValidateConfig(cfg) → []string
    //   → 若清單非空 → 印出每個錯誤/警告 + return error
    //   → 若清單為空 → 印出 "Config is valid"
    //
    // ★cobra 子命令結構：
    //   configCmd := &cobra.Command{Use: "config", Short: "Config management"}
    //   configCmd.AddCommand(initCmd, validateCmd)
    //   return configCmd
    ...
}
```

→ 以上為骨架，具體實作由你填。

═══════════════════════════════════════════════════════════════
【go.mod 管理】
═══════════════════════════════════════════════════════════════

★go.mod 已由 S1 產出——不要自己建立 go.mod。
★安裝依賴：
  exec("go get github.com/spf13/cobra@latest")
  exec("go mod tidy")

═══════════════════════════════════════════════════════════════
【實作規則】
═══════════════════════════════════════════════════════════════

1. --responders 用逗號分隔解析成 []string（strings.Split）
2. --task 可從 stdin 讀（若值為 "-"）——用 io.ReadAll(os.Stdin)
3. research 模式 --executor 可省（只印 synthesis/responses）
4. code/agent 模式 --executor 必填，否則報錯
5. --verbose 時印出各 responder 原始回應 + metadata
6. 錯誤訊息印到 stderr（fmt.Fprintf(os.Stderr, ...)），exit code 非 0（os.Exit(1)）
7. config 路徑解析：--config > 環境變數 OPENCONVENE_CLI_CONFIG > 預設 ~/.config/openconvene-cli/models.yaml
8. ★Go 的 context.Context：用 context.Background() 傳入 engine.Run()
9. ★cobra 的 RunE 回傳 error，cobra 自動處理 exit code

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- cmd/openconvene-cli/main.go 含 buildRootCmd + buildRunCmd + runRun + buildDetectCmd + runDetect + buildConfigCmd + main
- go.mod 含 cobra 依賴
- exec("go run ./cmd/openconvene-cli --help") 不報錯
- exec("go run ./cmd/openconvene-cli run --help") 不報錯
- exec("go run ./cmd/openconvene-cli detect --help") 不報錯
- exec("go run ./cmd/openconvene-cli config --help") 不報錯
- exec("go run ./cmd/openconvene-cli config init --help") 不報錯
- exec("go run ./cmd/openconvene-cli config validate --help") 不報錯
- 所有 <PLACEHOLDER> 已填入
- git commit: feat(S4): implement Go CLI interface with cobra run/list-models/detect/config subcommands

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S4.md，內容：
- CLI 完整參數清單（cobra flags）
- 各子命令的行為
- go.mod 的最終依賴清單
- git commit hash
```
