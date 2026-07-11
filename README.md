<div align="center">

# OpenConveneCLI

### Multi-Model AI Collaboration CLI Tool — Orchestrate N AI Coding Agents via Native CLIs

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

**English** | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## Overview

**OpenConveneCLI** is an open-source Go command-line tool that implements **multi-model AI collaboration** — dispatching the same prompt simultaneously to N responder models (each via its native CLI in read-only mode), synthesizing their responses into a unified conclusion, then delegating to an executor model that acts on the synthesized result (writing code, modifying files, or running long-horizon agent tasks).

This approach aligns with [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) and [OpenRouter Fusion](https://openrouter.ai/), but introduces a key innovation: **CLI-as-Model** — instead of requiring a unified API, it orchestrates each model's native CLI (Devin, Grok, Codex, Antigravity, Cursor, Kimi, Hermes, Aider, OpenCode). Even if a model lacks a public API, as long as it has a CLI, it can participate in the collaboration.

> **Keywords**: AI CLI orchestration, multi-model collaboration, Mixture-of-Agents, MoA, AI code generation, multi-agent system, CLI-as-Model, AI coding agent, LLM orchestration, fan-out AI

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Supported AI CLIs](#supported-ai-clis)
- [Commands](#commands)
- [Interactive REPL](#interactive-repl)
- [CLI Flags](#cli-flags)
- [Why Go](#why-go)
- [Documentation](#documentation)
- [License](#license)

---

## Installation

### One-line install (recommended)

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### Install with Go

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### Build from source

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> Prerequisite: Go 1.24+

---

## Quick Start

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

> Responders can use any installed CLI: agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (at least 1 required).

### Update

In the REPL, type `/update` to see the update command for your platform. Or run the install command again — it will overwrite the old binary with the latest version.

---

## How It Works

OpenConveneCLI provides three modes matching real developer workflows:

| Mode | Command | Pipeline | Executes? | Typical Use Case |
|------|---------|----------|-----------|-----------------|
| `ask` | `openconvene ask "..."` | N responders → synthesizer → print conclusion | No | Technical research, solution comparison |
| `code` (default) | `openconvene "..."` | N responders → synthesizer (optional) → executor writes code | Yes — writes code | Implement features, fix bugs |
| `agent` | `openconvene agent "..."` | N responders → synthesizer → executor agent | Yes — agent mode | Complex multi-step tasks |

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

## Supported AI CLIs

OpenConveneCLI supports 9 AI coding-agent CLIs out of the box. Each CLI connects to its own model backend — OpenConveneCLI itself does not depend on any cloud service. At least 1 CLI must be installed to use the tool.

| CLI | Description | Read-Only | Executor | Install |
|-----|-------------|-----------|----------|---------|
| [Devin](https://devin.ai) | Autonomous AI software engineer by Cognition. Full-stack coding agent with shell access, browser control, and long-horizon task planning. | Maybe | Yes | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | xAI's AI coding CLI powered by Grok models. Fast reasoning and code generation with real-time knowledge access. | Maybe | Yes | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | OpenAI's terminal-based coding agent. Sandboxed execution with `--sandbox read-only` for safe research and `workspace-write` for code execution. | Yes | Yes | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | Google's AI coding agent CLI powered by Gemini. Multi-file editing, code review, and agentic task execution with Gemini 2.5 Pro/Flash. | Maybe | Yes | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | AI-first code editor with agent mode. Read-only analysis without `--force`; autonomous file editing with `--force`. Powered by Claude, GPT-4, and Gemini. | Yes | Yes | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Moonshot AI's coding CLI powered by Kimi K2. Long-context code understanding (256K tokens) with read-only operations auto-approved. | Yes | Yes | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | HashiCorp's AI agent CLI. Single-query mode via `chat -q`; agentic mode for multi-step infrastructure and code tasks. | Maybe | Yes | `hermes setup --portal` |
| [Aider](https://aider.chat) | Open-source AI pair programming tool. Works with Git, supports GPT-4o, Claude 3.5, DeepSeek, and local LLMs. Edit-first design — modifies files by default. | No | Yes | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | Open-source AI coding agent. Non-interactive `run` subcommand for single prompts; agentic mode for autonomous development. Supports multiple LLM providers. | Maybe | Yes | See [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> **Read-Only** indicates whether the CLI can safely operate in responder mode (no file modifications). `Yes` = enforced read-only, `Maybe` = non-interactive mode but may trigger tools, `No` = modifies files by default (executor only).

---

## Commands

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

## Interactive REPL

Running `openconvene`, `openconvene ask`, or `openconvene agent` without a task argument enters an interactive REPL, similar to codex, grok, agy, and devin.

In the REPL, you can type prompts directly or use slash commands to switch settings:

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

> **REPL Features**: fish-style menu-complete (Tab shows completion menu, Up/Down arrows navigate candidates, Enter confirms, Shift-Tab cycles backward), incremental history search (Ctrl-R/Ctrl-S), cross-session command history. Powered by [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4.

### Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/help` | `/h`, `/?` | Show all available commands |
| `/status` | | Show session status (mode, models, run count) |
| `/mode [ask\|code\|agent]` | | Show or switch current mode |
| `/models` | `/m` | List all configured models |
| `/responders [a,b,c]` | | Show or set responders |
| `/executor [name]` | | Show or set executor |
| `/synthesizer [name]` | | Show or set synthesizer (`none` to clear) |
| `/language [lang]` | `/lang` | Show or set model response language |
| `/usage` | `/u` | Show per-CLI usage statistics |
| `/config` | `/c`, `/settings` | Show current configuration summary |
| `/detect` | `/d` | Detect installed CLIs |
| `/clear` | `/new` | Clear screen and reset session |
| `/compact` | | (stub) Summarize conversation to free tokens |
| `/resume` | `/continue` | (stub) Resume a previous session |
| `/update` | | (stub) Check and install updates |
| `/exit` | `/quit`, `/q` | Exit REPL |

---

## CLI Flags

| Flag | Description |
|------|-------------|
| `-p`, `--print` | Non-interactive single-shot mode |
| `-m`, `--model <name>` | Specify model (alias for `--executor`) |
| `--json` | JSON output format |
| `--responders <a,b,c>` | Specify responders |
| `--executor <name>` | Specify executor |
| `--synthesizer <name>` | Specify synthesizer |
| `--config <path>` | Specify config file path |
| `--timeout <sec>` | Override timeout |
| `--verbose` | Show raw responses and metadata |
| `--language <lang>` | Set model response language |
| `--` | Separator (add before prompt) |

---

## Why Go

- **Single static binary** — compiled output has zero runtime dependencies; `curl + chmod` and it works
- **Goroutines for native concurrency** — N responders fan-out in parallel, lighter than Python asyncio
- **Fast startup** — ~5ms launch, ideal for CLI use
- **Static typing** — strong-typed structs replace maps, refactoring is safe
- **Cross-platform** — `GOOS=windows/linux/darwin` one-command cross-compilation

---

## Documentation

| Document | Content |
|----------|---------|
| [Overview](Docs/00-Overview.md) | Design motivation, comparison with Fusion/MoA |
| [Architecture](Docs/01-Architecture.md) | System architecture, Go module structure, data flow |
| [Usage Guide](Docs/02-Usage-Guide.md) | Complete usage guide (install, config, flags, modes) |
| [Model Adapters](Docs/03-Model-Adapters.md) | 9 CLI adapter designs, read-only capability matrix |
| [Configuration](Docs/04-Configuration.md) | Full `models.yaml` schema + examples |
| [Examples](Docs/05-Examples.md) | Real-world usage examples for each mode |
| [Troubleshooting](Docs/06-Troubleshooting.md) | Common issues and solutions |

---

## License

[MIT](LICENSE)
