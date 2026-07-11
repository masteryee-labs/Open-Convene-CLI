<div align="center">

# OpenConveneCLI

### Outil CLI de Collaboration Multi-Modèle IA — Orchestrez N Agents de Codage IA via CLIs Natives

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#build-from-source)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/masteryee-labs/open-convene-cli/pulls)

[English](README.md) | [繁體中文](README.zh-TW.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | **Français** | [Deutsch](README.de.md)

</div>

---

## Présentation

**OpenConveneCLI** est un outil en ligne de commande open-source écrit en Go qui met en œuvre la **collaboration multi-modèle IA** — il dispatche simultanément le même prompt vers N modèles répondants (chacun via son CLI nat en mode lecture seule), synthétise leurs réponses en une conclusion unifiée, puis délègue à un modèle exécuteur qui agit sur le résultat synthétisé (écriture de code, modification de fichiers, ou exécution de tâches d'agent à long horizon).

Cette approche s'inscrit dans la lignée de [Mixture-of-Agents (MoA)](https://arxiv.org/abs/2406.04692) et d'[OpenRouter Fusion](https://openrouter.ai/), mais introduit une innovation clé : **CLI-as-Model** — au lieu d'exiger une API unifiée, il orchestre le CLI natif de chaque modèle (Devin, Grok, Codex, Antigravity, Cursor, Kimi, Hermes, Aider, OpenCode). Même si un modèle ne dispose pas d'API publique, tant qu'il possède un CLI, il peut participer à la collaboration multi-modèle IA. L'orchestration CLI d'IA devient ainsi accessible à tout modèle disposant d'une interface en ligne de commande, élargissant considérablement l'écosystème de génération de code IA.

> **Mots-clés** : orchestration CLI d'IA, collaboration multi-modèle IA, Mixture-of-Agents, MoA, génération de code IA, système multi-agent, CLI-as-Model, agent de codage IA, orchestration LLM, fan-out IA

---

## Table des Matières

- [Démarrage Rapide](#démarrage-rapide)
- [Fonctionnement](#fonctionnement)
- [CLIs IA Pris en Charge](#clis-ia-pris-en-charge)
- [Commandes](#commandes)
- [REPL Interactif](#repl-interactif)
- [Options CLI](#options-cli)
- [Pourquoi Go](#pourquoi-go)
- [Documentation](#documentation)
- [Compilation depuis les Sources](#compilation-depuis-les-sources)
- [Licence](#licence)

---

## Démarrage Rapide

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

> Les modèles répondants peuvent utiliser n'importe quel CLI installé : agy / codex / devin / grok / cursor / kimi / hermes / aider / opencode (au moins 1 requis).

---

## Fonctionnement

OpenConveneCLI propose trois modes correspondant aux flux de travail réels des développeurs :

| Mode | Commande | Pipeline | Exécute ? | Cas d'usage typique |
|------|----------|----------|-----------|---------------------|
| `ask` | `openconvene ask "..."` | N répondants → synthétiseur → affiche la conclusion | Non | Recherche technique, comparaison de solutions |
| `code` (par défaut) | `openconvene "..."` | N répondants → synthétiseur (optionnel) → exécuteur écrit le code | Oui — écrit du code | Implémenter des fonctionnalités, corriger des bugs |
| `agent` | `openconvene agent "..."` | N répondants → synthétiseur → exécuteur en mode agent | Oui — mode agent | Tâches complexes multi-étapes |

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

## CLIs IA Pris en Charge

OpenConveneCLI prend en charge 9 CLIs d'agents de codage IA prêts à l'emploi :

| CLI | Lecture seule | Exécuteur | Commande d'installation |
|-----|---------------|-----------|--------------------------|
| [Devin](https://devin.ai) | Oui | Oui | `curl -fsSL https://cli.devin.ai/install.sh \| bash` |
| [Grok](https://x.ai) | Oui | Oui | `curl -fsSL https://x.ai/cli/install.sh \| bash` |
| [Codex](https://github.com/openai/codex) | Oui | Oui | `npm install -g @openai/codex` |
| [Antigravity (agy)](https://antigravity.google) | Oui | Oui | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` |
| [Cursor](https://cursor.com) | Oui | Non | `curl https://cursor.com/install -fsS \| bash` |
| [Kimi Code](https://code.kimi.com) | Oui | Oui | `curl -fsSL https://code.kimi.com/kimi-code/install.sh \| bash` |
| [Hermes](https://github.com/hashicorp/hermes) | Oui | Oui | `hermes setup --portal` |
| [Aider](https://aider.chat) | Oui | Oui | `python -m pip install aider-install && aider-install` |
| [OpenCode](https://opencode.ai) | Oui | Oui | Voir [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) |

> Chaque CLI se connecte à son propre backend de modèle. OpenConveneCLI lui-même ne dépend d'aucun service cloud.

---

## Commandes

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

## REPL Interactif

Lancer `openconvene`, `openconvene ask`, ou `openconvene agent` sans argument de tâche déclenche un REPL interactif, similaire à codex, grok, agy et devin.

Dans le REPL, vous pouvez saisir des prompts directement ou utiliser des slash-commands pour modifier les paramètres :

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

> **Fonctionnalités du REPL** : complétion par menu de type fish (Tab affiche le menu de complétion, flèches Haut/Bas naviguent entre les candidats, Entrée confirme, Shift-Tab parcourt en sens inverse), recherche incrémentale dans l'historique (Ctrl-R/Ctrl-S), historique de commandes inter-sessions. Propulsé par [`reeflective/readline`](https://github.com/reeflective/readline) v1.1.4.

### Slash-Commands

| Commande | Alias | Description |
|----------|-------|-------------|
| `/help` | `/h`, `/?` | Afficher toutes les commandes disponibles |
| `/status` | | Afficher le statut de la session (mode, modèles, nombre d'exécutions) |
| `/mode [ask\|code\|agent]` | | Afficher ou changer le mode courant |
| `/models` | `/m` | Lister tous les modèles configurés |
| `/responders [a,b,c]` | | Afficher ou définir les modèles répondants |
| `/executor [name]` | | Afficher ou définir l'exécuteur |
| `/synthesizer [name]` | | Afficher ou définir le synthétiseur (`none` pour effacer) |
| `/language [lang]` | `/lang` | Afficher ou définir la langue de réponse des modèles |
| `/usage` | `/u` | Afficher les statistiques d'utilisation par CLI |
| `/config` | `/c`, `/settings` | Afficher le résumé de la configuration courante |
| `/detect` | `/d` | Détecter les CLIs installés |
| `/clear` | `/new` | Effacer l'écran et réinitialiser la session |
| `/compact` | | (brouillon) Résumer la conversation pour libérer des tokens |
| `/resume` | `/continue` | (brouillon) Reprendre une session précédente |
| `/update` | | (brouillon) Vérifier et installer les mises à jour |
| `/exit` | `/quit`, `/q` | Quitter le REPL |

---

## Options CLI

| Option | Description |
|--------|-------------|
| `-p`, `--print` | Mode non-interactif à exécution unique |
| `-m`, `--model <name>` | Spécifier un modèle (alias pour `--executor`) |
| `--json` | Format de sortie JSON |
| `--responders <a,b,c>` | Spécifier les modèles répondants |
| `--executor <name>` | Spécifier l'exécuteur |
| `--synthesizer <name>` | Spécifier le synthétiseur |
| `--config <path>` | Spécifier le chemin du fichier de configuration |
| `--timeout <sec>` | Remplacer le délai d'attente |
| `--verbose` | Afficher les réponses brutes et les métadonnées |
| `--language <lang>` | Définir la langue de réponse des modèles |
| `--` | Séparateur (à ajouter avant le prompt) |

---

## Pourquoi Go

- **Binaire statique unique** — le résultat compilé n'a aucune dépendance d'exécution ; `curl + chmod` et il fonctionne
- **Goroutines pour la concurrence native** — N répondants se déploient en parallèle, plus léger que Python asyncio
- **Démarrage rapide** — ~5 ms au lancement, idéal pour un usage en CLI
- **Typage statique** — des structures fortement typées remplacent les maps, le refactoring est sûr
- **Multi-plateforme** — `GOOS=windows/linux/darwin` compilation croisée en une seule commande

---

## Documentation

| Document | Contenu |
|----------|---------|
| [Overview](Docs/00-Overview.md) | Motivation de conception, comparaison avec Fusion/MoA |
| [Architecture](Docs/01-Architecture.md) | Architecture système, structure des modules Go, flux de données |
| [Usage Guide](Docs/02-Usage-Guide.md) | Guide d'utilisation complet (installation, configuration, options, modes) |
| [Model Adapters](Docs/03-Model-Adapters.md) | Conception des 9 adaptateurs CLI, matrice des capacités en lecture seule |
| [Configuration](Docs/04-Configuration.md) | Schéma complet de `models.yaml` + exemples |
| [Examples](Docs/05-Examples.md) | Exemples d'utilisation réelle pour chaque mode |
| [Troubleshooting](Docs/06-Troubleshooting.md) | Problèmes courants et solutions |

---

## Compilation depuis les Sources

```bash
git clone https://github.com/masteryee-labs/open-convene-cli.git
cd open-convene-cli
go build -o openconvene ./cmd/openconvene
```

> Prérequis : Go 1.24+

---

## Licence

[MIT](LICENSE)
