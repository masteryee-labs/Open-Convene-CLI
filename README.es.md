<div align="center">

# OpenConveneCLI

### Herramienta CLI de Colaboración Multi-Modelo AI — Orquesta N Agentes de Codificación AI vía CLIs Nativas

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#compilar-desde-el-codigo-fuente)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | **Español** | [Français](README.fr.md) | [Deutsch](README.de.md)

</div>

---

## Descripción General

**OpenConveneCLI** es una herramienta de línea de comandos de código abierto escrita en Go que implementa la **colaboración multi-modelo AI** — despachando el mismo prompt simultáneamente a N modelos respondedores (cada uno mediante su CLI nativa en modo de solo lectura), sintetizando sus respuestas en una conclusión unificada, y luego delegando a un modelo ejecutor que actúa sobre el resultado sintetizado (escribiendo código, modificando archivos o ejecutando tareas de agente de larga duración).

Este enfoque se alinea con [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) y [OpenRouter Fusion](https://openrouter.ai/), pero introduce una innovación clave: **CLI-as-Model** — en lugar de requerir una API unificada, orquesta la CLI nativa de cada modelo (Devin, Grok, Codex, Antigravity, Cursor, Kimi, Hermes, Aider, OpenCode). Incluso si un modelo carece de una API pública, siempre y cuando tenga una CLI, puede participar en la orquestación CLI de IA.

> **Palabras clave**: orquestación CLI de IA, colaboración multi-modelo AI, Mixture-of-Agents, MoA, generación de código AI, sistema multi-agente, CLI-as-Model, agente de codificación AI, orquestación LLM, fan-out AI

---

## Tabla de Contenidos

- [Inicio Rápido](#inicio-rapido)
- [Cómo Funciona](#como-funciona)
- [CLIs de IA Soportados](#clis-de-ia-soportados)
- [Comandos](#comandos)
- [REPL Interactivo](#repl-interactivo)
- [Banderas CLI](#banderas-cli)
- [Por Qué Go](#por-que-go)
- [Documentación](#documentacion)
- [Compilar desde el Código Fuente](#compilar-desde-el-codigo-fuente)
- [Licencia](#licencia)

---

## Inicio Rápido

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

> Los respondedores pueden usar cualquier CLI instalada: agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (se requiere al menos 1).

---

## Cómo Funciona

OpenConveneCLI ofrece tres modos que corresponden a los flujos de trabajo reales de los desarrolladores:

|| Modo | Comando | Pipeline | ¿Ejecuta? | Caso de Uso Típico |
||------|---------|----------|-----------|-----------------|
|| `ask` | `openconvene ask "..."` | N respondedores → sintetizador → imprimir conclusión | No | Investigación técnica, comparación de soluciones |
|| `code` (predeterminado) | `openconvene "..."` | N respondedores → sintetizador (opcional) → ejecutor escribe código | Sí — escribe código | Implementar funciones, corregir errores |
|| `agent` | `openconvene agent "..."` | N respondedores → sintetizador → ejecutor agente | Sí — modo agente | Tareas complejas de múltiples pasos |

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

## CLIs de IA Soportados

OpenConveneCLI soporta 9 CLIs de agentes de codificación AI listas para usar:

|| CLI | Solo Lectura | Ejecutor | Comando de Instalación |
||-----|-----------|----------|-----------------|
|| [Devin](https://devin.ai) | Sí | Sí | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
|| [Grok](https://x.ai) | Sí | Sí | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
|| [Codex](https://github.com/openai/codex) | Sí | Sí | `npm install -g @openai/codex` |
|| [Antigravity (agy)](https://antigravity.google) | Sí | Sí | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
|| [Cursor](https://cursor.com) | Sí | No | `curl https://cursor.com/install -fsS \| bash` |
|| [Kimi Code](https://code.kimi.com) | Sí | Sí | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
|| [Hermes](https://github.com/hashicorp/hermes) | Sí | Sí | `hermes setup --portal` |
|| [Aider](https://aider.chat) | Sí | Sí | `python -m pip install aider-install && aider-install` |
|| [OpenCode](https://opencode.ai) | Sí | Sí | Consulte [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> Cada CLI se conecta a su propio backend de modelo. OpenConveneCLI en sí no depende de ningún servicio en la nube.

---

## Comandos

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

## REPL Interactivo

Ejecutar `openconvene`, `openconvene ask` o `openconvene agent` sin un argumento de tarea inicia un REPL interactivo, similar a codex, grok, agy y devin.

En el REPL, puede escribir prompts directamente o usar comandos de barra diagonal para cambiar la configuración:

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

> **Características del REPL**: menu-complete estilo fish (Tab muestra el menú de autocompletado, flechas Arriba/Abajo navegan entre candidatos, Enter confirma, Shift-Tab recorre hacia atrás), búsqueda incremental en el historial (Ctrl-R/Ctrl-S), historial de comandos entre sesiones. Desarrollado con [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4.

### Comandos de Barra Diagonal

|| Comando | Alias | Descripción |
||---------|---------|-------------|
|| `/help` | `/h`, `/?` | Mostrar todos los comandos disponibles |
|| `/status` | | Mostrar estado de la sesión (modo, modelos, conteo de ejecuciones) |
|| `/mode [ask\|code\|agent]` | | Mostrar o cambiar el modo actual |
|| `/models` | `/m` | Listar todos los modelos configurados |
|| `/responders [a,b,c]` | | Mostrar o establecer los respondedores |
|| `/executor [name]` | | Mostrar o establecer el ejecutor |
|| `/synthesizer [name]` | | Mostrar o establecer el sintetizador (`none` para limpiar) |
|| `/language [lang]` | `/lang` | Mostrar o establecer el idioma de respuesta del modelo |
|| `/usage` | `/u` | Mostrar estadísticas de uso por CLI |
|| `/config` | `/c`, `/settings` | Mostrar resumen de la configuración actual |
|| `/detect` | `/d` | Detectar CLIs instaladas |
|| `/clear` | `/new` | Limpiar pantalla y reiniciar sesión |
|| `/compact` | | (stub) Resumir conversación para liberar tokens |
|| `/resume` | `/continue` | (stub) Reanudar una sesión anterior |
|| `/update` | | (stub) Verificar e instalar actualizaciones |
|| `/exit` | `/quit`, `/q` | Salir del REPL |

---

## Banderas CLI

|| Bandera | Descripción |
||------|-------------|
|| `-p`, `--print` | Modo de ejecución única no interactiva |
|| `-m`, `--model <name>` | Especificar modelo (alias de `--executor`) |
|| `--json` | Formato de salida JSON |
|| `--responders <a,b,c>` | Especificar respondedores |
|| `--executor <name>` | Especificar ejecutor |
|| `--synthesizer <name>` | Especificar sintetizador |
|| `--config <path>` | Especificar ruta del archivo de configuración |
|| `--timeout <sec>` | Sobrescribir el tiempo de espera |
|| `--verbose` | Mostrar respuestas sin procesar y metadatos |
|| `--language <lang>` | Establecer el idioma de respuesta del modelo |
|| `--` | Separador (añadir antes del prompt) |

---

## Por Qué Go

- **Binario estático único** — el resultado compilado no tiene dependencias de tiempo de ejecución; `curl + chmod` y funciona
- **Goroutines para concurrencia nativa** — N respondedores se despliegan en paralelo, más ligero que Python asyncio
- **Inicio rápido** — ~5ms de lanzamiento, ideal para uso en CLI
- **Tipado estático** — estructuras fuertemente tipadas reemplazan maps, el refactor es seguro
- **Multiplataforma** — `GOOS=windows/linux/darwin` compilación cruzada con un solo comando

---

## Documentación

|| Documento | Contenido |
||----------|---------|
|| [Overview](Docs/00-Overview.md) | Motivación de diseño, comparación con Fusion/MoA |
|| [Architecture](Docs/01-Architecture.md) | Arquitectura del sistema, estructura de módulos Go, flujo de datos |
|| [Usage Guide](Docs/02-Usage-Guide.md) | Guía de uso completa (instalación, configuración, banderas, modos) |
|| [Model Adapters](Docs/03-Model-Adapters.md) | Diseño de 9 adaptadores CLI, matriz de capacidades de solo lectura |
|| [Configuration](Docs/04-Configuration.md) | Esquema completo de `models.yaml` + ejemplos |
|| [Examples](Docs/05-Examples.md) | Ejemplos de uso reales para cada modo |
|| [Troubleshooting](Docs/06-Troubleshooting.md) | Problemas comunes y soluciones |

---

## Compilar desde el Código Fuente

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> Requisito previo: Go 1.24+

---

## Licencia

[MIT](LICENSE)
