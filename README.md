<div align="right"><sub>[English](./README.en.md) &nbsp;⇄&nbsp; <b>简体中文</b></sub></div>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./assets/hero-light.svg">
  <img src="./assets/hero-light.svg" width="880" alt="DropDiff — Spec-diff CLI for the Chinese model carousel">
</picture>

<p align="center"><sub>国产模型 carousel 的 Renovate —— 一行命令，只 diff 影响你 Agent 配置的 drop</sub></p>

<p align="center"><strong>每周 DeepSeek / Qwen / Kimi / GLM 的 drop，你不必再翻五个 tab —— DropDiff 只标出影响你已 wired 配置的那一条。</strong></p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/SuperMarioYL/dropdiff?color=blue" alt="license"></a>
  <a href="https://github.com/SuperMarioYL/dropdiff/releases"><img src="https://img.shields.io/github/v/release/SuperMarioYL/dropdiff" alt="release"></a>
  <img src="https://img.shields.io/github/actions/workflow/status/SuperMarioYL/dropdiff/ci.yml?branch=main&label=ci" alt="ci">
  <img src="https://img.shields.io/badge/Go-1.24-0071E3?logo=go" alt="go">
  <img src="https://img.shields.io/badge/anchor-Agent-8985FF" alt="Agent">
  <img src="https://img.shields.io/badge/anchor-Skill-5E5CE6" alt="Skill">
</p>

<details>
<summary>目录</summary>

- [架构](#架构)
- [为什么做这个](#为什么做这个)
- [安装](#安装)
- [快速开始](#快速开始)
- [用法](#用法)
- [Demo](#demo)
- [对比](#对比)
- [配置](#配置)
- [路线图](#路线图)
- [License](#license)
</details>

<h2><img src="https://api.iconify.design/tabler:topology-star-3.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 架构</h2>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/atlas-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./assets/atlas-light.svg">
  <img src="./assets/atlas-light.svg" width="880" alt="架构：Wired Config + Drop Registry → Differ → 红/绿 Config Deltas">
</picture>

单进程、单二进制、三个命令。`init` 把你已 wire 的配置快照成 baseline；`sync` 拉取策展的 drop-spec registry；`check` 用 `DropSpec × WiredConfig → ConfigDelta` 这组三元做 diff，只打印红/绿 delta。唯一的网络调用是 `sync`，`check` 默认用嵌入二进制的 bundled registry，零网络也能跑。

<h2><img src="https://api.iconify.design/tabler:bolt.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 为什么做这个</h2>

国产模型的 drop 不像 npm 包那样有版本号 —— 它们散落在 ModelScope 卡片、国产 SDK 的 commit log 和 Reddit 的 benchmark drama 里。把一个新 drop（定价 / 上下文 / 缓存价 / quant / tool-call 怪癖）和你**当前 wired 的配置**对上，每周要手动扒五个 tab，约 2 小时。

DropDiff 把这件事压成一行命令。它的护城河是那个 `Affects` 布尔：**「这条 delta 是不是动到了你真正 wired 的字段？」** ModelScope 卡片是模型作者写的，它不知道你 wire 的是 `glm-4.6` 还是 `deepseek-v4` —— 卡片能把「Qwen3.8-Max 输入 14.0 ¥/M」告诉你，却没法告诉你「这比你 wire 的 Qwen3-VL 贵了 28 倍」。

这恰好是 [langgenius/dify](https://github.com/langgenius/dify) 这类 Agent 平台用户每周都在做的 chore：在 Dify 里 wire 一个国产模型 endpoint，然后每周判断要不要换到这周更好的 drop。DropDiff 就是那个判断的那一行命令 —— Dify 负责跑 Agent，DropDiff 负责告诉你这周的 drop 该不该换。`Agent` 是这整个流程的消费端锚点。

<h2><img src="https://api.iconify.design/tabler:rocket.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 安装</h2>

**Go（推荐）：**

```bash
go install github.com/SuperMarioYL/dropdiff@latest
```

**从源码：**

```bash
git clone https://github.com/SuperMarioYL/dropdiff
cd dropdiff && go build -o dropdiff . && sudo mv dropdiff /usr/local/bin/
```

**预编译二进制：** 见 [Releases](https://github.com/SuperMarioYL/dropdiff/releases)（首个 tag 推送后，GoReleaser 会产出 linux/darwin/windows × amd64/arm64 + sha256）。

<h2><img src="https://api.iconify.design/tabler:terminal-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 快速开始</h2>

从冷启动到首个可见结果，两条命令：

```bash
dropdiff init     # 检测 ~/.claude.json / ~/.aider/conf.yml / .env，快照到 ~/.dropdiff/wired.yaml
dropdiff check    # diff 本周国产 drop，只打印影响你配置的红/绿 delta
```

<details>
<summary>示例输出（GLM 5.3 cache-price 抓取）</summary>

```
wired: ~/.dropdiff/wired.yaml  (model=glm-4.6, base_url=https://open.bigmodel.cn)
registry: bundled  (10 drops, updated 2026-08-09)

glm-5.3  2026-07-29  [GLM]
  ✓ pricing.cache: 0.28 CNY/M -> 0.14 CNY/M
      affects your wired config — implicit cache price
  ✓ quants: fp16,int8 -> fp16,int8,fp8
      informational — quants removed=[] added=[fp8]
2 delta(s): 0 red, 2 green, 0 amber
```

</details>

<h2><img src="https://api.iconify.design/tabler:terminal-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 用法</h2>

```bash
# 从 .env wire 一个国产模型 endpoint（带 cache + context，让 delta 标记为 Affects）
printf 'MODEL=glm-4.6\nOPENAI_API_BASE=https://open.bigmodel.cn\nCACHE_MODE=implicit\nMAX_CONTEXT=128000\n' > .env
dropdiff init --cwd .

# 拉取最新的 drop-spec registry（默认从 hosted raw GitHub；$DROPDIFF_REGISTRY_URL 可覆盖）
dropdiff sync

# 列出 registry 里的所有 drop（含 ModelScope card + 国产 SDK commit 来源）
dropdiff list

# 用一个手动指定的 wired 配置做 check
dropdiff --config /path/to/wired.yaml check

# 看版本
dropdiff --version
```

`init` 会按 `~/.claude.json → ~/.aider/conf.yml → .env` 的顺序探测，先找到的字段胜出，后续来源只补空缺 —— 所以 `.env` 里写 `BASE_URL`、`claude.json` 里写 `model` 也能合并成一份完整快照。

<h2><img src="https://api.iconify.design/tabler:photo.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Demo</h2>

![demo](assets/demo.gif)

asciinema 录制（GLM 5.3 cache-price 在 ModelScope 卡片更新前被抓到）：[assets/demo.cast](./assets/demo.cast)。GIF 由 [`docs/demo.tape`](./docs/demo.tape) 经 vhs 在 CI 渲染。

<h2><img src="https://api.iconify.design/tabler:scale.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 对比</h2>

DropDiff 是 diff 工具，不是 Agent 平台，也不是 awesome 目录。和它放在一起看的：

| 维度 | DropDiff | [langgenius/dify](https://github.com/langgenius/dify) | ModelScope 卡片 |
|---|:---:|:---:|:---:|
| 知道你 wired 什么 | ✓ | ✓（在 Dify 里 wire） | ✗ |
| per-drop 红绿 delta | ✓ | ✗ | ✗ |
| 国产 SDK commit 优先 | ✓ | ✗ | ✗ |
| 跑 Agent / 编排 | ✗ | ✓ | ✗ |
| 模型作者维护的 spec | ✗ | ✗ | ✓ |

DropDiff 不替 Dify 编排 Agent，也不做 awesome-claude-skills 那种 **Skill** 目录 —— 它只 diff，且只标影响你配置的那一条。（诚实地说：Dify 在「跑 Agent / 编排」这一行比 DropDiff 强得多 —— 那本来就不是 DropDiff 的活。）

<h2><img src="https://api.iconify.design/tabler:adjustments.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 配置</h2>

`~/.dropdiff/wired.yaml` —— 由 `dropdiff init` 从你的 claude.json / aider / .env 快照而来：

| key | 类型 | 默认 | 含义 |
|---|---|---|---|
| `model` | string | — | 你 wired 的模型 id，如 `glm-4.6`、`deepseek-v4` |
| `base_url` | string | — | 你 wired 的 endpoint URL |
| `context` | string | — | wired 的 max context（token） |
| `cache_mode` | string | — | `implicit` / `off` 等；决定缓存价 delta 是否 `Affects` |
| `source` | string | — | 快照来源文件（逗号分隔） |

`check` 优先读 `~/.dropdiff/registry.yaml`（`sync` 写入），没有就回退到嵌入二进制的 bundled registry。`$DROPDIFF_REGISTRY_URL` 可覆盖 sync 的来源。

<h2><img src="https://api.iconify.design/tabler:map-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> 路线图</h2>

- [x] **m1 wired config** — `dropdiff init` 解析 claude/aider/.env → `wired.yaml`
- [x] **m2 spec registry** — 策展 `registry.yaml`（8 周国产 drop）+ `dropdiff sync` 拉取 hosted JSON
- [x] **m3 diff check** — `dropdiff check` 红绿 config-affecting delta + `assets/demo.cast`
- [ ] v0.2 — live 国产 SDK commit 抓取（取代手工策展，z-ai-sdk-java 等）
- [ ] v0.3 — 多 provider routing baseline（self-hosted Qwen 等 OpenRouter 看不到的）
- [ ] v0.4 — 每周 diff 邮件 / webhook（OSS-first，单配置免费）

<h2><img src="https://api.iconify.design/tabler:license.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> License</h2>

MIT —— 见 [LICENSE](./LICENSE)。欢迎在 [Issues](https://github.com/SuperMarioYL/dropdiff/issues) 提问题或 PR。

## Share this

```
DropDiff — 给国产模型 carousel 做的 Renovate。一行命令把每周 DeepSeek/Qwen/Kimi/GLM 的定价/缓存/quant 变更 diff 成只影响你 Agent 的红绿标记。 https://github.com/SuperMarioYL/dropdiff
```

<p align="center"><sub><a href="./LICENSE">MIT</a> © 2026 SuperMarioYL</sub></p>
