<h1 align="center">agy-statusline · Antigravity CLI 状态栏</h1>

<p align="center">
  基于 Go 编写的 Antigravity CLI（<code>agy</code>）自定义双行状态栏：<br>
  零外部依赖、Catppuccin / Tokyo-Night 风格配色、平滑渐变上下文进度条与响应式布局。
</p>

<p align="center">
  <a href="https://github.com/jiazixin/agy-statusline/actions/workflows/ci.yml"><img src="https://github.com/jiazixin/agy-statusline/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="https://github.com/jiazixin/agy-statusline/releases/latest"><img src="https://img.shields.io/github/v/release/jiazixin/agy-statusline?sort=semver" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-4c8b3f" alt="Linux | macOS | Windows">
</p>

<p align="center">
  <b>简体中文</b> · <a href="README.en.md">English</a>
</p>

---

`agy-statusline` 从标准输入读取 Antigravity CLI 传入的 JSON 状态负载，并向标准输出渲染对齐的彩色双行状态栏。它仅依赖 Go 标准库，编译为单一静态可执行文件，启动开销极低，支持 Linux、macOS 和 Windows 多种架构。

```text
󰙴 Gemini 4 Neon (high) │ ━━──────── 21% 42k/200k │ cache 84%                             ◉ working  » auto
󰆥 Ultra                │ 󰝰 agy-statusline   main* │ agents:2 tasks:3     5h 85%·3h35m  wk 72%·4d4h  ⚡120
```

> 终端中安装并启用 Nerd Font 时，模型、订阅计划、目录与 Git 分支图标会正常显示。

## 功能特性

- **双行对齐与响应式布局**：按 `terminal_width` 自动左右分散对齐，并对齐上下两行首列；终端宽度小于 80 列时自动切换紧凑模式（缩短进度条、精简文案并隐藏次要字段）。内置零分配 ANSI 跳过与 CJK / Emoji 宽字符计算。
- **TrueColor 与 256 色自适应**：根据 `COLORTERM`（`truecolor` / `24bit`）及常见现代终端的 `TERM_PROGRAM` 自动启用 24 位真彩色与逐格渐变进度条，其余环境自动回退到 256 色。
- **上下文与缓存监控**：显示渐变上下文用量进度条（常规青蓝渐变、≥60% 橙黄警示、≥80% 红橙高亮）、使用百分比、已用/总 token 比例，以及当前请求的 Prompt Cache 命中率。
- **工作区与 Git 状态**：展示当前工作目录名、分支名及脏工作区标记（`*`）；当负载未提供 VCS 字段时，会自动在 500ms 超时内调用 `git --no-optional-locks status --porcelain=v2 --branch` 获取状态。
- **配额与积分展示**：支持 `quota.windows`、`quota` 分桶与 `rate_limits` 多种格式，按当前模型系列自动筛选对应配额并附带重置倒计时，同时显示 AI Credits（`⚡`）。
- **状态与扩展计数**：展示 Agent 运行状态（`working` / `thinking` / `tool_use` / `running` / `idle`）、协作模式（`auto` / `accept-edits` / `plan` / `default`），以及活跃子代理、任务数、产物数和 Sandbox 标记。
- **零依赖单文件**：纯 Go 标准库实现，无第三方依赖；遇到非法 JSON 输入时回退输出 `✦ Antigravity`，不中断 CLI 界面。

## 运行要求

| 项目 | 要求 |
| --- | --- |
| 操作系统 | Linux、macOS 或 Windows |
| Antigravity CLI | 已安装 `agy`，并在 `settings.json` 中配置 `statusLine` |
| 字体（推荐） | 任意 Nerd Font（用于显示模型、计划、目录与 Git 图标） |
| 终端颜色（推荐） | 支持 24-bit TrueColor 的终端（不支持时自动回退 256 色） |
| Git（可选） | 当状态负载未提供 `vcs` 字段时用于回退探测当前分支与脏状态 |
| 从源码构建 | Go 1.27 或更高版本（使用了标准库 `encoding/json/v2`） |

## 快速开始

### 1. 下载预编译版本

在 [Releases](https://github.com/jiazixin/agy-statusline/releases/latest) 下载对应平台与架构的压缩包：

- **Linux / macOS**（`.tar.gz`）：

```sh
ver=1.0.0   # 替换为 Releases 页面上的实际版本（不带 v）
os=linux    # 或 darwin
arch=amd64  # 或 arm64 等目标架构

curl -LO "https://github.com/jiazixin/agy-statusline/releases/download/v${ver}/agy-statusline_${ver}_${os}_${arch}.tar.gz"
curl -LO "https://github.com/jiazixin/agy-statusline/releases/download/v${ver}/checksums.txt"
sha256sum --ignore-missing -c checksums.txt   # macOS 可用: shasum -a 256 --ignore-missing -c checksums.txt

mkdir -p ~/.local/bin
tar -xzf "agy-statusline_${ver}_${os}_${arch}.tar.gz" -C ~/.local/bin agy-statusline
chmod +x ~/.local/bin/agy-statusline
```

> macOS 预编译二进制未经公证。通过浏览器下载后如被 Gatekeeper 拦截，可执行 `xattr -d com.apple.quarantine ~/.local/bin/agy-statusline`。

- **Windows**（`.zip`）：

在 Releases 页面下载 `agy-statusline_<版本>_windows_<架构>.zip`（支持 `amd64`、`arm64`、`386`），解压出 `agy-statusline.exe` 并放到任意目录（例如 `%USERPROFILE%\.local\bin\agy-statusline.exe` 或加入 `PATH`）。

### 2. 从源码构建

```sh
git clone https://github.com/jiazixin/agy-statusline.git
cd agy-statusline
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ~/.local/bin/agy-statusline .
```

### 3. 配置 Antigravity CLI

编辑 `~/.gemini/antigravity-cli/settings.json`，添加或更新 `statusLine` 字段：

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.local/bin/agy-statusline"
  }
}
```

Windows 用户填写实际的 `.exe` 路径（若已加入 `PATH` 也可直接写 `"agy-statusline.exe"`）。保存后启动或刷新 `agy` 即可生效。

## 界面说明与环境变量

### 显示内容

| 位置 | 字段说明 |
| --- | --- |
| 第一行左侧 | 模型名称与推理强度（`effort`）、上下文渐变进度条与用量、Prompt Cache 命中率 |
| 第一行右侧 | Agent 当前状态（非紧凑模式）、协作模式（`auto` / `accept-edits` / `plan` / `default`） |
| 第二行左侧 | 订阅计划（`plan_tier`，非紧凑模式）、工作目录与 Git 分支/脏状态、子代理 / 任务 / 产物 / Sandbox 计数 |
| 第二行右侧 | 按模型系列匹配的配额剩余百分比与重置倒计时、AI Credits（`⚡`） |

### 环境变量

| 变量 | 说明 |
| --- | --- |
| `COLORTERM` | 包含 `truecolor` 或 `24bit` 时启用 24 位真彩色输出 |
| `TERM_PROGRAM` | 识别 iTerm2、Ghostty、WezTerm、Warp、VS Code、Kitty、Alacritty、Hyper、Rio、Tabby 并自动启用真彩色 |
| `AGY_STATUS_DEBUG` | 设为 `1` 时将原始标准输入 JSON 写入 `/tmp/agy-statusline-debug.json`，便于排查字段格式 |

## 支持的发布平台

每次推送 `v*` 标签后，GitHub Actions 会自动交叉编译以下全部目标并随 `checksums.txt` 一同发布到 GitHub Releases：

| 操作系统 | 架构 | 归档格式 |
| --- | --- | --- |
| Linux | `amd64`、`arm64`、`386`、`armv6`、`armv7`、`loong64`、`mips`、`mipsle`、`mips64`、`mips64le`、`ppc64`、`ppc64le`、`riscv64`、`s390x` | `.tar.gz` |
| macOS (`darwin`) | `amd64`、`arm64` | `.tar.gz` |
| Windows | `amd64`、`arm64`、`386` | `.zip` |

> Linux 除 `amd64` / `arm64` 之外的架构及 Windows 目标由 GitHub Actions 交叉编译生成，如在特定设备上遇到显示或兼容问题，欢迎提交 Issue。

## 本地调试与开发

直接通过管道喂入测试 JSON 即可在终端预览渲染效果：

```sh
printf '%s\n' '{
  "terminal_width": 110,
  "model": {"id": "gemini-4-neon-high", "display_name": "Gemini 4 Neon", "effort": "high"},
  "context_window": {
    "used_tokens": 42000,
    "context_window_size": 200000,
    "current_usage": {"input_tokens": 12000, "cache_read_input_tokens": 84000, "cache_creation_input_tokens": 4000}
  },
  "agent_state": "working",
  "cycle_mode": "auto",
  "plan_tier": "Ultra",
  "workspace": {"current_dir": "/path/to/agy-statusline"},
  "vcs": {"type": "git", "branch": "main", "dirty": true},
  "quota": {"gemini-5h": {"remaining_fraction": 0.85, "reset_in_seconds": 12900}},
  "ai_credits": 120
}' | go run .
```

运行本地静态检查与竞态测试：

```sh
gofmt -l .
go vet ./...
CGO_ENABLED=1 go test -race -count=1 ./...
```

推送到 `main` 和发起 Pull Request 时，[CI 工作流](.github/workflows/ci.yml) 会自动执行格式检查、全目标 `go vet`、竞态测试以及 GoReleaser 快照构建检查。

## 版本与发布

推送形如 `v1.0.0` 或 `v1.0.0-rc.1` 的 `v*` Git 标签即可触发 [Release 工作流](.github/workflows/release.yml)：

```sh
git tag v1.0.0
git push origin v1.0.0
```

工作流会先运行 `go vet` 和 `go test`，再通过 [`.goreleaser.yaml`](.goreleaser.yaml) 编译 Linux / macOS / Windows 全架构产物，生成 SHA-256 校验文件 `checksums.txt` 并自动创建 GitHub Release（带预发布后缀的标签自动标记为 Pre-release）。

## 项目结构

```text
main.go                  状态栏解析、配色渲染与 Git 状态回退逻辑
main_test.go             单元测试（宽字符计算、上下文/配额边界、输入校验）
go.mod                   Go 模块定义（零外部依赖）
.goreleaser.yaml         GoReleaser 多平台多架构打包与发布配置
.github/workflows/       GitHub Actions CI 与 v* 标签自动发布工作流
README.md                简体中文说明
README.en.md             English README
```
