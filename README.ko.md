<div align="center">

# OpenConveneCLI

### 다중 모델 AI 협업 CLI 도구 — 네이티브 CLI로 N개의 AI 코딩 에이전트 오케스트레이션

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | **한국어** | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## 개요

**OpenConveneCLI**는 **다중 모델 AI 협업**을 구현하는 오픈소스 Go 명령줄 도구입니다. 동일한 프롬프트를 N개의 응답자 모델에 동시에 전달하고(각 모델은 자체 네이티브 CLI를 통해 읽기 전용 모드로 실행), 응답들을 하나의 통합된 결론으로 종합한 뒤, 종합된 결과를 바탕으로 코드 작성, 파일 수정, 또는 장기 에이전트 작업 실행을 수행하는 실행자 모델에 위임합니다. 이는 AI CLI 오케스트레이션과 Mixture-of-Agents (MoA) 접근 방식을 결합한 AI 코드 생성 도구입니다.

이 접근 방식은 [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) 및 [OpenRouter Fusion](https://openrouter.ai/)과 방향성이 일치하지만, 핵심 혁신을 도입합니다: **CLI-as-Model** — 통합 API를 요구하는 대신, 각 모델의 네이티브 CLI(Devin, Grok, Codex, Antigravity, Cursor, Kimi, Hermes, Aider, OpenCode)를 오케스트레이션합니다. 모델에 공개 API가 없더라도 CLI만 있으면 협업에 참여할 수 있습니다.

> **키워드**: AI CLI 오케스트레이션, 다중 모델 AI 협업, Mixture-of-Agents, MoA, AI 코드 생성, 멀티 에이전트 시스템, CLI-as-Model, AI 코딩 에이전트, LLM 오케스트레이션, fan-out AI

---

## 목차

- [빠른 시작](#빠른-시작)
- [작동 방식](#작동-방식)
- [지원 AI CLI](#지원-ai-cli)
- [명령어](#명령어)
- [대화형 REPL](#대화형-repl)
- [CLI 플래그](#cli-플래그)
- [Go를 선택한 이유](#go를-선택한-이유)
- [문서](#문서)
- [소스 빌드](#소스-빌드)
- [라이선스](#라이선스)

---

## 빠른 시작

```bash
# 1. Install
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest

# 2. Detect installed AI CLIs
openconvene detect

# 3. Generate config
openconvene init --path ~/.config/openconvene/models.yaml

# 4. Run multi-model collaboration
openconvene ask "your question" --responders agy,grok

# 5. Write code (default code mode)
openconvene "fix the bug in foo.go"

# 6. Agent task
openconvene agent "deploy the app"
```

> 응답자는 설치된 모든 CLI를 사용할 수 있습니다: agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (최소 1개 필요).

---

## 작동 방식

OpenConveneCLI는 실제 개발자 워크플로에 대응하는 세 가지 모드를 제공합니다:

| 모드 | 명령어 | 파이프라인 | 실행 여부 | 일반적 사용 사례 |
|------|---------|----------|-----------|-----------------|
| `ask` | `openconvene ask "..."` | N개 응답자 → 종합자 → 결론 출력 | 아니오 | 기술 조사, 솔루션 비교 |
| `code` (기본) | `openconvene "..."` | N개 응답자 → 종합자 (선택) → 실행자가 코드 작성 | 예 — 코드 작성 | 기능 구현, 버그 수정 |
| `agent` | `openconvene agent "..."` | N개 응답자 → 종합자 → 실행자 에이전트 | 예 — 에이전트 모드 | 복잡한 다단계 작업 |

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

## 지원 AI CLI

OpenConveneCLI는 9개의 AI 코딩 에이전트 CLI를 즉시 지원합니다:

| CLI | 읽기 전용 | 실행자 | 설치 명령어 |
|-----|-----------|----------|-----------------|
| [Devin](https://devin.ai) | 예 | 예 | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | 예 | 예 | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | 예 | 예 | `npm install -g @openai/codex` |
| [Antigravity (agy)](https://antigravity.google) | 예 | 예 | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | 예 | 아니오 | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | 예 | 예 | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | 예 | 예 | `hermes setup --portal` |
| [Aider](https://aider.chat) | 예 | 예 | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | 예 | 예 | [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) 참조 |

> 각 CLI는 자체 모델 백엔드에 연결됩니다. OpenConveneCLI 자체는 어떤 클라우드 서비스에도 의존하지 않습니다.

---

## 명령어

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

## 대화형 REPL

`openconvene`, `openconvene ask`, 또는 `openconvene agent`를 작업 인수 없이 실행하면 codex, grok, agy, devin과 유사한 대화형 REPL에 진입합니다.

REPL에서 프롬프트를 직접 입력하거나 슬래시 명령어를 사용하여 설정을 전환할 수 있습니다:

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

> **REPL 기능**: fish 스타일 메뉴 완성(Tab으로 완성 메뉴 표시, 상/하 화살표로 후보 탐색, Enter로 확정, Shift-Tab으로 역방향 순환), 점진적 히스토리 검색(Ctrl-R/Ctrl-S), 세션 간 명령어 히스토리. [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4 기반.

### 슬래시 명령어

| 명령어 | 별칭 | 설명 |
|---------|---------|-------------|
| `/help` | `/h`, `/?` | 사용 가능한 모든 명령어 표시 |
| `/status` | | 세션 상태 표시 (모드, 모델, 실행 횟수) |
| `/mode [ask\|code\|agent]` | | 현재 모드 표시 또는 전환 |
| `/models` | `/m` | 구성된 모든 모델 목록 |
| `/responders [a,b,c]` | | 응답자 표시 또는 설정 |
| `/executor [name]` | | 실행자 표시 또는 설정 |
| `/synthesizer [name]` | | 종합자 표시 또는 설정 (`none`으로 해제) |
| `/language [lang]` | `/lang` | 모델 응답 언어 표시 또는 설정 |
| `/usage` | `/u` | CLI별 사용 통계 표시 |
| `/config` | `/c`, `/settings` | 현재 구성 요약 표시 |
| `/detect` | `/d` | 설치된 CLI 감지 |
| `/clear` | `/new` | 화면 지우기 및 세션 초기화 |
| `/compact` | | (스텁) 토큰 확보를 위해 대화 요약 |
| `/resume` | `/continue` | (스텁) 이전 세션 이어서 실행 |
| `/update` | | (스텁) 업데이트 확인 및 설치 |
| `/exit` | `/quit`, `/q` | REPL 종료 |

---

## CLI 플래그

| 플래그 | 설명 |
|------|-------------|
| `-p`, `--print` | 비대화형 단일 실행 모드 |
| `-m`, `--model <name>` | 모델 지정 (`--executor`의 별칭) |
| `--json` | JSON 출력 형식 |
| `--responders <a,b,c>` | 응답자 지정 |
| `--executor <name>` | 실행자 지정 |
| `--synthesizer <name>` | 종합자 지정 |
| `--config <path>` | 구성 파일 경로 지정 |
| `--timeout <sec>` | 타임아웃 재정의 |
| `--verbose` | 원본 응답 및 메타데이터 표시 |
| `--language <lang>` | 모델 응답 언어 설정 |
| `--` | 구분자 (프롬프트 앞에 추가) |

---

## Go를 선택한 이유

- **단일 정적 바이너리** — 컴파일 결과물은 런타임 의존성이 제로; `curl + chmod`로 바로 작동
- **고루틴으로 네이티브 동시성** — N개 응답자가 병렬로 fan-out, Python asyncio보다 가벼움
- **빠른 시작** — ~5ms 실행, CLI 사용에 이상적
- **정적 타이핑** — 강 타입 구조체가 맵을 대체, 리팩토링이 안전
- **크로스 플랫폼** — `GOOS=windows/linux/darwin` 원 명령 크로스 컴파일

---

## 문서

| 문서 | 내용 |
|----------|---------|
| [Overview](Docs/00-Overview.md) | 설계 동기, Fusion/MoA와의 비교 |
| [Architecture](Docs/01-Architecture.md) | 시스템 아키텍처, Go 모듈 구조, 데이터 흐름 |
| [Usage Guide](Docs/02-Usage-Guide.md) | 완전한 사용 가이드 (설치, 구성, 플래그, 모드) |
| [Model Adapters](Docs/03-Model-Adapters.md) | 9개 CLI 어댑터 설계, 읽기 전용 기능 매트릭스 |
| [Configuration](Docs/04-Configuration.md) | 전체 `models.yaml` 스키마 + 예시 |
| [Examples](Docs/05-Examples.md) | 각 모드별 실제 사용 예시 |
| [Troubleshooting](Docs/06-Troubleshooting.md) | 일반적인 문제 및 해결 방법 |

---

## 소스 빌드

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> 사전 요구 사항: Go 1.24+

---

## 라이선스

[MIT](LICENSE)
