# envonce

[![CI](https://github.com/laidbackgeek/envonce/actions/workflows/ci.yml/badge.svg)](https://github.com/laidbackgeek/envonce/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**English** | [简体中文](README.md)

> env once — configure once, apply everywhere.

`envonce` is a macOS command-line tool for **unified management of environment variables across your shell and launchd background services**. It solves two pain points:

1. Your interactive shell and multiple background services each maintain their own env, making consistency hard.
2. Manually editing a Homebrew-managed plist gets overwritten by `brew upgrade`.

## Features

- 🎯 **Configure once, apply everywhere** — a single env source drives both your interactive shell and `launchd` background services
- 🍺 **Take over brew services** — `envonce` maintains the plist + wrapper; `brew upgrade` only swaps the binary and never overwrites your env
- 🔐 **Keychain integration** — sensitive values are referenced via `@keychain:`; secrets never touch disk
- 🧰 **Manual services too** — `service add` defines custom (non-brew) background services
- 📦 **Multiple env groups** — manage separate variable sets per project/service with groups
- 🩺 **Fail-loud diagnostics** — `doctor` self-checks, status labels, and error messages are all readable and localized
- 🌐 **Bilingual** (Chinese / English) interface

## Origin story

My motivation for building this tool: when I started ollama via Homebrew, the model files were downloaded to my local disk instead of the external drive I used when testing in the shell. The reason was that the `OLLAMA_MODELS` I configured in `~/.zprofile` had no effect on the ollama instance launched by Homebrew. The same problem affects my own projects too.

I considered a few workarounds. Editing ollama's plist directly to add the env var — but `brew upgrade ollama` would overwrite that plist and lose the config. Or adding a dedicated plist to inject global env vars — but that introduces race conditions.

I wanted a way to configure environment variables in one place and have them take effect both in the shell and in processes launched by launchd. So, for my use case — developing and testing multiple projects on a Mac that run both in the shell (for dev/debugging) and as launchd background services — I built this CLI.

## Prerequisites

- **macOS** — `envonce` relies on `launchd` / `launchctl`; macOS only (not Linux / Windows).
- **Homebrew** (optional) — required for `service take` to take over a brew service; not needed if you only use shell integration.
- **Go 1.21+** (optional) — only needed if you install via `go install`.

## Installation

Choose one of two methods:

**Homebrew tap (recommended)**

```bash
brew tap laidbackgeek/homebrew-tap
brew install envonce
```

After installation, run `envonce init` as noted below.

**go install (requires Go 1.21+ toolchain, suited for Go developers)**

```bash
go install github.com/laidbackgeek/envonce/cmd/envonce@latest
```

`go install` places the binary in `$GOPATH/bin` (`~/go/bin` by default). **That directory must be in your `$PATH`**, or your shell won't find the `envonce` command.

## Quick start (3 steps)

> Want your terminal to use the unified env too? Run `envonce init` — it appends `eval "$(envonce shell-init)"` to `~/.zshrc` (or `~/.bashrc`); new terminals pick it up automatically.

```bash
# 1. Initialize: create the directory skeleton + wire up the shell (edits ~/.zshrc, idempotent)
envonce init

# 2. Set an env var (writes to ~/.config/envonce/env.d/default.env)
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models

# 3. Take over the brew service (import + migrate plist env vars + stop brew + start under envonce)
envonce service take ollama
```

Done: new terminals auto-load the unified env; the ollama service picks up the custom model directory on its next restart, and `brew upgrade ollama` no longer overwrites it.

> For the full walkthrough (launchctl verification, Keychain secrets, multiple env groups), see [docs/usage.en.md](docs/usage.en.md).

## Command reference

| Command | Purpose |
|---|---|
| `envonce init [--uninstall]` | First-time setup: build the directory skeleton, detect `$SHELL`, append a marker-tagged `eval` line to your rc file (idempotent, uninstallable) |
| `envonce shell-init` | Emit shell `export` lines (resolved at runtime, includes Keychain). Usually called indirectly by `init`; run manually to preview / debug / wire up non-standard shells |
| `envonce env set KEY=VALUE [--group G]` | Write to env.d (default group by default) |
| `envonce env get KEY [--group G]` | Read |
| `envonce env unset KEY [--group G]` | Delete |
| `envonce env list [--group G]` | List |
| `envonce env export [--service X \| --groups g1,g2]` | Emit export lines (the shared resolution core used by both shell-init and the wrapper) |
| `envonce group create/list/rename/delete` | Group management |
| `envonce service take NAME` | Import from `brew info --json=v2`, `brew services stop`, generate plist + wrapper, bootstrap |
| `envonce service add NAME [-- BINARY_ARGS...]` | Manually define a non-brew service (`--binary`, `--keep-alive`, `--run-at-load`) |
| `envonce service drop NAME [--restore-brew]` | Unload plist + wrapper; `--restore-brew` hands control back to brew |
| `envonce service start/stop/restart NAME` | Lifecycle (restart applies changed env immediately) |
| `envonce service status NAME` | Run state + env resolution health check |
| `envonce service sync NAME` | Regenerate plist + wrapper from config.toml and reload (only needed when the binary path / args / groups change) |
| `envonce service list` | Managed-service inventory + status |
| `envonce doctor` | Self-check: initialization, brew reachable, security, brew leftover-plist collision, XDG drift |
| `envonce --version` / `-v` | Print the version |
| `envonce completion zsh\|bash\|fish\|powershell` | Generate shell completion scripts (cobra-provided) |

Every command supports `--help`, e.g. `envonce service take --help`.

## Interface language (i18n)

`envonce` supports both Chinese and English. Language-detection priority:

1. `--lang zh\|en` flag (explicit override, highest)
2. `ENVONCE_LANG` environment variable
3. `LC_ALL` → `LC_MESSAGES` → `LANG` (language part before `.`; `zh*` → Chinese, otherwise English)
4. Falls back to English

```bash
envonce --lang zh init      # force Chinese summaries
ENVONCE_LANG=en envonce init  # force English via env var
```

### Localization scope

`envonce` ships with **full Chinese/English bilingual** support — all of the following switch with `--lang` / the system `LANG`:

- **Help text** (`envonce --help`, each subcommand's `Short`/`Long`, every flag description)
- **Operation summaries** (structured `✓ what it did` / `next step` / `rollback` output)
- **Error messages** (e.g. `Unknown service: ...`)
- **Status labels** (`service status` / `list`: Running / Not loaded)
- **`doctor` check descriptions**, **first-run banner**

## Shell completion

`envonce` is built on cobra, which provides a `completion` subcommand that generates completion scripts (for command names, service names, and group names).

**zsh**

```bash
# 1. Create the completion directory (one time)
mkdir -p ~/.zsh/completion
# 2. Generate the completion script
envonce completion zsh > ~/.zsh/completion/_envonce
# 3. Add the directory to fpath (in ~/.zshrc, before compinit)
echo 'fpath+=~/.zsh/completion' >> ~/.zshrc
# 4. Reload
exec zsh
```

**bash**

```bash
mkdir -p ~/.bash_completion.d
envonce completion bash > ~/.bash_completion.d/envonce
echo 'source ~/.bash_completion.d/envonce' >> ~/.bashrc
exec bash
```

**fish** / **powershell**: run `envonce completion <shell>` and follow the header instructions in the output.

> Completion relies on cobra's runtime completion mechanism; the `envonce` binary must be in `$PATH`.

## More docs

- [User guide (docs/usage.en.md)](docs/usage.en.md) — full brew-service takeover flow, shell integration, Keychain setup, multiple env groups
- [Troubleshooting (docs/troubleshooting.en.md)](docs/troubleshooting.en.md) — fail-loud diagnostics, doctor interpretation, common conflicts

## License

MIT
