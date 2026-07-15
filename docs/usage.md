# 用户指南

[English](usage.en.md) | **简体中文**

本文档涵盖 `envonce` 的核心使用场景：接管 Homebrew 服务（以 ollama 为例）、shell 集成细节、KeyChain 敏感值管理与多组 env 用法。

> 命令速查见 [README](../README.md#命令速查表)。首次使用请先运行 `envonce init`。

## 目录布局

`envonce` 的所有数据都在 `${XDG_CONFIG_HOME:-$HOME/.config}/envonce/` 下（下文以 `~/.config/envonce/` 指代）：

```
~/.config/envonce/
├── config.toml            # 接线：服务定义、分组分配、shell 组选择
├── env.d/                 # 纯 env 数据，一组一文件
│   ├── default.env        #   全量注入组（shell + 所有受管服务，强制）
│   ├── work.env
│   └── golang.env
├── services/              # 生成产物（受管，勿手改）
│   └── ollama.wrapper.sh  #   包装脚本
├── logs/                  # 操作日志 + 各服务 stdout/stderr
│   ├── envonce.log
│   ├── ollama.out.log
│   └── ollama.err.log
├── state/                 # 运行时状态
└── .initialized           # init 完成标记
```

plist 由 launchd 要求落在 `~/Library/LaunchAgents/com.envonce.<service>.plist`（绝对路径），指向 `services/` 下的 wrapper。

> 若设了 `XDG_CONFIG_HOME`，整体路径随之漂移，plist/wrapper 里会展开为绝对路径。doctor 会检测 XDG 漂移（`xdg-drift` 项比较 `.initialized` 记录路径与当前 ConfigDir）；若仅改了 `XDG_CONFIG_HOME` 未迁移配置，doctor 报 `initialized` 失败同样提示漂移，见 [故障排查](troubleshooting.md#xdg_config_home-漂移)。

---

## 接管 brew 服务：ollama 完整流程

这是 `envonce` 的核心场景：让 brew 只管更新二进制，envonce 负责维护 plist 与环境变量。

### 1. 初始化（如未做过）

```bash
envonce init
```

会创建目录骨架、`env.d/default.env`、默认 `config.toml`，并往 `~/.zshrc` 追加 shell 集成行（marker：`# >>> envonce >>>`）。幂等，可重复运行。

### 2. 配置环境变量

把 `OLLAMA_MODELS` 写进 `default` 组（shell 与所有受管服务都会拿到）：

```bash
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models
```

可放明文非敏感值，也可放 KeyChain 引用（见下文 [KeyChain 设置](#keychain-设置敏感值)）：

```bash
envonce env set GOPATH=$HOME/go
envonce env set GITHUB_TOKEN=@keychain:github-token
```

> `$VAR` 在值里会被消费端 shell/服务进程展开；`envonce` 原样写入。含空格的值请自行加引号。

### 3. 接管服务

```bash
envonce service take ollama
```

这一条命令做了 7 件事：

1. 读取 `~/Library/LaunchAgents/homebrew.mxcl.ollama.plist`（brew 生成的 launchd 定义）→ 解析服务定义（run 命令、keep_alive、run_at_load、日志路径、`EnvironmentVariables`）
2. binary 路径**转成 `opt/` 软链形式**（`/opt/homebrew/opt/ollama/bin/ollama`），升级时自动跟进
3. 写 `[services.ollama]`（`source="brew"`、`args=["serve"]`）进 `config.toml`
4. **迁移** plist 的 `EnvironmentVariables` 到服务同名 group（`env.d/ollama.env`），`services.ollama.groups` 引用之——接管后这些变量由 envonce 统一管理，不再依赖 brew plist（升级也不丢）。用户已在 group 里配置的同名 key 不覆盖
5. `brew services stop ollama`（卸载 brew 的 plist，断开 brew 管理）
6. 生成 `services/ollama.wrapper.sh` + `~/Library/LaunchAgents/com.envonce.ollama.plist`
7. `launchctl bootstrap gui/$UID` 启动

### 4. 验证

**确认服务状态**

```bash
envonce service status ollama
```

输出加载状态 + env 解析健康检查（确认 `OLLAMA_MODELS` 能正确解析）。

**用 launchctl 直接核对**

```bash
launchctl print gui/$(id -u)/com.envonce.ollama
```

可看到 `program` 指向 `~/.config/envonce/services/ollama.wrapper.sh`（不是 brew 的 plist）。

**确认 brew 已不再管理**

```bash
ls ~/Library/LaunchAgents/ | grep ollama
# 应只看到 com.envonce.ollama.plist，不再有 homebrew.mxcl.ollama.plist
```

### 5. 改 env 后生效

改 env 不动 plist 也不动 wrapper（env 运行时解析）：

- **shell**：新开终端即生效（`shell-init` 运行时解析）。
- **服务**：`envonce service restart ollama` 让改动立即生效（无需 `sync`）。

```bash
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models
envonce service restart ollama
```

### 6. brew 升级行为

```bash
brew upgrade ollama
```

升级会更新 Cellar 与 `opt/ollama` 软链 → wrapper 自动用上新版本；plist/wrapper 不受影响；**brew 不会重建 `com.envonce.ollama.plist`**（接管时已 `brew services stop` 移除了 brew 的 plist）。

**`OLLAMA_MODELS` 配置不丢，升级覆盖问题消失。**

> 若误对已接管的服务又跑了 `brew services start ollama`，brew 会重建 `homebrew.mxcl.ollama.plist`，与 `com.envonce.ollama` 端口冲突。`envonce doctor` / `service status` 会检测残留 plist 并警告 —— 见 [故障排查](troubleshooting.md#brew-残留-plist-冲突)。

---

## Shell 集成

`envonce init` 会按 `$SHELL` 往 rc 文件追加带 marker 的集成行：

```sh
# >>> envonce >>>
eval "$(envonce shell-init)"
# <<< envonce <<<
```

- **zsh**：写 `~/.zshrc`
- **bash**：写 `~/.bashrc`

集成行**幂等**：重复 `init` 不会重复追加。`envonce init --uninstall` 会按 marker 精确移除这几行（保留配置与数据）。

### `shell-init` 做了什么

`envonce shell-init` 产出标准 `export KEY=VALUE` 行（运行时解析 `config.toml` 的 `[shell].groups`，默认 `["default"]`）。`eval` 后这些变量进入 shell 进程内存。

手动预览/排查：

```bash
envonce shell-init          # 查看会注入哪些 export
envonce shell-init | sh -x  # 跟踪解析过程
```

### 非 zsh/bash / 非标准 shell 接入

如果你的 shell 不是 zsh/bash（或 rc 文件路径非标准），手动把 `eval "$(envonce shell-init)"` 加到你的启动脚本里即可。`shell-init` 输出的是 POSIX `export`，兼容任何 POSIX shell。

---

## KeyChain 设置（敏感值）

`envonce` 的原则：**秘密永不落盘**。敏感值用 `@keychain:<ref>` 占位符表达，配置文件里只有引用名，真实值只在消费瞬间（shell 启动 / 服务启动）从 macOS KeyChain 解析进进程内存。

### 1. 往 KeyChain 写入值

```bash
# -s 是引用名（service name），-w 是值；-a 是账户名（可选）
security add-generic-password -s github-token -w 'ghp_xxxxxxxxxxxxxxxx'
security add-generic-password -s my-api-key -a envonce -w 'secret-value'
```

### 2. 在 envonce 里引用

```bash
envonce env set GITHUB_TOKEN=@keychain:github-token
envonce env set API_KEY=@keychain:my-api-key
```

`@keychain:<ref>` 在解析时执行 `security find-generic-password -s <ref> -w`，输出 `export KEY='<resolved>'`（单引号包裹，禁止展开）。

### 3. 验证解析

```bash
# 看 shell 会拿到什么（shell 侧默认组）
envonce shell-init | grep GITHUB_TOKEN

# 看某服务的 env 解析结果
envonce env export --service ollama | grep GITHUB_TOKEN

# service status 也会做 env 健康检查
envonce service status ollama
```

### 首次解析的授权弹窗

KeyChain 项首次被 `envonce`（或经由 wrapper 启动的服务进程）读取时，macOS 会弹一次授权框，问是否允许访问。**点「始终允许」**后，系统会写 ACL（access control list），之后该进程再读不再弹窗。

> 若弹窗过于频繁，或希望后台服务静默访问，可在「钥匙串访问」App 里编辑该项的「访问控制」，把 `envonce` 与对应服务二进制加入允许列表。`envonce doctor` 会预检 KeyChain 可达性。

---

## 多组 env 用法

`envonce` 支持**命名的 env 分组**。关键规则：

- **`default` 组是全量注入组**：自动注入**每个受管服务**（强制，不可 opt-out），并且是 shell 默认加载的组。这是 shell↔launchd 统一的黏合点。
- **其它组**按服务选择：一个服务可在 `config.toml` 里叠加一个或多个「额外」组。

### 服务实际 env = default 组 + 该服务的额外组

同 KEY 冲突时**后者覆盖前者**，解析时打一条 warning 到 `logs/envonce.log`。

### 1. 创建组并写入变量

```bash
envonce group create work
envonce env set WORK_API_URL=https://api.work.local --group work
envonce env set DB_PASSWORD=@keychain:work-db-pw --group work
```

### 2. 让某服务叠加额外组

编辑 `config.toml`，给服务加 `groups`：

```toml
[services.ollama]
source     = "brew"
binary     = "/opt/homebrew/opt/ollama/bin/ollama"
args       = ["serve"]
groups     = ["work"]          # default 自动注入；这里只列「额外」组
keep_alive = true
run_at_load = true
```

然后重生 wrapper（分组变了需要 sync）：

```bash
envonce service sync ollama
```

此后 ollama 服务的 env = `default` 组（含 `OLLAMA_MODELS`、`GITHUB_TOKEN` 等）⊕ `work` 组（含 `WORK_API_URL`、`DB_PASSWORD`）。

### 3. shell 侧加载额外组（可选）

shell 默认只加载 `default`。若想终端也拿到某些额外组，改 `config.toml`：

```toml
[shell]
groups = ['default', 'golang']   # shell 显式列出所有要加载的组（含 default）；与服务侧不同，shell 不会自动注入 default
```

新开终端即生效（运行时解析，无需重生）。

> 注意 shell 侧与服务侧的区别：**服务**强制注入 `default`（不可 opt-out）+ 额外组；**shell** 由用户在 `config.toml` 自由选择加载哪些组。

### 组管理命令

```bash
envonce group list                    # 列出全部分组
envonce group create NAME             # 创建空组
envonce group rename OLD NEW          # 重命名
envonce group delete NAME             # 删除组（连同其中的变量）
envonce env list --group golang       # 查看某组内容
envonce env export --groups default,work   # 预览合并后的 export
```

---

## 下一步

- 出问题了？看 [故障排查](troubleshooting.md)。
