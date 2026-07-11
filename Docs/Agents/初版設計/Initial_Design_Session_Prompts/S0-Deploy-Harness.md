# S0 — 部署 Agent Harness

> 類型：D (Deploy) | 依賴：無 | 並行限制：exclusive
> 本 Session 部署 Tool.Agent-Harness-Deploy 到本專案，讓後續 Session 擁有 caveman 壓縮、commander-worker、loop memory 等 harness 規則。
> ★本專案使用 Go 語言開發——harness 本身語言無關，部署流程不變。

---

## === S0 PROMPT（複製以下 code block 內容）===

```
你是 S0 Deploy SubAgent。你的任務是部署 Agent Harness 到本專案。

═══════════════════════════════════════════════════════════════
【目標】
═══════════════════════════════════════════════════════════════

部署 https://github.com/masteryee-labs/Tool.Agent-Harness-Deploy 到本專案，
讓本專案擁有與 Yee-World-Life 相同的 agent harness：
- caveman token 壓縮（~65% token cut）
- commander-worker 階層
- loop engineering + 三層記憶
- skills 系統
- 跨工具適配（Devin / Codex / Antigravity / Cursor 等）

★本專案是 Go 語言 CLI 專案（OpenConveneCLI），但 harness 部署語言無關。

═══════════════════════════════════════════════════════════════
【步驟 0：安裝 Go 編譯器（★必做，後續所有 Session 依賴）】
═══════════════════════════════════════════════════════════════

★系統可能尚未安裝 Go。你必須先安裝，否則 S2-S6 無法編譯/測試 Go 程式碼。

1. 檢查 Go 是否已安裝：
   exec("go version")
   → 若成功（顯示版本號）→ 跳過安裝，直接進入步驟 1
   → 若失敗 → 繼續安裝

2. 安裝 Go（★跨平台——先偵測 OS 再選對應方式）：

   ★偵測作業系統：
   exec("python -c \"import platform; print(platform.system())\"")
   → "Windows" → 用 Windows 流程
   → "Linux"   → 用 Linux 流程
   → "Darwin"  → 用 macOS 流程

   ─── Windows 流程 ───
   a. 下載 Go MSI installer：
      exec("curl -L -o %TEMP%\\go-installer.msi https://go.dev/dl/go1.22.5.windows-amd64.msi")
      → 若 curl 不可用，用 PowerShell：
      exec("powershell -Command \"Invoke-WebRequest -Uri 'https://go.dev/dl/go1.22.5.windows-amd64.msi' -OutFile $env:TEMP\\go-installer.msi\"")

   b. 靜默安裝：
      exec("msiexec /i %TEMP%\\go-installer.msi /quiet /norestart")
      → 安裝到 C:\\Program Files\\Go\\

   c. ★安裝後 PATH 不會立即生效——在當前 session 手動設定：
      exec("set PATH=%PATH%;C:\\Program Files\\Go\\bin")
      exec("set GOROOT=C:\\Program Files\\Go")

   d. 驗證：
      exec("C:\\Program Files\\Go\\bin\\go.exe version")
      → 應顯示 "go version go1.22.5 windows/amd64"
      → 若失敗 → 在 handoff 標記 BLOCKED_REASON="Go installation failed: <錯誤訊息>"

   e. ★也設定全域 PATH（讓後續 SubAgent 能用）：
      exec("setx PATH \"%PATH%;C:\\Program Files\\Go\\bin\"")
      exec("setx GOROOT \"C:\\Program Files\\Go\"")
      → setx 只影響新 process，不影響當前 session

   f. 清理安裝包：
      exec("del %TEMP%\\go-installer.msi")

   ★若 msiexec 靜默安裝失敗（權限問題），改用 zip 方式：
   a. exec("curl -L -o %TEMP%\\go.zip https://go.dev/dl/go1.22.5.windows-amd64.zip")
   b. exec("powershell -Command \"Expand-Archive -Path $env:TEMP\\go.zip -DestinationPath C:\\Go -Force\"")
   c. exec("set PATH=%PATH%;C:\\Go\\bin")
   d. exec("setx PATH \"%PATH%;C:\\Go\\bin\"")
   e. exec("C:\\Go\\bin\\go.exe version")  # 驗證

   ─── Linux 流程 ───
   a. 偵測架構：
      exec("uname -m")  → x86_64 = amd64 / aarch64 = arm64
   b. 下載（amd64 範例）：
      exec("curl -L -o /tmp/go.tar.gz https://go.dev/dl/go1.22.5.linux-amd64.tar.gz")
   c. 解壓到 /usr/local（需 sudo）：
      exec("sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz")
   d. 設定 PATH（當前 session + 永久）：
      exec("export PATH=$PATH:/usr/local/go/bin")
      exec("echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc")
      → 若用 zsh：>> ~/.zshrc
   e. 驗證：
      exec("/usr/local/go/bin/go version")
      → 應顯示 "go version go1.22.5 linux/amd64"
   f. 清理：
      exec("rm /tmp/go.tar.gz")

   ─── macOS 流程 ───
   ★macOS 最簡方式是用 Homebrew（若已安裝）：
   a. exec("brew install go")
   b. exec("go version")  # 驗證
   → Homebrew 會自動處理 PATH

   ★若 Homebrew 不可用，用官方 tar.gz：
   a. 偵測架構：
      exec("uname -m")  → arm64 = Apple Silicon / x86_64 = Intel
   b. 下載（Apple Silicon 範例）：
      exec("curl -L -o /tmp/go.tar.gz https://go.dev/dl/go1.22.5.darwin-arm64.tar.gz")
   c. 解壓到 /usr/local：
      exec("sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz")
   d. 設定 PATH：
      exec("export PATH=$PATH:/usr/local/go/bin")
      exec("echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc")
   e. 驗證：
      exec("/usr/local/go/bin/go version")
   f. 清理：
      exec("rm /tmp/go.tar.gz")

═══════════════════════════════════════════════════════════════
【步驟 1+：部署 Harness】
═══════════════════════════════════════════════════════════════

1. clone harness repo 到臨時目錄（跨平台）：
   exec("git clone https://github.com/masteryee-labs/Tool.Agent-Harness-Deploy <TEMP_DIR>")
   → <TEMP_DIR> 用系統臨時目錄：
     - Linux/macOS: /tmp/harness-deploy
     - Windows: %TEMP%\harness-deploy（如 C:\Users\<user>\AppData\Local\Temp\harness-deploy）
   → 偵測方式：exec("python -c \"import tempfile; print(tempfile.gettempdir())\"")
   → ★若 git clone 失敗（網路/私有庫/URL 變更）：
     1. 重試一次（間隔 5 秒）
     2. 仍失敗 → 在 handoff 標記 BLOCKED_REASON="git clone failed: <錯誤訊息>"
     3. 不自行猜測替代 URL → 留給指揮官 ask_user_question

2. 進入 clone 目錄，讀取 AGENTS.md 了解部署流程

3. 執行部署腳本：
   exec("python <TEMP_DIR>/scripts/distill.py")
   → 這會：偵測已安裝的 AI 工具 → 生成 canonical harness → 寫入各工具 native config
   → 若 distill.py 失敗，嘗試 manual deploy:
     exec("python <TEMP_DIR>/scripts/detect.py")  # 先看偵測結果
     exec("python <TEMP_DIR>/scripts/sync.py")    # 再手動 sync

4. 確認部署結果：
   - read(".agent/loop_state.md") ← 應存在（registry）
   - read(".agent/knowledge_distill.md") ← 應存在（anti-patterns）
   - read("AGENTS.md") ← 應存在（入口路由）
   - 確認 distill.py 輸出報告了哪些工具被偵測 + 部署

5. 若 harness 部署到本專案而非全域：
   - 確認 .devin/ 或 .codex/ 或 .claude/ 等目錄已建立（依偵測到的工具）
   - 確認 skills 目錄已部署

6. 清理臨時目錄（跨平台）：
   exec("python -c \"import shutil; shutil.rmtree('<TEMP_DIR>', ignore_errors=True)\"")

7. git add + commit：
   exec("git add -A && git commit -m 'feat(S0): deploy agent harness from Tool.Agent-Harness-Deploy'")

═══════════════════════════════════════════════════════════════
【驗收條件】
═══════════════════════════════════════════════════════════════

- ★Go 已安裝：exec("go version") 或 exec("C:\\Program Files\\Go\\bin\\go.exe version") 成功
- .agent/loop_state.md 存在且 <3KB
- .agent/knowledge_distill.md 存在且 <8KB
- AGENTS.md 存在
- 至少一個工具的 config 目錄被部署（.devin/ 或 .codex/ 或 .claude/ 等）
- git commit 已建立

═══════════════════════════════════════════════════════════════
【handoff】
═══════════════════════════════════════════════════════════════

完成後寫 .agent/handoff/S0.md，內容：
- ★Go 安裝狀態（版本、安裝路徑、PATH 是否已設定）
- 部署了哪些工具的 config
- .agent/ 目錄結構
- git commit hash
- 已知問題（若有）
```
