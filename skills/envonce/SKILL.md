---
name: envonce
description: Manage macOS environment variables across interactive shells and launchd background services with envonce. Trigger when starting/stopping/restarting macOS background services (especially Homebrew-installed ones like ollama, postgresql, redis, nginx), adding or changing a background service's environment variables, preventing brew upgrade from overwriting service config (taking over a brew service), troubleshooting "service won't start / env vars not applied / plist overwritten", managing secrets via KeyChain, or touching ~/Library/LaunchAgents plists or ~/.config/envonce/. Ironclad rule — never use `brew services start/stop/restart` on a service envonce has taken over: it rebuilds brew's plist, invalidating envonce-managed env vars and causing port conflicts; always use `envonce service start/stop/restart` instead. Use whenever launchd services, brew services, unified env, OLLAMA_MODELS and other service-level env vars, or sharing one set of env vars between the terminal and background services comes up.
---

# envonce

`envonce` unifies "the interactive shell's environment variables" with "launchd background services' environment variables" into one place (`~/.config/envonce/env.d/`), keeping them consistent and stopping `brew upgrade` from overwriting the env vars you configure for a service.

## ⚠️ The one rule (most important)

**Never use `brew services` on a service envonce has taken over** — not `start`, `stop`, `restart`, or `run`.

`brew services <cmd>` rebuilds `~/Library/LaunchAgents/homebrew.mxcl.<svc>.plist` — that plist does not carry envonce-managed env vars, and it competes with envonce's `com.envonce.<svc>.plist` for ports/resources. Result: **env vars "stop working", or the service fails to start due to a port conflict.**

Hand the service lifecycle to envonce:

| What you want | ❌ Wrong (breaks envonce) | ✅ Right |
|---|---|---|
| Restart so new env takes effect | `brew services restart ollama` | `envonce service restart ollama` |
| Stop the service | `brew services stop ollama` | `envonce service stop ollama` |
| Start the service | `brew services start ollama` | `envonce service start ollama` |
| Check status | `brew services info ollama` | `envonce service status ollama` |

> The only place `brew services` legitimately appears: `envonce service take` itself calls `brew services stop` once to detach brew management — envonce does that for you, **don't repeat it by hand**.

## Before you act: is this service managed by envonce?

```bash
envonce service list                            # list every managed service + status
ls ~/Library/LaunchAgents/ | grep com.envonce   # a com.envonce.<svc>.plist means envonce owns it
```

- **Managed** (in the list) → follow the rule above strictly; use only `envonce service ...`.
- **Not managed** (a plain brew service) → `brew services` is fine; but if you want it to share env with the shell or survive `brew upgrade`, take it over with `envonce service take <svc>`.

## Mental model (three lines)

1. **Env is resolved at runtime.** Changing an env value never touches the plist/wrapper; a new shell picks it up, and a service picks it up after `restart`.
2. **wrapper + plist are the wiring.** `com.envonce.<svc>.plist` → `~/.config/envonce/services/<svc>.wrapper.sh` → your binary. Changing binary/args/groups needs `sync` to regenerate both.
3. **The `default` group is the linchpin.** It is force-injected into every managed service and is what the shell loads by default → shell and services share the same env.

## Command cheat sheet

Init / shell:
- `envonce init` — create the directory skeleton + hook into `~/.zshrc` (idempotent; **edits your rc file**); `--uninstall` removes the integration line (keeps data)
- `envonce shell-init` — emit shell `export` lines (resolved at runtime; run manually to preview/debug)

Env vars (written to the `default` group by default):
- `envonce env set KEY=VALUE [--group G]` / `get` / `unset` / `list [--group G]`
- `envonce env export --service X` or `--groups g1,g2` — preview the merged exports for a service/groups (great for debugging)

Groups:
- `envonce group create|list|rename|delete NAME`

Services (the core):
- `envonce service take NAME` — take over a brew service (import definition, migrate plist env vars, stop brew, generate plist+wrapper, start)
- `envonce service add NAME [-- BINARY_ARGS...]` — define a non-brew service manually (`--binary`, `--keep-alive`, `--run-at-load`)
- `envonce service drop NAME [--restore-brew]` — uninstall; `--restore-brew` hands control back to brew
- `envonce service start|stop|restart NAME` — **lifecycle; replaces brew services**
- `envonce service status NAME` — run state + env-resolution health check
- `envonce service sync NAME` — regenerate plist+wrapper and reload after changing binary/args/groups
- `envonce service list` — managed inventory + status

Diagnostics:
- `envonce doctor` — read-only self-check (init, brew, security, stale-plist conflicts, XDG drift)
- `envonce --version` / `-v`

i18n: `--lang zh|en` (or the `ENVONCE_LANG` env var). Command and flag names are always English/ASCII, so they're scriptable.

## Common workflows

### Take over a brew service (so it uses unified env, survives upgrade)
```bash
envonce init                                   # first time only
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models
envonce service take ollama
envonce service status ollama                  # verify: load state + env health check
```

### Apply a changed env var to a service (no sync needed)
```bash
envonce env set OLLAMA_MODELS=/new/path
envonce service restart ollama                 # the shell side takes effect in a new terminal
```

### Add a service-specific var (not exposed to the shell)
```bash
envonce group create work
envonce env set WORK_API_URL=https://api.work.local --group work
# Edit ~/.config/envonce/config.toml and add under [services.ollama]: groups = ["work"]
envonce service sync ollama                    # a group change needs sync
```

### Secret values via KeyChain (never written to disk)
```bash
security add-generic-password -s github-token -w 'ghp_xxx'
envonce env set GITHUB_TOKEN=@keychain:github-token
envonce service restart ollama                 # re-resolve for the service
```

## Agent decision flow

```
Operating on a background service?
├─ Run `envonce service list` first to check if it's managed
├─ Managed   → envonce service start/stop/restart (NEVER brew services)
└─ Unmanaged → brew services is OK; take over with `service take` to unify env

Changing an env var?
├─ envonce env set KEY=VALUE [--group G]
├─ shell: open a new terminal
└─ service: envonce service restart (changing an env value NEVER needs sync)

Changing a service's binary / args / extra groups?
├─ Edit ~/.config/envonce/config.toml
└─ envonce service sync <svc>

Service won't start / env not applied?
├─ tail ~/.config/envonce/logs/<svc>.err.log   ← first step
├─ envonce service status <svc>
└─ envonce doctor
```

## Recovery after mistakenly using brew services

If `brew services start/stop/restart` was run on an already-managed service, fix it in order:

```bash
brew services stop <svc>                       # make brew drop its own plist
ls ~/Library/LaunchAgents/ | grep <svc>        # confirm only com.envonce.<svc>.plist remains
envonce service restart <svc>                  # restart the envonce-managed instance
envonce doctor                                 # confirm no stale conflict
```

## Troubleshooting entry points

- **Service won't start:** first `tail -50 ~/.config/envonce/logs/<svc>.err.log` (on env-resolution failure the wrapper `exit 1`s and writes the diagnosis here).
- **Env not applied:** `envonce env export --service <svc>` to see the resolved result; `envonce service status <svc>` for an env health check.
- **Full self-check:** `envonce doctor` (read-only, safe to re-run).
- **KeyChain resolution failure** (e.g. `exit status 44`): `security find-generic-password -s <ref> -w` to verify the item exists.
- Full docs: `docs/troubleshooting.md` and `docs/usage.md` in the repo ([GitHub](https://github.com/laidbackgeek/envonce)).

> On env-resolution failure envonce **fails loud** (non-zero exit; it never starts a service with a half-resolved env). But **a missing group file is not an error** — it's silently treated as an empty group. If a var isn't taking effect, first confirm the service's declared groups have a matching `env.d/<group>.env` file.
