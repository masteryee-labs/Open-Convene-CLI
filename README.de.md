<div align="center">

# OpenConveneCLI

### Multi-Modell-KI-Kollaborations-CLI-Tool — Orchestrieren Sie N KI-Coding-Agenten über native CLIs

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#aus-dem-quellcode-kompilieren)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | **Deutsch**

</div>

---

## Übersicht

**OpenConveneCLI** ist ein quelloffenes Go-Kommandozeilen-Tool, das **Multi-Modell-KI-Kollaboration** implementiert — es sendet denselben Prompt gleichzeitig an N Responder-Modelle (jeweils über die native CLI im Read-Only-Modus), synthetisiert deren Antworten zu einer einheitlichen Schlussfolgerung und delegiert anschließend an ein Executor-Modell, das auf dem synthetisierten Ergebnis agiert (Code schreibt, Dateien modifiziert oder langfristige Agent-Aufgaben ausführt). Diese **KI-CLI-Orchestrierung** ermöglicht eine flexible **Mixture-of-Agents (MoA)**-Architektur für die **KI-Codegenerierung**.

Dieser Ansatz orientiert sich an [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) und [OpenRouter Fusion](https://openrouter.ai/), führt jedoch eine zentrale Innovation ein: **CLI-as-Model** — anstatt eine einheitliche API vorauszusetzen, orchestriert es die native CLI jedes Modells (Devin, Grok, Codex, Antigravity, Cursor, Kimi, Hermes, Aider, OpenCode). Selbst wenn ein Modell über keine öffentliche API verfügt, kann es an der Kollaboration teilnehmen, solange es eine CLI besitzt.

> **Schlüsselwörter**: KI-CLI-Orchestrierung, Multi-Modell-KI-Kollaboration, Mixture-of-Agents, MoA, KI-Codegenerierung, Multi-Agenten-System, CLI-as-Model, KI-Coding-Agent, LLM-Orchestrierung, Fan-out-KI

---

## Inhaltsverzeichnis

- [Installation](#installation)
- [Schnellstart](#schnellstart)
- [Funktionsweise](#funktionsweise)
- [Unterstützte KI-CLIs](#unterstützte-ki-clis)
- [Befehle](#befehle)
- [Interaktives REPL](#interaktives-repl)
- [CLI-Flags](#cli-flags)
- [Warum Go](#warum-go)
- [Dokumentation](#dokumentation)
- [Lizenz](#lizenz)

---

## Installation

### Einzeilige Installation (empfohlen)

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### Mit Go installieren

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### Aus dem Quellcode kompilieren

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> Voraussetzung: Go 1.24+

---

## Schnellstart

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

> Responder können jede installierte CLI verwenden: agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (mindestens 1 erforderlich).

### Aktualisieren

Geben Sie im REPL `/update` ein, um den Update-Befehl für Ihre Plattform zu sehen. Oder führen Sie den Installationsbefehl erneut aus — er überschreibt die alte Binärdatei mit der neuesten Version.

---

## Funktionsweise

OpenConveneCLI bietet drei Modi, die echten Entwickler-Workflows entsprechen:

|| Modus | Befehl | Pipeline | Führt aus? | Typischer Anwendungsfall |
||------|---------|----------|-----------|-----------------|
|| `ask` | `openconvene ask "..."` | N Responder → Synthesizer → Ausgabe der Schlussfolgerung | Nein | Technische Recherche, Lösungsvergleich |
|| `code` (Standard) | `openconvene "..."` | N Responder → Synthesizer (optional) → Executor schreibt Code | Ja — schreibt Code | Funktionen implementieren, Bugs beheben |
|| `agent` | `openconvene agent "..."` | N Responder → Synthesizer → Executor-Agent | Ja — Agent-Modus | Komplexe mehrstufige Aufgaben |

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

## Unterstützte KI-CLIs

OpenConveneCLI unterstützt standardmäßig 9 KI-Coding-Agent-CLIs. Jede CLI verbindet sich mit ihrem eigenen Modell-Backend — OpenConveneCLI selbst ist von keinem Cloud-Dienst abhängig. Mindestens 1 CLI muss installiert sein, um das Tool zu nutzen.

| CLI | Beschreibung | Read-Only | Executor | Installation |
|-----|-------------|-----------|----------|--------------|
| [Devin](https://devin.ai) | Autonomer KI-Softwareingenieur von Cognition. Full-Stack-Coding-Agent mit Shell-Zugriff, Browser-Steuerung und Planung langfristiger Aufgaben. | Vielleicht | Ja | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | KI-Coding-CLI von xAI mit Grok-Modellen. Schnelles Reasoning und Code-Generierung mit Echtzeit-Wissenszugriff. | Vielleicht | Ja | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | Terminalbasierter Coding-Agent von OpenAI. Sandbox-Ausführung — `--sandbox read-only` für sichere Recherche, `workspace-write` für Code-Ausführung. | Ja | Ja | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | KI-Coding-Agent-CLI von Google mit Gemini. Multi-Datei-Bearbeitung, Code-Review und agentische Aufgaben-Ausführung (Gemini 2.5 Pro/Flash). | Vielleicht | Ja | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | AI-First-Code-Editor mit Agent-Modus. Read-Only-Analyse ohne `--force`; autonome Dateibearbeitung mit `--force`. Angetrieben von Claude, GPT-4 und Gemini. | Ja | Ja | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Coding-CLI von Moonshot AI mit Kimi K2. Long-Context-Code-Verständnis (256K Tokens), Read-Only-Operationen automatisch genehmigt. | Ja | Ja | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | KI-Agent-CLI von HashiCorp. Einzelabfrage-Modus via `chat -q`; agentischer Modus für Multi-Step-Infrastruktur- und Code-Aufgaben. | Vielleicht | Ja | `hermes setup --portal` |
| [Aider](https://aider.chat) | Open-Source-KI-Pair-Programming-Tool. Git-Integration, unterstützt GPT-4o, Claude 3.5, DeepSeek und lokale LLMs. Edit-First-Design — ändert standardmäßig Dateien. | Nein | Ja | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | Open-Source-KI-Coding-Agent. `run`-Subcommand für nicht-interaktive Single-Prompts; agentischer Modus für autonome Entwicklung. Unterstützt mehrere LLM-Provider. | Vielleicht | Ja | Siehe [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> **Read-Only** gibt an, ob die CLI sicher im Responder-Modus arbeiten kann (keine Dateimodifikationen). `Ja` = erzwungenes Read-Only, `Vielleicht` = nicht-interaktiver Modus, kann aber Tools auslösen, `Nein` = ändert standardmäßig Dateien (nur Executor).

---

## Befehle

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

## Interaktives REPL

Wenn Sie `openconvene`, `openconvene ask` oder `openconvene agent` ohne Aufgabenargument ausführen, wird ein interaktives REPL gestartet, ähnlich wie bei codex, grok, agy und devin.

Im REPL können Sie Prompts direkt eingeben oder Slash-Befehle verwenden, um Einstellungen zu wechseln:

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

> **REPL-Funktionen**: fish-artiges Menu-Complete (Tab zeigt das Vervollständigungsmenü, Up/Down-Pfeiltasten navigieren durch Kandidaten, Enter bestätigt, Shift-Tab blättert rückwärts), inkrementelle Verlaufssuche (Ctrl-R/Ctrl-S), sitzungsübergreifender Befehlsverlauf. Bereitgestellt von [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4.

### Slash-Befehle

|| Befehl | Aliase | Beschreibung |
||---------|---------|-------------|
|| `/help` | `/h`, `/?` | Alle verfügbaren Befehle anzeigen |
|| `/status` | | Sitzungsstatus anzeigen (Modus, Modelle, Ausführungsanzahl) |
|| `/mode [ask\|code\|agent]` | | Aktuellen Modus anzeigen oder wechseln |
|| `/models` | `/m` | Alle konfigurierten Modelle auflisten |
|| `/responders [a,b,c]` | | Responder anzeigen oder festlegen |
|| `/executor [name]` | | Executor anzeigen oder festlegen |
|| `/synthesizer [name]` | | Synthesizer anzeigen oder festlegen (`none` zum Löschen) |
|| `/language [lang]` | `/lang` | Modellsprache für Antworten anzeigen oder festlegen |
|| `/usage` | `/u` | Nutzungsstatistiken pro CLI anzeigen |
|| `/config` | `/c`, `/settings` | Aktuelle Konfigurationszusammenfassung anzeigen |
|| `/detect` | `/d` | Installierte CLIs erkennen |
|| `/clear` | `/new` | Bildschirm leeren und Sitzung zurücksetzen |
|| `/compact` | | (Stub) Konversation zusammenfassen, um Tokens freizugeben |
|| `/resume` | `/continue` | (Stub) Eine vorherige Sitzung fortsetzen |
|| `/update` | | (Stub) Updates prüfen und installieren |
|| `/exit` | `/quit`, `/q` | REPL beenden |

---

## CLI-Flags

|| Flag | Beschreibung |
||------|-------------|
|| `-p`, `--print` | Nicht-interaktiver Single-Shot-Modus |
|| `-m`, `--model <name>` | Modell angeben (Alias für `--executor`) |
|| `--json` | JSON-Ausgabeformat |
|| `--responders <a,b,c>` | Responder angeben |
|| `--executor <name>` | Executor angeben |
|| `--synthesizer <name>` | Synthesizer angeben |
|| `--config <path>` | Konfigurationsdateipfad angeben |
|| `--timeout <sec>` | Timeout überschreiben |
|| `--verbose` | Rohe Antworten und Metadaten anzeigen |
|| `--language <lang>` | Modellsprache für Antworten festlegen |
|| `--` | Trennzeichen (vor dem Prompt hinzufügen) |

---

## Warum Go

- **Einzelne statische Binärdatei** — die kompilierte Ausgabe hat keine Runtime-Abhängigkeiten; `curl + chmod` und es funktioniert
- **Goroutines für native Nebenläufigkeit** — N Responder fan-out parallel, leichter als Python asyncio
- **Schneller Start** — ~5ms Startzeit, ideal für CLI-Nutzung
- **Statische Typisierung** — stark typisierte Structs ersetzen Maps, Refactoring ist sicher
- **Plattformübergreifend** — `GOOS=windows/linux/darwin` Ein-Befehl-Kreuzkompilierung

---

## Dokumentation

|| Dokument | Inhalt |
||----------|---------|
|| [Overview](Docs/00-Overview.md) | Designmotivation, Vergleich mit Fusion/MoA |
|| [Architecture](Docs/01-Architecture.md) | Systemarchitektur, Go-Modulstruktur, Datenfluss |
|| [Usage Guide](Docs/02-Usage-Guide.md) | Vollständiger Nutzungsleitfaden (Installation, Konfiguration, Flags, Modi) |
|| [Model Adapters](Docs/03-Model-Adapters.md) | 9 CLI-Adapter-Designs, Read-Only-Fähigkeitsmatrix |
|| [Configuration](Docs/04-Configuration.md) | Vollständiges `models.yaml`-Schema + Beispiele |
|| [Examples](Docs/05-Examples.md) | Praxisnahe Anwendungsbeispiele für jeden Modus |
|| [Troubleshooting](Docs/06-Troubleshooting.md) | Häufige Probleme und Lösungen |

---

## Lizenz

[MIT](LICENSE)
