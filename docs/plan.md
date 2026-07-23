# Pi Go Core-first 实施计划

> 前置规格：[spec.md](./spec.md)。所有实现通过短分支和 PR 进入 `main`，由项目负责人审核和合并。

## 一、当前目标

先完成 Pi 的核心运行时，不先实现完整 TUI 或外围功能。核心必须能独立完成：

```text
标准 Model/Message
  -> Provider stream
  -> Agent loop
  -> Tool execution
  -> Tool result
  -> Session JSONL
```

当前里程碑结束时，Go core 可以被 print、RPC 和未来 TUI 共同使用，但这些 UI/CLI 入口不属于本阶段的主要实现范围。

## 二、目录边界

```text
packages/
  ai/
    model/       标准消息、内容块、usage、错误、stop reason
    provider/    Provider interface、模型目录、认证
    protocol/    OpenAI、Anthropic、Google、SSE wire protocol

  agent/
    loop/        Agent turn、tool call 和状态机
    event/       Agent lifecycle event
    tool/        工具契约、参数校验和执行
    queue/       abort、continue、steering、follow-up

  storage/
    session/     JSONL、原子写入、恢复、分支
    config/      settings、context files、trust、凭据

  coding-agent/
    cli/         当前只保留最小 help/version 入口
    cmd/pi/      可执行入口；print、RPC、TUI 后续接入

  tui/           后续阶段：Bubble Tea、编辑器、主题、终端渲染
  extension/     后续阶段：子进程 JSON-RPC 扩展
  server/        后续阶段：server 包

docs/             Spec、Plan、Tasks、兼容矩阵
testdata/         Provider、Model、Agent、Session fixture
scripts/          parity、迁移、构建和发布脚本
```

依赖方向固定为：

```text
packages/ai/model
    -> packages/ai/provider + packages/ai/protocol
    -> packages/agent/event + packages/agent/loop + packages/agent/tool
    -> packages/storage/session + packages/storage/config
    -> packages/coding-agent / packages/tui / packages/extension / packages/server
```

约束：

- `packages/ai` 不依赖 Agent、Storage、CLI 或 TUI。
- `packages/agent` 依赖 AI contract，不依赖 TUI 或 CLI。
- `packages/storage` 只负责持久化和配置，不拥有 Agent loop。
- `packages/coding-agent` 负责组装 core，不把业务状态机写进 CLI。
- `packages/tui` 后续只消费 Agent events，不直接调用 Provider。

## 三、技术栈

- Go 1.26.1，Go Modules
- `github.com/spf13/cobra` v1.10.2 + PFlag：仅用于 coding-agent CLI
- `net/http`、`context.Context`、`encoding/json`
- 标准库 SSE/JSON stream parser，Provider 协议稳定后再评估外部 SSE 依赖
- Go typed structs + JSON Schema：核心公共 contract 和工具参数
- goroutine、channel、`errgroup`：Provider stream、Agent event 和工具并发
- JSONL Session，临时文件 + rename 原子写入
- GitHub Actions：从 package scaffold 阶段开始执行 gofmt、vet、test、race、cross-build
- TDD：RED -> GREEN -> REFACTOR

TUI 技术栈 Bubble Tea、Lip Gloss、`x/term`、Glamour、Chroma 延后到 Core checkpoint 通过后再引入。

## 四、依赖图和阶段

### 阶段 0：规格与行为基线

- 确认 core-first 范围和首批 Provider。
- 建立 TypeScript 到 Go 的兼容矩阵。
- 提取不含秘密的 Model、Provider event 和 Session fixture。

### 阶段 1：Package scaffold 和 CI/CD

- 建立 `packages/ai`、`packages/agent`、`packages/storage`、`packages/coding-agent`。
- 将最小 CLI 归入 `packages/coding-agent`。
- CI/CD 在所有 core 行为前生效。

### 阶段 2：AI contracts

- 定义 Message、ContentBlock、ToolCall、ToolResult、Usage、Error 和 StopReason。
- 定义 Provider stream event 和 Agent lifecycle event。
- 建立 faux Provider。

### 阶段 3：Provider core

- 共享 HTTP/SSE/JSON stream parser。
- OpenAI-compatible Completions。
- OpenAI Responses。
- Anthropic Messages。
- Google Generative AI。
- API Key credential store、模型目录和一个 OAuth 纵向切片。

### 阶段 4：Agent core

- Agent loop、turn 状态机和标准 event stream。
- read、write、edit、bash。
- 工具参数校验、并行/串行调度、超时和取消。
- abort、continue、steering、follow-up。

### 阶段 5：Storage core

- Session JSONL 读取、写入和恢复。
- 原子写入、损坏尾记录恢复、parentId tree、fork。
- compaction 所需的上下文接口，但先不实现 TUI。
- 配置、context files、trust 和凭据路径。

### Core checkpoint

必须同时满足：

- faux Provider 可以完成文本和 tool call stream。
- Agent loop 可以执行四个内置工具并继续 turn。
- abort/continue/steering/follow-up 有状态机测试。
- 旧版 Session fixture 可以读取、追加、fork 和恢复。
- print 或 RPC 可以通过同一 core 完成一次端到端对话。
- `go test ./...`、`go test -race ./...`、`go vet ./...` 和 CI cross-build 通过。

### 阶段 6：Core 之后

Core checkpoint 通过后，才按独立 PR 进入：

1. coding-agent print/RPC 完整 CLI。
2. Bubble Tea TUI、编辑器、主题和终端图片。
3. slash commands、settings、resume/tree UI。
4. extension JSON-RPC。
5. 其余 Provider、SQLite index 和 server。
6. parity、迁移、发布和 TypeScript 退役。

## 五、并行边界

可以并行：

- AI message contract 和 Provider fixture 的测试准备。
- OpenAI、Anthropic、Google 协议 adapter，前提是 stream event schema 已冻结。
- write、edit、bash 工具，前提是 tool contract 已冻结。
- Session serializer 和 config loader，前提是 model contract 已冻结。

必须串行：

- Spec -> package scaffold -> public contracts。
- Message/event contract -> Provider -> Agent loop。
- Agent event -> print/RPC/TUI adapter。
- Session schema 迁移 -> parity -> TypeScript 退役。

## 六、风险控制

| 风险 | 应对 |
|---|---|
| Core 被 TUI 需求拖慢 | TUI 只能消费事件，Core checkpoint 前不引入 TUI 依赖 |
| 包边界再次漂移 | 公共类型只能从 `packages/ai/model` 或明确的 contract 包导出 |
| Provider 细节污染 Agent | 所有 Provider 输出先转换为标准 event/message |
| Session 格式不稳定 | 先用旧 JSONL fixture，schema 变化必须单独 PR |
| 任务过大 | 每个任务最多 5 个文件，超过则拆分 |
| CI 被延后 | package scaffold 后立即启用，所有后续 PR 必须通过门禁 |

## 七、下一阶段完成定义

Core-first 阶段完成，不代表完整 Pi 完成。它只代表以下公共能力已经稳定：

- `packages/ai` 能提供标准 Model、Provider 和 stream contract。
- `packages/agent` 能在 faux Provider 上稳定执行 Agent loop 和工具。
- `packages/storage` 能安全保存、恢复和分支 Session。
- `packages/coding-agent` 可以作为薄组装层调用 Core。
- 后续 TUI、RPC、扩展和 server 都不需要复制 Core 逻辑。
