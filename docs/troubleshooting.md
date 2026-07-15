# 故障排查

[English](troubleshooting.en.md) | **简体中文**

`envonce` 的设计原则是 **fail-loud**（明确报错、不静默吞掉问题）。本文档覆盖最常见的故障与排查路径。

> 不确定哪里出问题？先跑 `envonce doctor`（见下文）做一轮自检。

## 目录

- [服务起不来 / 立即退出：查 *.err.log](#服务起不来--立即退出查-errlog)
- [envonce doctor 输出解读](#envonce-doctor-输出解读)
- [brew 残留 plist 冲突](#brew-残留-plist-冲突)
- [KeyChain 授权弹窗](#keychain-授权弹窗)
- [XDG_CONFIG_HOME 漂移](#xdg_config_home-漂移)
- [env 解析失败（fail-loud）](#env-解析失败fail-loud)
- [i18n 双语支持](#i18n-双语支持)
- [其它常见问题](#其它常见问题)

---

## 服务起不来 / 立即退出：查 *.err.log

受管服务的 stdout/stderr 被 launchd 收进 `~/.config/envonce/logs/`：

```
~/.config/envonce/logs/
├── envonce.log          # envonce 自身操作日志（含 env 解析 warning）
├── ollama.out.log       # 服务 stdout
└── ollama.err.log       # 服务 stderr（含 wrapper 的 env 解析诊断）
```

**第一步永远是看 `*.err.log`**。envonce 的 wrapper 在 env 解析失败时会 `exit 1` 并把详细诊断打到 stderr（即 `.err.log`）：

```bash
tail -50 ~/.config/envonce/logs/ollama.err.log
```

典型诊断行（内部库错误，始终为英文）：

```
Error: ollama: GITHUB_TOKEN=@keychain:github-token failed to resolve: exit status 44
Error: env.d:3: invalid line "..." (missing '=')
```

> 注：缺失的组文件不会出现在这里 —— 它会被静默忽略（视为空组），不报错。

### KeepAlive 下的 crash-loop

若服务 `keep_alive = true`（默认），解析失败导致 `exit 1` 后，launchd 会按 `ThrottleInterval`（默认 10 秒）反复重试。失败信息会在 `.err.log` 累积，`envonce service status` 能看到反复重启的迹象。

**临时止血**：

```bash
envonce service stop ollama      # bootout，停止 launchd 管理
# 修复 env / KeyChain 后
envonce service start ollama
```

若希望失败后不自动拉起，对该服务设 `keep_alive = false`（编辑 `config.toml` 后 `envonce service sync <name>`），由你手动 `service start` 重试。

---

## envonce doctor 输出解读

```bash
envonce doctor
```

批量自检并给修复建议。典型输出：

```
✗ initialized 未运行 envonce init       # 未运行 init → 跑 envonce init
✓ brew                                   # brew 可达
✓ security                               # /usr/bin/security 存在
```

逐项含义（doctor 做这 5 项检查；不检测文件权限）：

| 检查项 | 失败含义 | 建议 |
|---|---|---|
| `initialized` | 缺 `.initialized` 标记 | `envonce init` |
| `brew` | 找不到 `brew` 或不可执行 | 确认 Homebrew 已安装且在 `$PATH` |
| `security` | `/usr/bin/security` 不存在（KeyChain 工具） | 正常 macOS 都有；极少见 |
| `brew-plist-collision:<svc>` | 已接管的服务仍存在 `homebrew.mxcl.<svc>.plist` | 见 [brew 残留 plist 冲突](#brew-残留-plist-冲突) |
| `xdg-drift` | `.initialized` 记录的 ConfigDir 与当前不一致 | 见 [XDG_CONFIG_HOME 漂移](#xdg_config_home-漂移) |

> `doctor` 是只读命令，不改任何东西，安全重复运行。修复后重跑确认 `✓`。

---

## brew 残留 plist 冲突

**症状**：已 `envonce service take ollama` 接管，但又手动跑了 `brew services start ollama`。brew 重建了 `~/Library/LaunchAgents/homebrew.mxcl.ollama.plist`，与 `com.envonce.ollama.plist` 可能抢端口/资源。

**检测**：

```bash
envonce doctor                    # 会警告残留 plist
envonce service status ollama     # status 也会提示
ls ~/Library/LaunchAgents/ | grep ollama
# homebrew.mxcl.ollama.plist  ← 残留，应清除
# com.envonce.ollama.plist    ← envonce 的
```

**修复**：

```bash
# 1. 让 brew 停掉它自己的管理
brew services stop ollama
# 2. 确认 brew 的 plist 已移除
ls ~/Library/LaunchAgents/ | grep ollama   # 应只剩 com.envonce.*
# 3. 重启 envonce 的服务
envonce service restart ollama
```

> 根因：接管后**不要再用 `brew services` 操作该服务**（start/stop/restart 都可能重建 brew 的 plist）。服务的生命周期交给 `envonce service start/stop/restart`。

---

## KeyChain 授权弹窗

**症状**：`@keychain:<ref>` 首次被解析时，macOS 弹「XXX 想要使用钥匙串」授权框。

**这是正常的**：首次访问某 KeyChain 项时系统会确认。点「**始终允许**」后系统写 ACL，后续访问不再弹窗。

### 如果频繁弹窗

后台服务（由 wrapper 启动）读取 KeyChain 时也会触发授权。若你希望服务进程静默访问：

1. 打开「钥匙串访问」App
2. 找到对应项（`-s` 指定的引用名）
3. 双击 → 「访问控制」标签
4. 把 `envonce` 与对应服务二进制（如 `/opt/homebrew/opt/ollama/bin/ollama`）加入「始终允许」列表

### KeyChain 解析失败的诊断

```bash
# 手动验证某引用能否解析
security find-generic-password -s github-token -w
# 非零退出码 = 解析失败（项不存在、被拒绝授权等；具体码值由 security 返回）
```

若返回非零退出码（项不存在最常见），说明该项还没写入 KeyChain —— 见 [用户指南 · KeyChain 设置](usage.md#keychain-设置敏感值)。

---

## XDG_CONFIG_HOME 漂移

**症状**：你改了 `XDG_CONFIG_HOME`，或换了用户/机器后，plist/wrapper 里烘焙的绝对路径（如 `/Users/old/.config/envonce/...`）指向了不存在的位置，服务找不到 wrapper。

> **doctor 会自动检测漂移**：`envonce doctor` 的 `xdg-drift` 项比较 `.initialized` 记录的 ConfigDir 与当前 `ConfigDir`，不一致即告警。注意——这只在 `.initialized` 随配置一起被复制到新路径时触发；若仅改了 `XDG_CONFIG_HOME` 而没迁移配置，doctor 会在新路径找不到 `.initialized`、报 `initialized` 失败（同样提示漂移）。也可手动 `envonce env list` 确认（返回为空或缺少预期变量 = 漂移）。

**修复**：对每个受管服务重生 wrapper 与 plist（让新路径重新烘焙进去）：

```bash
envonce service sync ollama     # 单个
# 或对所有服务（遍历 config.toml）
for svc in $(envonce service list | awk 'NR>1{print $1}'); do
  envonce service sync "$svc"
done
```

> 根因：plist/wrapper 里的路径在生成时展开为绝对路径（launchd 要求）。`XDG_CONFIG_HOME` 变动后，旧绝对路径失效。改 env 不会触发（env 运行时解析），只有改路径/binary/args/分组才需要 `sync`。

---

## env 解析失败（fail-loud）

`envonce` 解析 env 时遇到任何错误都会**非零退出**并输出详细诊断，不会用半套 env 启动服务。

常见原因（仅这两类会 fail-loud）：

| 原因 | 诊断示例 | 修复 |
|---|---|---|
| KeyChain 引用解析失败（项缺失或被拒绝） | `Error: ollama: KEY=@keychain:ref failed to resolve: exit status 44`（退出码取 `security` 实际返回） | `security add-generic-password -s <ref> -w <value>`；详见 [KeyChain 授权弹窗 · 解析失败的诊断](#keychain-解析失败的诊断) |
| env.d 文件语法错误（非法行） | `Error: env.d:3: invalid line "..." (missing '=')` | 编辑文件修正（行格式必须是 `KEY=VALUE` 或 `#` 注释） |

> **注意：缺失的组文件不是错误**。若 `config.toml` 引用了组 `work` 但 `env.d/work.env` 不存在，该组会被**静默忽略**（视为空组），不报错。若某变量没生效，请检查服务声明的组是否有对应的 `env.d/<group>.env` 文件。

**手动复现解析**（不走 launchd，直接看输出）：

```bash
envonce env export --service ollama      # 看该服务的完整 env 解析
envonce env export --groups default,work # 看指定组合并
envonce shell-init                        # 看 shell 侧解析
```

这些命令解析失败时会非零退出、诊断到 stderr，是排查「服务起不来」的利器。

---

## i18n 双语支持

`envonce` 已实现**完整中英双语**（`--lang`/`ENVONCE_LANG`/`LANG` 自动检测）。下列内容全部随语言切换：

- **帮助文本**（`envonce --help`、各子命令 `Short`/`Long`、各 flag 说明）
- **操作摘要**（`✓ 做了什么` / `下一步` / `回滚`）
- **错误信息**（如 `envonce --lang en service start nope` → `Unknown service: nope`）
- **状态标签**（`service status`/`list` 的 `Running`/`Not loaded`）
- **`doctor` 检查项描述**、**首运行 banner**

**实现要点**：cobra 的 `Short`/`Long` 在命令树构造期固化、早于语言检测，且 `--help` 会短路 `PersistentPreRunE`（语言检测不运行）。`envonce` 用自定义 `HelpFunc` 在 flag 解析后重新设语言、按 annotation 重译命令树文案（`applyI18n`），从而 `--help` 与正常执行两条路径都正确切换。命令名与 flag 名始终英文/ASCII，可脚本化。

**内部库错误**：`envfile`/`resolver` 等内部库的错误信息保持英文（Go 库惯例），不经 i18n 目录——这些是边缘场景诊断（如 `env.d` 文件语法错误），正常使用不触发。

---

## 其它常见问题

### `envonce` 报「尚未初始化」

任何子命令执行时若检测到 `.initialized` 缺失，会先打印醒目提示（随 `--lang` 切换语言）：

```
⚠ envonce 尚未初始化 — 运行 `envonce init` 完成 shell 集成与目录创建
```

这是兜底提示（不依赖安装方式）。运行 `envonce init` 即可。

### 改了 config.toml 但服务没变

`config.toml` 里改 binary 路径 / args / 服务的额外组 → 需要 `envonce service sync <name>` 重生 wrapper 与 plist（因为这些都烘焙在 wrapper 里）。改 **env 值** 不需要 sync（env 运行时解析），只需 `service restart`。

### 想彻底卸载

```bash
envonce service drop <name>          # 逐个卸载受管服务
envonce init --uninstall             # 移除 shell 集成行（保留配置与数据）
# 若想连配置一起删：
rm -rf ~/.config/envonce
rm ~/Library/LaunchAgents/com.envonce.*.plist
```

`service drop <name> --restore-brew` 会把服务交还 brew 管理（重新 `brew services start`）。
