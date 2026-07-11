<div align="center">

# OpenConveneCLI

### 多模型 AI 协作 CLI 工具 — 通过原生 CLI 编排 N 个 AI 代码代理

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | **简体中文** | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## 概述

**OpenConveneCLI** 是一款开源的 Go 命令行工具，实现了**多模型协作** — 将同一个 prompt 同时分发给 N 个响应者模型（每个通过其原生 CLI 以只读模式运行），将其响应综合为统一结论，再委托给执行者模型根据综合结果采取行动（编写代码、修改文件或运行长程代理任务）。通过 AI CLI 编排与 Mixture-of-Agents (MoA) 架构，实现高效的 AI 代码生成与多代理协同。

该方法与 [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) 和 [OpenRouter Fusion](https://openrouter.ai/) 理念一致，但引入了一项关键创新：**CLI-as-Model** — 无需统一 API，而是编排每个模型的原生 CLI（Devin、Grok、Codex、Antigravity、Cursor、Kimi、Hermes、Aider、OpenCode）。即使某个模型没有公开 API，只要它有 CLI，就能参与协作。

> **关键词**：AI CLI 编排、多模型协作、Mixture-of-Agents、MoA、AI 代码生成、多代理系统、CLI-as-Model、AI 代码代理、LLM 编排、fan-out AI

---

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [工作原理](#工作原理)
- [支持的 AI CLI](#支持的-ai-cli)
- [命令](#命令)
- [交互式 REPL](#交互式-repl)
- [CLI 标志](#cli-标志)
- [为什么选择 Go](#为什么选择-go)
- [文档](#文档)
- [许可证](#许可证)

---

## 安装

### 一键安装（推荐）

**Linux / macOS：**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows（PowerShell）：**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### 使用 Go 安装

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### 从源码编译

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> 前置条件：Go 1.24+

---

## 快速开始

```bash
# 1. Detect installed AI CLIs
openconvene detect

# 2. Generate config
openconvene init --path ~/.config/openconvene/models.yaml

# 3. Run multi-model collaboration
openconvene ask "your question" --responders agy,grok

# 4. Write code (default code mode)
openconvene "fix the bug in foo.go"

# 5. Agent task
openconvene agent "deploy the app"
```

> 响应者可使用任何已安装的 CLI：agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode（至少需要 1 个）。

### 更新

在 REPL 中输入 `/update` 查看适合你平台的更新命令。或者再次运行安装命令——它会用最新版本覆盖旧的二进制文件。

---

## 工作原理

OpenConveneCLI 提供三种模式，匹配真实的开发者工作流：

| 模式 | 命令 | 流水线 | 是否执行 | 典型用例 |
|------|------|--------|----------|----------|
| `ask` | `openconvene ask "..."` | N 个响应者 → 综合器 → 打印结论 | 否 | 技术调研、方案对比 |
| `code`（默认） | `openconvene "..."` | N 个响应者 → 综合器（可选）→ 执行者编写代码 | 是 — 编写代码 | 实现功能、修复 bug |
| `agent` | `openconvene agent "..."` | N 个响应者 → 综合器 → 执行者代理 | 是 — 代理模式 | 复杂的多步骤任务 |

```
                    ┌──────────┐
                    │  Prompt  │
                    └────┬─────┘
                         │ fan-out
            ┌────────────┼────────────┐
            ▼            ▼            ▼
       ┌────────┐  ┌────────┐  ┌────────┐
       │Responder│  │Responder│  │Responder│
       │  (agy) │  │ (grok) │  │ (codex)│
       └───┬────┘  └───┬────┘  └───┬────┘
           │           │           │
           └───────────┼───────────┘
                       ▼
                ┌─────────────┐
                │ Synthesizer │
                └──────┬──────┘
                       ▼
                ┌──────────┐
                │ Executor │
                └──────────┘
```

---

## 支持的 AI CLI

OpenConveneCLI 开箱即支持 9 款 AI 代码代理 CLI。每个 CLI 连接各自的模型后端——OpenConveneCLI 本身不依赖任何云服务。使用该工具至少需要安装 1 个 CLI。

| CLI | 说明 | 只读 | 执行者 | 安装命令 |
|-----|------|------|--------|----------|
| [Devin](https://devin.ai) | Cognition 的自主 AI 软件工程师。全栈 coding agent，具备 shell 访问、浏览器控制和长时程任务规划能力。 | 可能 | 是 | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | xAI 的 AI coding CLI，由 Grok 模型驱动。快速推理与代码生成，具备实时知识访问能力。 | 可能 | 是 | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | OpenAI 的终端 coding agent。沙箱执行——`--sandbox read-only` 用于安全研究，`workspace-write` 用于代码执行。 | 是 | 是 | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | Google 的 AI coding agent CLI，由 Gemini 驱动。支持多文件编辑、代码审查和 agentic 任务执行（Gemini 2.5 Pro/Flash）。 | 可能 | 是 | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | AI 优先的代码编辑器，具备 agent 模式。无 `--force` 时为只读分析；加 `--force` 时自主编辑文件。由 Claude、GPT-4、Gemini 驱动。 | 是 | 是 | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Moonshot AI 的 coding CLI，由 Kimi K2 驱动。长上下文代码理解（256K tokens），只读操作自动批准。 | 是 | 是 | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | HashiCorp 的 AI agent CLI。`chat -q` 单次查询模式；agentic 模式用于多步骤基础设施与代码任务。 | 可能 | 是 | `hermes setup --portal` |
| [Aider](https://aider.chat) | 开源 AI 结对编程工具。与 Git 集成，支持 GPT-4o、Claude 3.5、DeepSeek 及本地 LLM。编辑优先设计——默认会修改文件。 | 否 | 是 | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | 开源 AI coding agent。`run` 子命令用于非交互单一 prompt；agentic 模式用于自主开发。支持多个 LLM 提供商。 | 可能 | 是 | 参见 [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> **只读** 表示该 CLI 能否安全地以响应者模式运行（不修改文件）。`是` = 强制只读，`可能` = 非交互模式但可能触发工具，`否` = 默认修改文件（仅限执行者）。

---

## 命令

```bash
# Single-shot (with task argument)
openconvene "task"              # default code mode (writes code)
openconvene ask "task"          # ask mode (research, no execution)
openconvene agent "task"        # agent mode (agentic actions)

# Interactive mode (no task argument → enters REPL)
openconvene                     # interactive REPL (default code mode)
openconvene ask                 # interactive REPL (ask mode)
openconvene agent               # interactive REPL (agent mode)

# Utility commands
openconvene models              # list configured models
openconvene detect              # detect installed AI CLIs
openconvene init                # generate starter models.yaml
openconvene check               # validate models.yaml
```

---

## 交互式 REPL

运行 `openconvene`、`openconvene ask` 或 `openconvene agent` 时不带任务参数，将进入交互式 REPL，类似于 codex、grok、agy 和 devin。

在 REPL 中，你可以直接输入 prompt，或使用斜杠命令切换设置：

```
openconvene(code)> fix the bug in main.go     # direct prompt
openconvene(code)> /mode ask                  # switch to ask mode
openconvene(ask)> /executor devin             # switch executor model
openconvene(ask)> /responders agy,grok,codex  # switch responders
openconvene(ask)> /synthesizer grok           # switch synthesizer
openconvene(ask)> /language zh-TW             # set model response language
openconvene(ask)> /status                     # view session status
openconvene(ask)> /usage                      # view per-CLI usage stats
openconvene(ask)> /models                     # list configured models
openconvene(ask)> /detect                     # detect installed CLIs
openconvene(ask)> /config                     # show current config
openconvene(ask)> /new                        # clear session
openconvene(ask)> /help                       # show all commands
openconvene(ask)> /exit                       # exit REPL
```

> **REPL 功能**：fish 风格的菜单补全（Tab 显示补全菜单，上/下箭头浏览候选项，Enter 确认，Shift-Tab 反向循环）、增量历史搜索（Ctrl-R/Ctrl-S）、跨会话命令历史。由 [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4 驱动。

### 斜杠命令

| 命令 | 别名 | 说明 |
|------|------|------|
| `/help` | `/h`, `/?` | 显示所有可用命令 |
| `/status` | | 显示会话状态（模式、模型、运行次数） |
| `/mode [ask\|code\|agent]` | | 显示或切换当前模式 |
| `/models` | `/m` | 列出所有已配置的模型 |
| `/responders [a,b,c]` | | 显示或设置响应者 |
| `/executor [name]` | | 显示或设置执行者 |
| `/synthesizer [name]` | | 显示或设置综合器（`none` 清除） |
| `/language [lang]` | `/lang` | 显示或设置模型响应语言 |
| `/usage` | `/u` | 显示各 CLI 的使用统计 |
| `/config` | `/c`, `/settings` | 显示当前配置摘要 |
| `/detect` | `/d` | 检测已安装的 CLI |
| `/clear` | `/new` | 清屏并重置会话 |
| `/compact` | | （存根）压缩对话以释放 token |
| `/resume` | `/continue` | （存根）恢复之前的会话 |
| `/update` | | （存根）检查并安装更新 |
| `/exit` | `/quit`, `/q` | 退出 REPL |

---

## CLI 标志

| 标志 | 说明 |
|------|------|
| `-p`, `--print` | 非交互式单次运行模式 |
| `-m`, `--model <name>` | 指定模型（`--executor` 的别名） |
| `--json` | JSON 输出格式 |
| `--responders <a,b,c>` | 指定响应者 |
| `--executor <name>` | 指定执行者 |
| `--synthesizer <name>` | 指定综合器 |
| `--config <path>` | 指定配置文件路径 |
| `--timeout <sec>` | 覆盖超时时间 |
| `--verbose` | 显示原始响应和元数据 |
| `--language <lang>` | 设置模型响应语言 |
| `--` | 分隔符（在 prompt 之前添加） |

---

## 为什么选择 Go

- **单一静态二进制** — 编译产物零运行时依赖；`curl + chmod` 即可运行
- **Goroutine 原生并发** — N 个响应者并行 fan-out，比 Python asyncio 更轻量
- **启动快速** — 约 5ms 启动，适合 CLI 使用场景
- **静态类型** — 强类型结构体替代 map，重构更安全
- **跨平台** — `GOOS=windows/linux/darwin` 一条命令交叉编译

---

## 文档

| 文档 | 内容 |
|------|------|
| [Overview](Docs/00-Overview.md) | 设计动机、与 Fusion/MoA 的对比 |
| [Architecture](Docs/01-Architecture.md) | 系统架构、Go 模块结构、数据流 |
| [Usage Guide](Docs/02-Usage-Guide.md) | 完整使用指南（安装、配置、标志、模式） |
| [Model Adapters](Docs/03-Model-Adapters.md) | 9 个 CLI 适配器设计、只读能力矩阵 |
| [Configuration](Docs/04-Configuration.md) | 完整的 `models.yaml` schema 及示例 |
| [Examples](Docs/05-Examples.md) | 各模式的真实使用示例 |
| [Troubleshooting](Docs/06-Troubleshooting.md) | 常见问题与解决方案 |

---

## 许可证

[MIT](LICENSE)
