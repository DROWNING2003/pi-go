# coding-agent/core 迁移计划

> TS 45 个文件 → Go。按优先级分 4 批。

## 第 1 批：高价值小文件（今天）

| TS | 行 | Go 目标 | 说明 |
|---|---|---|---|
| `model-config.ts` | 287 | `coding-agent/model/config.go` | 模型配置加载 |
| `runtime-credentials.ts` | — | `coding-agent/auth/credentials.go` | 运行时凭据 |
| `resolve-config-value.ts` | 287 | `coding-agent/config/resolve.go` | 配置值解析 |
| `event-bus.ts` | — | `coding-agent/event/bus.go` | 事件总线 |
| `timings.ts` | — | `coding-agent/util/timing.go` | 计时工具 |
| `cache-stats.ts` | — | `coding-agent/util/cache.go` | 缓存统计 |
| `defaults.ts` | — | `coding-agent/config/defaults.go` | 默认配置 |
| `usage-totals.ts` | — | `coding-agent/util/usage.go` | 用量统计 |
| `provider-attribution.ts` | — | `coding-agent/util/attribution.go` | Provider 署名 |
| `session-cwd.ts` | — | `coding-agent/session/cwd.go` | 会话工作目录 |

## 第 2 批：中等文件（本周）

| TS | 行 | Go 目标 | 说明 |
|---|---|---|---|
| `model-resolver.ts` | 707 | `coding-agent/model/resolver.go` | 模型解析+补全 |
| `model-runtime.ts` | 602 | `coding-agent/model/runtime.go` | 模型运行时 |
| `resource-loader.ts` | 1040 | `coding-agent/resource/loader.go` | 资源加载 |
| `settings-manager.ts` | 1234 | `coding-agent/config/settings.go` | 设置管理 |
| `auth-storage.ts` | 271 | `coding-agent/auth/storage.go` | 凭据存储 |
| `http-dispatcher.ts` | — | `coding-agent/http/dispatch.go` | HTTP 分发 |

## 第 3 批：核心大文件（下周）

| TS | 行 | Go 目标 | 说明 |
|---|---|---|---|
| `agent-session.ts` | 3322 | `coding-agent/session/agent.go` | Agent 会话 |
| `agent-session-runtime.ts` | 438 | `coding-agent/session/runtime.go` | 会话运行时 |
| `agent-session-services.ts` | 219 | `coding-agent/session/services.go` | 会话服务 |
| `session-manager.ts` | 1712 | `coding-agent/session/manager.go` | 会话管理 |
| `provider-composer.ts` | 548 | `coding-agent/provider/composer.go` | Provider 组合 |

## 第 4 批：延后/不需要

| TS | 原因 |
|---|---|
| `package-manager.ts` (2650) | pi 特定，不需要 |
| `footer-data-provider.ts` (388) | TUI 专用 |
| `keybindings.ts` (370) | TUI 专用 |
| `output-guard.ts` | TUI 专用 |
| `sdk.ts` (398) | SDK 模式 |
| `compaction/` (4) | 需要 TUI |
| `extensions/` (4) | 扩展系统 |
| `export-html/` (3) | HTML 导出 |
| `diagnostics.ts` | CLI 诊断 |
| `experimental.ts` | 实验功能 |
