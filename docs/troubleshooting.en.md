# Troubleshooting

**English** | [简体中文](troubleshooting.md)

`envonce` is designed to **fail loud** (report problems explicitly, never swallow them silently). This document covers the most common failures and how to diagnose them.

> Not sure what's wrong? Start by running `envonce doctor` (below) for a round of self-checks.

## Contents

- [Service won't start / exits immediately: check *.err.log](#service-wont-start--exits-immediately-check-errlog)
- [Interpreting envonce doctor output](#interpreting-envonce-doctor-output)
- [brew leftover plist collision](#brew-leftover-plist-collision)
- [Keychain authorization prompt](#keychain-authorization-prompt)
- [XDG_CONFIG_HOME drift](#xdg_config_home-drift)
- [env resolution failure (fail-loud)](#env-resolution-failure-fail-loud)
- [i18n bilingual support](#i18n-bilingual-support)
- [Other common questions](#other-common-questions)

---

## Service won't start / exits immediately: check *.err.log

A managed service's stdout/stderr is collected by launchd under `~/.config/envonce/logs/`:

```
~/.config/envonce/logs/
├── envonce.log          # envonce's own operation log (incl. env-resolution warnings)
├── ollama.out.log       # service stdout
└── ollama.err.log       # service stderr (incl. the wrapper's env-resolution diagnostics)
```

**Your first step is always the `*.err.log`**. On env-resolution failure, envonce's wrapper does `exit 1` and writes detailed diagnostics to stderr (i.e. `.err.log`):

```bash
tail -50 ~/.config/envonce/logs/ollama.err.log
```

Typical diagnostic lines (internal-library errors are always in English):

```
Error: ollama: GITHUB_TOKEN=@keychain:github-token failed to resolve: exit status 44
Error: env.d:3: invalid line "..." (missing '=')
```

> Note: a missing group file does not appear here — it's silently ignored (treated as an empty group), with no error.

### Crash-loop under KeepAlive

If the service has `keep_alive = true` (the default), a resolution-failure `exit 1` causes launchd to retry repeatedly on its `ThrottleInterval` (10 seconds by default). Failure messages accumulate in `.err.log`, and `envonce service status` shows signs of repeated restarts.

**Stop the bleeding temporarily**:

```bash
envonce service stop ollama      # bootout, halting launchd management
# after fixing the env / Keychain
envonce service start ollama
```

If you'd rather it not auto-restart on failure, set `keep_alive = false` for that service (edit `config.toml`, then `envonce service sync <name>`) and retry manually with `service start`.

---

## Interpreting envonce doctor output

```bash
envonce doctor
```

Runs a batch of self-checks with fix suggestions. Typical output:

```
✗ initialized   envonce init not run       # init not run → run envonce init
✓ brew                                    # brew reachable
✓ security                                # /usr/bin/security present
```

Meaning of each item (doctor runs these 5 checks; it does not check file permissions):

| Check | Failure means | Suggestion |
|---|---|---|
| `initialized` | `.initialized` marker missing | `envonce init` |
| `brew` | `brew` not found or not executable | Confirm Homebrew is installed and in `$PATH` |
| `security` | `/usr/bin/security` missing (the Keychain tool) | Present on any normal macOS; very rare |
| `brew-plist-collision:<svc>` | A taken-over service still has `homebrew.mxcl.<svc>.plist` | See [brew leftover plist collision](#brew-leftover-plist-collision) |
| `xdg-drift` | The ConfigDir recorded in `.initialized` differs from the current one | See [XDG_CONFIG_HOME drift](#xdg_config_home-drift) |

> `doctor` is read-only — it changes nothing and is safe to re-run. After fixing, re-run to confirm `✓`.

---

## brew leftover plist collision

**Symptom**: You ran `envonce service take ollama` to take over the service, then manually ran `brew services start ollama`. brew rebuilt `~/Library/LaunchAgents/homebrew.mxcl.ollama.plist`, which may now fight `com.envonce.ollama.plist` for the port/resources.

**Detect**:

```bash
envonce doctor                    # warns about the leftover plist
envonce service status ollama     # status also flags it
ls ~/Library/LaunchAgents/ | grep ollama
# homebrew.mxcl.ollama.plist  ← leftover, should be removed
# com.envonce.ollama.plist    ← envonce's
```

**Fix**:

```bash
# 1. Stop brew's own management
brew services stop ollama
# 2. Confirm brew's plist is gone
ls ~/Library/LaunchAgents/ | grep ollama   # should be only com.envonce.*
# 3. Restart the envonce-managed service
envonce service restart ollama
```

> Root cause: after takeover, **don't operate that service with `brew services` anymore** (start/stop/restart can all rebuild brew's plist). Hand the service lifecycle to `envonce service start/stop/restart`.

---

## Keychain authorization prompt

**Symptom**: When `@keychain:<ref>` is resolved for the first time, macOS shows an "XXX wants to use the keychain" authorization dialog.

**This is normal**: the system confirms the first access to a Keychain item. Click **Always Allow** and the system writes an ACL, so subsequent accesses no longer prompt.

### If it prompts frequently

Background services (launched via the wrapper) also trigger the prompt when reading the Keychain. If you want a service process to access items silently:

1. Open the **Keychain Access** app
2. Find the item (the reference name you passed via `-s`)
3. Double-click → the **Access Control** tab
4. Add `envonce` and the relevant service binary (e.g. `/opt/homebrew/opt/ollama/bin/ollama`) to the "Always Allow" list

### Diagnosing Keychain resolution failure

```bash
# manually verify that a reference resolves
security find-generic-password -s github-token -w
# a non-zero exit code = resolution failure (item missing, access denied, etc.; the exact code comes from security)
```

If it returns a non-zero exit code (item-missing is the most common), the item hasn't been written into the Keychain yet — see [User guide · Keychain setup](usage.en.md#keychain-setup-sensitive-values).

---

## XDG_CONFIG_HOME drift

**Symptom**: You changed `XDG_CONFIG_HOME`, or switched users/machines, and now the absolute paths baked into the plist/wrapper (e.g. `/Users/old/.config/envonce/...`) point at locations that no longer exist, so the service can't find the wrapper.

> **doctor detects drift automatically**: the `xdg-drift` check in `envonce doctor` compares the ConfigDir recorded in `.initialized` against the current ConfigDir and warns on mismatch. Note — this only fires when `.initialized` was copied to the new path along with the config; if you only changed `XDG_CONFIG_HOME` without migrating the config, doctor won't find `.initialized` at the new path and reports `initialized` as failed (which hints at drift the same way). You can also confirm manually with `envonce env list` (empty or missing expected variables = drift).

**Fix**: Regenerate the wrapper and plist for each managed service (so the new path is re-baked):

```bash
envonce service sync ollama     # a single service
# or for all services (iterate config.toml)
for svc in $(envonce service list | awk 'NR>1{print $1}'); do
  envonce service sync "$svc"
done
```

> Root cause: paths inside the plist/wrapper are expanded to absolute paths at generation time (launchd requires this). After `XDG_CONFIG_HOME` changes, the old absolute paths are invalid. Changing env doesn't trigger this (env is resolved at runtime); only path/binary/args/group changes need `sync`.

---

## env resolution failure (fail-loud)

Whenever `envonce` hits an error resolving env, it **exits non-zero** with detailed diagnostics — it never starts a service with a half-resolved env.

Common causes (only these two fail loud):

| Cause | Diagnostic example | Fix |
|---|---|---|
| Keychain reference failed to resolve (item missing or denied) | `Error: ollama: KEY=@keychain:ref failed to resolve: exit status 44` (the code is whatever `security` returned) | `security add-generic-password -s <ref> -w <value>`; see [Keychain authorization prompt · Diagnosing resolution failure](#diagnosing-keychain-resolution-failure) |
| Syntax error in an env.d file (invalid line) | `Error: env.d:3: invalid line "..." (missing '=')` | Edit the file (lines must be `KEY=VALUE` or `#` comments) |

> **Note: a missing group file is not an error.** If `config.toml` references a group `work` but `env.d/work.env` doesn't exist, that group is **silently ignored** (treated as empty), with no error. If a variable isn't taking effect, check that the group declared by the service has a corresponding `env.d/<group>.env` file.

**Reproduce the resolution manually** (bypassing launchd, to see the output directly):

```bash
envonce env export --service ollama      # the full env resolution for that service
envonce env export --groups default,work # a specified group merge
envonce shell-init                        # the shell-side resolution
```

These commands exit non-zero on resolution failure and print diagnostics to stderr — they're your best tool for debugging "service won't start".

---

## i18n bilingual support

`envonce` ships with **full Chinese/English bilingual** support (auto-detected via `--lang` / `ENVONCE_LANG` / `LANG`). All of the following switch with the language:

- **Help text** (`envonce --help`, each subcommand's `Short`/`Long`, every flag description)
- **Operation summaries** (`✓ what it did` / `next step` / `rollback`)
- **Error messages** (e.g. `envonce --lang en service start nope` → `Unknown service: nope`)
- **Status labels** (`service status`/`list`: `Running`/`Not loaded`)
- **`doctor` check descriptions**, **first-run banner**

**Implementation note**: cobra's `Short`/`Long` are frozen during command-tree construction, before language detection, and `--help` short-circuits `PersistentPreRunE` (so language detection never runs). `envonce` works around this with a custom `HelpFunc` that re-sets the language after flag parsing and re-translates the command-tree text by annotation (`applyI18n`), so both the `--help` path and the normal-execution path switch correctly. Command names and flag names are always English/ASCII and scriptable.

**Internal-library errors**: error messages from internal libraries like `envfile`/`resolver` stay in English (Go-library convention) and don't go through the i18n catalog — these are edge-case diagnostics (e.g. an `env.d` syntax error) that normal usage never triggers.

---

## Other common questions

### `envonce` says "not initialized yet"

Whenever a subcommand detects that `.initialized` is missing, it first prints a prominent notice (switching language with `--lang`):

```
⚠ envonce is not initialized — run `envonce init` to set up shell integration and directories
```

This is a fallback reminder (independent of how you installed). Just run `envonce init`.

### I edited config.toml but the service didn't change

Changing the binary path / args / a service's extra groups in `config.toml` → you need `envonce service sync <name>` to regenerate the wrapper and plist (because those are baked into the wrapper). Changing **env values** does not need sync (env is resolved at runtime) — just `service restart`.

### I want to fully uninstall

```bash
envonce service drop <name>          # unload each managed service
envonce init --uninstall             # remove the shell-integration line (keeps config and data)
# to also remove the configuration:
rm -rf ~/.config/envonce
rm ~/Library/LaunchAgents/com.envonce.*.plist
```

`service drop <name> --restore-brew` hands the service back to brew management (re-running `brew services start`).
