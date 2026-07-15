# User guide

**English** | [简体中文](usage.md)

This document covers the core use cases of `envonce`: taking over a Homebrew service (using ollama as the example), shell-integration details, Keychain secret management, and working with multiple env groups.

> For a command cheat sheet, see the [README](../README.en.md#command-reference). If this is your first time, run `envonce init` first.

## Directory layout

All of `envonce`'s data lives under `${XDG_CONFIG_HOME:-$HOME/.config}/envonce/` (referred to below as `~/.config/envonce/`):

```
~/.config/envonce/
├── config.toml            # wiring: service definitions, group assignments, shell group selection
├── env.d/                 # plain env data, one file per group
│   ├── default.env        #   full-injection group (shell + all managed services, mandatory)
│   ├── work.env
│   └── golang.env
├── services/              # generated artifacts (managed — don't hand-edit)
│   └── ollama.wrapper.sh  #   wrapper script
├── logs/                  # operation log + per-service stdout/stderr
│   ├── envonce.log
│   ├── ollama.out.log
│   └── ollama.err.log
├── state/                 # runtime state
└── .initialized           # init completion marker
```

The plist, as launchd requires, lives at `~/Library/LaunchAgents/com.envonce.<service>.plist` (absolute path) and points at the wrapper under `services/`.

> If you set `XDG_CONFIG_HOME`, the whole tree moves with it; the plist/wrapper expand to absolute paths. `doctor` detects XDG drift (the `xdg-drift` check compares the path recorded in `.initialized` against the current ConfigDir); if you only changed `XDG_CONFIG_HOME` without migrating the config, `doctor` reports `initialized` as failed and hints at drift too — see [Troubleshooting](troubleshooting.en.md#xdg_config_home-drift).

---

## Taking over a brew service: the full ollama flow

This is `envonce`'s core scenario: let brew handle only binary updates while envonce owns the plist and the environment variables.

### 1. Initialize (if you haven't)

```bash
envonce init
```

Creates the directory skeleton, `env.d/default.env`, and a default `config.toml`, and appends the shell-integration line to `~/.zshrc` (marker: `# >>> envonce >>>`). Idempotent — safe to re-run.

### 2. Configure environment variables

Write `OLLAMA_MODELS` into the `default` group (both the shell and every managed service will receive it):

```bash
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models
```

You can store plaintext non-sensitive values, or Keychain references (see [Keychain setup](#keychain-setup-sensitive-values) below):

```bash
envonce env set GOPATH=$HOME/go
envonce env set GITHUB_TOKEN=@keychain:github-token
```

> `$VAR` inside a value is expanded by the consuming shell/service process; `envonce` writes it verbatim. Quote values containing spaces yourself.

### 3. Take over the service

```bash
envonce service take ollama
```

This single command does 7 things:

1. Reads `~/Library/LaunchAgents/homebrew.mxcl.ollama.plist` (the launchd definition brew generated) → parses the service definition (run command, keep_alive, run_at_load, log paths, `EnvironmentVariables`)
2. Converts the binary path **to its `opt/` symlink form** (`/opt/homebrew/opt/ollama/bin/ollama`), so upgrades are followed automatically
3. Writes `[services.ollama]` (`source="brew"`, `args=["serve"]`) into `config.toml`
4. **Migrates** the plist's `EnvironmentVariables` into a service-name group (`env.d/ollama.env`), referenced by `services.ollama.groups` — once taken over, these variables are managed centrally by envonce and no longer depend on the brew plist (they survive upgrades too). Same-named keys the user already configured in the group are not overwritten
5. `brew services stop ollama` (unloads brew's plist, detaching brew's management)
6. Generates `services/ollama.wrapper.sh` + `~/Library/LaunchAgents/com.envonce.ollama.plist`
7. `launchctl bootstrap gui/$UID` to start it

### 4. Verify

**Check the service status**

```bash
envonce service status ollama
```

Outputs the load state + an env-resolution health check (confirms `OLLAMA_MODELS` resolves correctly).

**Cross-check directly with launchctl**

```bash
launchctl print gui/$(id -u)/com.envonce.ollama
```

You can see `program` pointing at `~/.config/envonce/services/ollama.wrapper.sh` (not brew's plist).

**Confirm brew no longer manages it**

```bash
ls ~/Library/LaunchAgents/ | grep ollama
# you should see only com.envonce.ollama.plist; homebrew.mxcl.ollama.plist is gone
```

### 5. Applying changed env

Changing env touches neither the plist nor the wrapper (env is resolved at runtime):

- **Shell**: new terminals pick it up immediately (`shell-init` resolves at runtime).
- **Service**: `envonce service restart ollama` applies the change immediately (no `sync` needed).

```bash
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models
envonce service restart ollama
```

### 6. brew upgrade behavior

```bash
brew upgrade ollama
```

The upgrade refreshes the Cellar and the `opt/ollama` symlink → the wrapper picks up the new version automatically; the plist/wrapper are unaffected; **brew will not rebuild `com.envonce.ollama.plist`** (the takeover already ran `brew services stop`, removing brew's plist).

**Your `OLLAMA_MODELS` config is preserved — the upgrade-overwrite problem is gone.**

> If you accidentally run `brew services start ollama` again on an already-taken-over service, brew rebuilds `homebrew.mxcl.ollama.plist`, which conflicts with `com.envonce.ollama` for the port. `envonce doctor` / `service status` detect the leftover plist and warn — see [Troubleshooting](troubleshooting.en.md#brew-leftover-plist-collision).

---

## Shell integration

`envonce init` appends a marker-tagged integration line to your rc file based on `$SHELL`:

```sh
# >>> envonce >>>
eval "$(envonce shell-init)"
# <<< envonce <<<
```

- **zsh**: writes to `~/.zshrc`
- **bash**: writes to `~/.bashrc`

The integration line is **idempotent**: re-running `init` won't append duplicates. `envonce init --uninstall` removes exactly those lines by marker (keeping your config and data).

### What `shell-init` does

`envonce shell-init` emits standard `export KEY=VALUE` lines (resolved at runtime from `[shell].groups` in `config.toml`, defaulting to `["default"]`). After `eval`, these variables enter the shell process's memory.

Manual preview / debugging:

```bash
envonce shell-init          # see which exports would be injected
envonce shell-init | sh -x  # trace the resolution process
```

### Non-zsh/bash or non-standard shell wiring

If your shell isn't zsh/bash (or your rc-file path is non-standard), just add `eval "$(envonce shell-init)"` to your startup script manually. `shell-init` outputs POSIX `export`, compatible with any POSIX shell.

---

## Keychain setup (sensitive values)

`envonce`'s principle: **secrets never touch disk**. Sensitive values are expressed as `@keychain:<ref>` placeholders — config files hold only the reference name, and the real value is resolved from the macOS Keychain into process memory only at consumption time (shell startup / service start).

### 1. Write a value into the Keychain

```bash
# -s is the reference name (service name), -w is the value; -a is the account name (optional)
security add-generic-password -s github-token -w 'ghp_xxxxxxxxxxxxxxxx'
security add-generic-password -s my-api-key -a envonce -w 'secret-value'
```

### 2. Reference it in envonce

```bash
envonce env set GITHUB_TOKEN=@keychain:github-token
envonce env set API_KEY=@keychain:my-api-key
```

`@keychain:<ref>` runs `security find-generic-password -s <ref> -w` at resolution time, emitting `export KEY='<resolved>'` (single-quoted to prevent expansion).

### 3. Verify resolution

```bash
# see what the shell will receive (shell-side default group)
envonce shell-init | grep GITHUB_TOKEN

# see the env resolution for a specific service
envonce env export --service ollama | grep GITHUB_TOKEN

# service status also runs an env health check
envonce service status ollama
```

### The authorization prompt on first resolution

The first time a Keychain item is read by `envonce` (or by a service process launched via the wrapper), macOS shows an authorization dialog asking for permission. Click **Always Allow**, and the system writes an ACL (access control list) so subsequent reads by that process no longer prompt.

> If the prompt appears too often, or you want background services to access the item silently, edit that item's Access Control in the Keychain Access app and add `envonce` and the relevant service binary to the allow list. `envonce doctor` pre-checks Keychain reachability.

---

## Multiple env groups

`envonce` supports **named env groups**. Key rules:

- **The `default` group is the full-injection group**: it's injected automatically into **every managed service** (mandatory, can't be opted out) and is the default group the shell loads. This is the glue that unifies shell ↔ launchd.
- **Other groups** are selected per service: a service can stack one or more "extra" groups in `config.toml`.

### A service's effective env = the default group + that service's extra groups

On same-key conflicts, **the latter overwrites the former**, with a warning logged to `logs/envonce.log` at resolution time.

### 1. Create a group and write variables

```bash
envonce group create work
envonce env set WORK_API_URL=https://api.work.local --group work
envonce env set DB_PASSWORD=@keychain:work-db-pw --group work
```

### 2. Stack extra groups onto a service

Edit `config.toml` and add `groups` to the service:

```toml
[services.ollama]
source     = "brew"
binary     = "/opt/homebrew/opt/ollama/bin/ollama"
args       = ["serve"]
groups     = ["work"]          # default is auto-injected; only list "extra" groups here
keep_alive = true
run_at_load = true
```

Then regenerate the wrapper (group changes require sync):

```bash
envonce service sync ollama
```

From then on the ollama service's env = the `default` group (incl. `OLLAMA_MODELS`, `GITHUB_TOKEN`, …) ⊕ the `work` group (incl. `WORK_API_URL`, `DB_PASSWORD`).

### 3. Load extra groups on the shell side (optional)

The shell loads only `default` by default. If you want your terminal to also receive certain extra groups, change `config.toml`:

```toml
[shell]
groups = ['default', 'golang']   # the shell explicitly lists every group to load (including default); unlike the service side, the shell does NOT auto-inject default
```

New terminals pick it up immediately (resolved at runtime, no regeneration needed).

> Note the difference between the shell side and the service side: **services** forcibly inject `default` (can't opt out) + extra groups; the **shell** lets you freely choose which groups to load in `config.toml`.

### Group-management commands

```bash
envonce group list                    # list all groups
envonce group create NAME             # create an empty group
envonce group rename OLD NEW          # rename
envonce group delete NAME             # delete the group (along with its variables)
envonce env list --group golang       # view a group's contents
envonce env export --groups default,work   # preview the merged exports
```

---

## Next steps

- Something wrong? See [Troubleshooting](troubleshooting.en.md).
