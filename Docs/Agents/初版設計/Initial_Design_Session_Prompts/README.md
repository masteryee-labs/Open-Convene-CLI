# OpenConveneCLI 初版設計子 Session Prompt 清單

## Session 清單（8 個）

| Session | 檔案 | 功能 | 類型 | 依賴 |
|---------|------|------|------|------|
| S0 | S0-Deploy-Harness.md | 部署 Agent Harness + 安裝 Go | D | — |
| S1 | S1-Architecture-Docs.md | 架構文件 + 專案骨架（go.mod + models.go） | A | S0 |
| S2 | S2-Model-Adapters.md | Model Adapters（os/exec + interface） | C | S1 |
| S3 | S3-Convene-Core.md | Convene Core（goroutines fan-out） | C | S2 |
| S4 | S4-CLI-Interface.md | CLI Interface（cobra） | C | S3,S5 |
| S5 | S5-Config-System.md | Config System（yaml.v3） | C | S1 |
| S6 | S6-Tests.md | Tests（go test + testify） | T | S2,S3,S4,S5 |
| S7 | S7-User-Docs.md | User Docs（go install 範例） | DOC | S4,S5 |

## 派發順序

```
Wave 1（exclusive）：S0
Wave 2：S1
Wave 3（並行）：S2 + S5
Wave 4：S3
Wave 5：S4
Wave 6（並行）：S6 + S7
```

## 子 Prompt 格式

每個子 prompt 包含：
1. **類型/依賴標注**：檔頭 metadata
2. **前置必讀**：需讀取的 Docs / handoff / 源碼檔
3. **要產出的檔案**：清單 + 結構
4. **實作指引**：骨架程式碼 + 中空部分由 SubAgent 填
5. **實作規則**：Go 慣例 + 紅線
6. **驗收條件**：完成的判定標準
7. **handoff 格式**：產出檔案清單 + git commit hash + 已知問題
