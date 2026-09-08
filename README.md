[English](README.en.md) | **简体中文**

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/hero-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/hero-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/hero-dark.svg">
  <img src="assets/presentation/hero-light.svg" width="1000" alt="将模型规格快照与已配置字段对照，区分直接相关与仅供参考的变化。">
</picture>

**将模型规格快照与已配置字段对照，区分直接相关与仅供参考的变化。**

`v0.1.0` · `Go 1.24+` · [MIT](LICENSE)

[Website](https://dropdiff.lei6393.com) · [Demo record](docs/demo-results.json)

## 为什么使用

模型规格更新后，context、缓存模式或工具调用约定是否与你有关，取决于当前接入了什么。DropDiff 将 WiredConfig 和策展注册表做字段比较，保留 Before、After、Affects 和严重度，帮助定位需要进一步核对的项。

## 架构

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/architecture-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/architecture-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/architecture-dark.svg">
  <img src="assets/presentation/architecture-light.svg" width="1000" alt="config 解析或读取模型配置，spec 加载 YAML DropSpec，differ 先寻找明确 successor，没有时可按同 vendor 的较新记录回退。render 输出相关或 informational 标签；check/list 可离线，sync 才拉取远程 registry。">
</picture>

config 解析或读取模型配置，spec 加载 YAML DropSpec，differ 先寻找明确 successor，没有时可按同 vendor 的较新记录回退。render 输出相关或 informational 标签；check/list 可离线，sync 才拉取远程 registry。

当前本地示例显式加载 [data/registry.yaml](data/registry.yaml)，不读取用户的 ~/.dropdiff。注册表是仓库内容快照，里面的价格和来源引用不作为本次核验过的最新供应商事实。

## 安装

需要 Go 1.24+。演示使用完整的显式配置与本地 registry，不需要模型服务或用户配置。

```bash
git clone https://github.com/SuperMarioYL/dropdiff.git
cd dropdiff
go build -o dropdiff .
```

## 快速开始

实际调用核心 differ，分别以 implicit 和 off 的缓存模式读取相同快照。输出只证明字段相关性规则的执行，没有查询最新模型定价、SDK commit 或平台配置。

```bash
go run ./examples/presentation implicit
go run ./examples/presentation off
```

完整 fixture 在 [examples/presentation/main.go](examples/presentation/main.go)，数据来自随仓库提供的 registry YAML。

## 使用

init 可扫描常见本地配置并写入 ~/.dropdiff/wired.yaml；--config 可指定手动快照。check 比较，list 列表，sync 更新本地 registry。自动发现按来源优先级补齐字段，不会自动修改 Agent 的真实配置。

## 实际 Demo

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/process-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/process-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/process-dark.svg">
  <img src="assets/presentation/process-light.svg" width="1000" alt="实际调用核心 differ，分别以 implicit 和 off 的缓存模式读取相同快照。输出只证明字段相关性规则的执行，没有查询最新模型定价、SDK commit 或平台配置。">
</picture>

### 缓存启用

implicit 模式将缓存价格项标记为相关。

```text
$ go run ./examples/presentation implicit
Local registry snapshot: 2026-08-09; fixture cache_mode=implicit

glm-5.3  2026-07-29  [GLM]
  ✓ pricing.cache: 0.28 CNY/M -> 0.14 CNY/M
      affects your wired config — implicit cache price
  ✓ quants: fp16,int8 -> fp16,int8,fp8
      informational — quants removed=[] added=[fp8]
2 delta(s): 0 red, 2 green, 0 amber
```

### 缓存关闭

off 模式下同一字段作为 informational 显示。

```text
$ go run ./examples/presentation off
Local registry snapshot: 2026-08-09; fixture cache_mode=off

glm-5.3  2026-07-29  [GLM]
  ✓ pricing.cache: 0.28 CNY/M -> 0.14 CNY/M
      informational — implicit cache price
  ✓ quants: fp16,int8 -> fp16,int8,fp8
      informational — quants removed=[] added=[fp8]
2 delta(s): 0 red, 2 green, 0 amber
```

## 能力与接入

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/integrations-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/integrations-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/integrations-dark.svg">
  <img src="assets/presentation/integrations-light.svg" width="1000" alt="比较依赖注册表维护质量。明确 successor 优于同 vendor 的日期回退；价格只在单位相同时比较。报告用于筛选待核对变化，不替代供应商来源核验。">
</picture>

比较依赖注册表维护质量。明确 successor 优于同 vendor 的日期回退；价格只在单位相同时比较。报告用于筛选待核对变化，不替代供应商来源核验。



## 配置

WiredConfig 包含 model、base_url、context、cache_mode、source。check 优先读取 ~/.dropdiff/registry.yaml，否则回退 bundled。DROPDIFF_REGISTRY_URL 控制 sync 来源。缓存模式为空/off/none/false 时，缓存价格差异不标为 Affects；quant 差异保持 informational。

## 路线图与范围

当前提供配置解析、快照 diff 与 registry 同步。自动追踪真实 SDK 变更、多 provider 基线和定期通知仍为后续方向；没有自动升级接入配置。

- 本次未核验 bundled registry 的价格或来源引用，不能当作实时供应商信息。
- 同 vendor 回退匹配可能包含并非直接升级关系的模型。
- 工具只报告差异，不修改 Agent 配置或执行升级。

[Terminal recording](assets/demo.gif) · [Recording script](docs/demo.tape)

## 许可证

[MIT](LICENSE)
