# envonce

[![CI](https://github.com/laidbackgeek/envonce/actions/workflows/ci.yml/badge.svg)](https://github.com/laidbackgeek/envonce/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[English](README.en.md) | **简体中文**

`envonce` 是一个 macOS 命令行工具，用来**统一管理 shell 与 launchd 后台服务的环境变量**。它解决两个痛点：

1. 交互式 shell 与多个后台服务各自维护自己的环境变量难以一致；
2. 手动改由 Homebrew 维护的 plist，会在 `brew upgrade` 时被覆盖。

## 功能特性

- 🎯 **一处配置，多处生效** —— 同一份 env 同时驱动交互式 shell 与 `launchd` 后台服务
- 🍺 **接管 brew 服务** —— `envonce` 维护 plist+wrapper，`brew upgrade` 只更新二进制、不再覆盖你的环境变量
- 🔐 **KeyChain 集成** —— 敏感值用 `@keychain:` 引用，密钥不落盘
- 🧰 **手动服务也支持** —— `service add` 定义非 brew 的自定义后台服务
- 📦 **多组 env** —— 按 group 管理不同项目/服务的变量集合
- 🩺 **fail-loud 诊断** —— `doctor` 自检、状态标签、错误信息全部可读且本地化
- 🌐 **中英双语**界面

## 起源故事

我开发这个命令行工具的动机是，当我用 Homebrew 启动 ollama 时，模型文件被下载到了我本地磁盘，而没有使用我在 shell 中测试时下载到外接硬盘的模型文件。原因是我在`~/.zprofile`中配置的`OLLAMA_MODELS`，对 Homebrew 启动 ollama 无效。这个问题也同样存在于我自己开发的项目中。

我考虑过几种解决方案，比如直接修改ollama 的 plist 来添加环境变量，但这样做当`brew upgrade ollama` 时，这个 plist 会被覆盖，导致配置丢失。又比如添加一个 plist 专门用来添加全局环境变量，但又会存在竞态问题。

我希望找到一种方法，可以只在一个地方配置一次环境变量，然后无论是在 shell 还是在由 launchd 启动的进程中，都能生效。因此我针对我的使用场景，即在 Mac 上进行开发测试时，多个项目既会在 shell 中进行开发调试，又会通过 launchd 启动为后台服务，在这样的场景下开发了这个命令行工具。

## 前提条件

- **macOS** —— `envonce` 依赖 `launchd` / `launchctl`，仅支持 macOS（不支持 Linux / Windows）。
- **Homebrew**（可选）—— `service take` 接管 brew 服务时需要；仅用 shell 集成则不必安装。
- **Go 1.21+**（可选）—— 仅当用 `go install` 安装时需要。

## 安装

两种方式任选其一：

**Homebrew tap（推荐）**

```bash
brew tap laidbackgeek/homebrew-tap
brew install envonce
```

> 新版 Homebrew 首次安装第三方 tap 会要求信任：若 `brew install envonce` 被拒绝并提示 `brew trust`，执行 `brew trust laidbackgeek/tap` 后重试即可。

安装完成后，按说明提示执行 `envonce init`（见下文）。

**go install（需 Go 1.21+ 工具链，适合 Go 开发者）**

```bash
go install github.com/laidbackgeek/envonce/cmd/envonce@latest
```

`go install` 会把二进制装到 `$GOPATH/bin`（默认 `~/go/bin`）。**该目录必须在 `$PATH` 中**，否则 shell 提示找不到 `envonce` 命令。

## 快速上手（三步）

> 想让终端也使用统一 env？运行 `envonce init` —— 它会自动把 `eval "$(envonce shell-init)"` 接进 `~/.zshrc`（或 `~/.bashrc`），新开终端即生效。

```bash
# 1. 初始化：建目录骨架 + 接入 shell（改 ~/.zshrc，幂等）
envonce init

# 2. 配置环境变量（写入 ~/.config/envonce/env.d/default.env）
envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models

# 3. 接管 brew 服务（导入 + 迁移 plist 环境变量 + 停 brew + 启动 envonce 管理）
envonce service take ollama
```

完成后：新开终端会自动加载统一 env；ollama 服务下次 restart 即用上自定义模型目录，`brew upgrade ollama` 也不再覆盖。

> 详细流程（含 launchctl 验证、KeyChain 敏感值、多组 env）见 [docs/usage.md](docs/usage.md)。

## 命令速查表

| 命令 | 作用 |
|---|---|
| `envonce init [--uninstall]` | 首次安装：建目录骨架、检测 `$SHELL`、往 rc 文件追加带 marker 的 `eval` 行（幂等、可卸载） |
| `envonce shell-init` | 产出 shell `export` 行（运行时解析，含 KeyChain）。通常由 `init` 间接调用；手动用于预览/排查/非标准 shell 接入 |
| `envonce env set KEY=VALUE [--group G]` | 写入 env.d（默认 default 组） |
| `envonce env get KEY [--group G]` | 读取 |
| `envonce env unset KEY [--group G]` | 删除 |
| `envonce env list [--group G]` | 列出 |
| `envonce env export [--service X \| --groups g1,g2]` | 产出 export 行（shell-init 与 wrapper 共用的解析内核） |
| `envonce group create/list/rename/delete` | 组管理 |
| `envonce service take NAME` | 从 `brew info --json=v2` 导入、`brew services stop`、生成 plist+wrapper、bootstrap |
| `envonce service add NAME [-- BINARY_ARGS...]` | 手动定义非 brew 服务（`--binary`、`--keep-alive`、`--run-at-load`） |
| `envonce service drop NAME [--restore-brew]` | 卸载 plist+wrapper；`--restore-brew` 交还 brew 管理 |
| `envonce service start/stop/restart NAME` | 生命周期（restart 让改过的 env 立即对服务生效） |
| `envonce service status NAME` | 运行状态 + env 解析健康检查 |
| `envonce service sync NAME` | 按 config.toml 重生 plist+wrapper 并 reload（binary 路径/args/分组变更才需要） |
| `envonce service list` | 受管服务清单 + 状态 |
| `envonce doctor` | 自检：初始化、brew 可达、security、brew 残留 plist 冲突、XDG 漂移 |
| `envonce --version` / `-v` | 打印版本号 |
| `envonce completion zsh\|bash\|fish\|powershell` | 生成 shell 补全脚本（cobra 自动提供） |

每条命令都可用 `--help` 查看用法，例如 `envonce service take --help`。

## 界面语言（i18n）

`envonce` 支持中英双语。语言检测优先级：

1. `--lang zh\|en` （显式覆盖，最高）
2. `ENVONCE_LANG` 环境变量
3. `LC_ALL` → `LC_MESSAGES` → `LANG`（取 `.` 前语言部分，`zh*` → 中文，否则英文）
4. 兜底英文

```bash
envonce --lang zh init      # 强制中文摘要
ENVONCE_LANG=en envonce init  # 通过环境变量强制英文
```

### 本地化范围

`envonce` 已实现**完整中英双语**——下列内容全部随 `--lang`/系统 LANG 切换：

- **帮助文本**（`envonce --help`、各子命令 `Short`/`Long`、各 flag 说明）
- **操作摘要**（`✓ 做了什么` / `下一步` / `回滚` 结构化输出）
- **错误信息**（如 `Unknown service: ...` / `未知服务 ...`）
- **状态标签**（`service status`/`list` 的 Running / 运行中、Not loaded / 未加载）
- **`doctor` 检查项描述**、**首运行 banner**

## Shell 补全（completion）

`envonce` 基于 cobra，内置 `completion` 子命令生成补全脚本（补全命令名、服务名、组名）。

**zsh**

```bash
# 1. 创建补全目录（只需一次）
mkdir -p ~/.zsh/completion
# 2. 生成补全脚本
envonce completion zsh > ~/.zsh/completion/_envonce
# 3. 把目录加入 fpath（在 ~/.zshrc 里，需在 compinit 之前）
echo 'fpath+=~/.zsh/completion' >> ~/.zshrc
# 4. 重新加载
exec zsh
```

**bash**

```bash
mkdir -p ~/.bash_completion.d
envonce completion bash > ~/.bash_completion.d/envonce
echo 'source ~/.bash_completion.d/envonce' >> ~/.bashrc
exec bash
```

**fish** / **powershell**：运行 `envonce completion <shell>` 并按输出头部提示安装。

## 更多文档

- [用户指南（docs/usage.md）](docs/usage.md) —— 接管 brew 服务完整流程、shell 集成、KeyChain 设置、多组 env
- [故障排查（docs/troubleshooting.md）](docs/troubleshooting.md) —— fail-loud 诊断、doctor 解读、常见冲突

## AI 助手 Skill

仓库内 [`skills/envonce/`](skills/envonce/SKILL.md) 提供了一个 [Agent Skill](https://code.claude.com/docs/en/skills)，让 Claude Code 等 AI 编码助手正确使用 envonce——尤其是避免误用 `brew services` 而让 envonce 维护的环境变量失效。

全局安装（之后在任意目录都生效）——在仓库根目录下执行，把该目录软链到 Claude Code 的 skills 目录：

```bash
mkdir -p ~/.claude/skills
ln -s "$(pwd)/skills/envonce" ~/.claude/skills/envonce
```

不想用软链，也可以直接拷贝：

```bash
cp -r skills/envonce ~/.claude/skills/envonce
```

之后新开一个 Claude Code 会话即可加载该 skill。

## 许可

MIT
