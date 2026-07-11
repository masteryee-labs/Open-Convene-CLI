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

- [Instalación](#instalacion)
- [Inicio Rápido](#inicio-rapido)
- [Cómo Funciona](#como-funciona)
- [CLIs de IA Soportados](#clis-de-ia-soportados)
- [Comandos](#comandos)
- [REPL Interactivo](#repl-interactivo)
- [Banderas CLI](#banderas-cli)
- [Por Qué Go](#por-que-go)
- [Documentación](#documentacion)
- [Licencia](#licencia)

---

## Instalación

### Instalación en una línea (recomendado)

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
```

### Instalar con Go

```bash
go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest
```

### Compilar desde el código fuente

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> Requisito previo: Go 1.24+

---

## Inicio Rápido

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

> Los respondedores pueden usar cualquier CLI instalada: agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (se requiere al menos 1).

### Actualizar

En el REPL, escriba `/update` para ver el comando de actualización de su plataforma. O ejecute nuevamente el comando de instalación — sobrescribirá el binario anterior con la última versión.

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

OpenConveneCLI soporta 9 CLIs de agentes de codificación AI listas para usar. Cada CLI se conecta a su propio backend de modelo — OpenConveneCLI en sí no depende de ningún servicio en la nube. Debe instalarse al menos 1 CLI para usar la herramienta.

| CLI | Descripción | Solo Lectura | Ejecutor | Instalación |
|-----|-------------|-----------|----------|---------|
| [Devin](https://devin.ai) | Ingeniero de software IA autónomo de Cognition. Agente de codificación full-stack con acceso a shell, control de navegador y planificación de tareas de larga duración. | Tal vez | Sí | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | CLI de codificación IA de xAI con modelos Grok. Razonamiento rápido y generación de código con acceso a conocimiento en tiempo real. | Tal vez | Sí | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | Agente de codificación basado en terminal de OpenAI. Ejecución en sandbox — `--sandbox read-only` para investigación segura, `workspace-write` para ejecución de código. | Sí | Sí | `npm install -g @openai/codex` |
| [Antigravity / agy](https://antigravity.google) | CLI de agente de codificación IA de Google con Gemini. Edición multi-archivo, revisión de código y ejecución de tareas agentic (Gemini 2.5 Pro/Flash). | Tal vez | Sí | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | Editor de código AI-first con modo agente. Análisis de solo lectura sin `--force`; edición autónoma de archivos con `--force`. Impulsado por Claude, GPT-4 y Gemini. | Sí | Sí | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | CLI de codificación de Moonshot AI con Kimi K2. Comprensión de código de contexto largo (256K tokens), operaciones de solo lectura aprobadas automáticamente. | Sí | Sí | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | CLI de agente IA de HashiCorp. Modo de consulta única vía `chat -q`; modo agentic para tareas de infraestructura y código de múltiples pasos. | Tal vez | Sí | `hermes setup --portal` |
| [Aider](https://aider.chat) | Herramienta de programación en pareja IA de código abierto. Integra con Git, soporta GPT-4o, Claude 3.5, DeepSeek y LLMs locales. Diseño de edición primero — modifica archivos por defecto. | No | Sí | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | Agente de codificación IA de código abierto. Subcomando `run` para prompts únicos no interactivos; modo agentic para desarrollo autónomo. Soporta múltiples proveedores de LLM. | Tal vez | Sí | Consulte [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> **Solo Lectura** indica si la CLI puede operar de forma segura en modo respondedor (sin modificaciones de archivos). `Sí` = solo lectura forzado, `Tal vez` = modo no interactivo pero puede activar herramientas, `No` = modifica archivos por defecto (solo ejecutor).

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

## Licencia

[MIT](LICENSE)
