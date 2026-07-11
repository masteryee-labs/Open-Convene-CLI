# Docs/Agents/ — AI Agent 提示詞目錄

## 本目錄的用途

存放**人類撰寫/審核後交給 AI Agent（Devin / Claude / Antigravity / Gemini CLI 等）執行的提示詞計畫檔**。

每個子資料夾是獨立「計畫主題」，`.md` 檔是給 AI 用的 Session Prompt / Session Plan。

## ⚠️ 重要：本目錄不是 AI Slop

| 項目 | 說明 |
|------|------|
| **什麼是 AI Slop** | AI 在執行 Session 過程中**臨時產出**的拋棄式檔案（.tmp_*.go / debug_*.log / 臨時測試腳本等），commit 前應被清理 |
| **本目錄不是 AI Slop** | 本目錄的檔案是**人類刻意撰寫的提示詞規格**，是給 AI 讀的「任務說明書」，不是 AI 自己產生的垃圾 |
| **清理規則不適用** | 任何 AI Agent 執行清理時，**禁止刪除、移動、修改 `Docs/Agents/` 底下任何檔案**，除非該 Session 的 LOCK 範圍明確包含此目錄 |

## 給 AI Agent 的指示

```
1. Docs/Agents/ 底下所有檔案都是「人類撰寫的提示詞規格」，受保護。
2. 執行清理時，Docs/Agents/ 不在清理範圍內。
3. 除非你的 Session GoalSpec / LOCK 明確指明要修改 Docs/Agents/ 底下某檔，否則禁碰。
```

## 目前的子資料夾

| 子資料夾 | 用途 | 狀態 |
|---------|------|------|
| `指揮官/` | OpenConveneCLI 建造管線指揮官 prompt（Orchestrator_Prompt.md）——人類複製貼上用入口 | v1.0，Go 語言版 |
| `初版設計/` | OpenConveneCLI 初版設計 Session 拆分計畫（Initial_Design_Session_Plan.md + Initial_Design_Session_Prompts/）。8 個子 Session（S0-S7），Go 語言 | v1.0，待執行 |

> 每個子資料夾放獨立計畫主題的提示詞檔。`指揮官/` 是人類入口，`初版設計/` 是 Session 規格。
