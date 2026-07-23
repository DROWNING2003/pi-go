# Pi Go Core-first 任务清单

> 所有任务通过 PR 提交。每个任务遵循 RED -> GREEN -> REFACTOR，单次不超过约 5 个文件。

## 执行规则

- 不直接提交 `main`，由项目负责人审核和合并 PR。
- 先写失败测试，确认失败原因是目标行为缺失。
- 只实现当前任务需要的最小代码，再重构。
- 每个任务必须写清验收标准、验证命令、依赖和文件范围。
- Core checkpoint 前不引入 TUI、扩展或 server 实现。

## 阶段 0：范围与基线

### Task 0：确认 Core-first Spec

**说明：** 确认先做 `ai`、`agent`、`storage`，并把 TUI、扩展和 server 延后。

**验收标准：**
- [ ] `docs/spec.md` 明确 core-first 范围。
- [ ] 首批 Provider、Session 兼容范围和 Core checkpoint 已确认。
- [ ] TUI、扩展、server 被标记为后续阶段。

**验证：** 人工审阅 `docs/spec.md` 和 `docs/plan.md`。

**依赖：** 无
**文件：** `docs/spec.md`、`docs/plan.md`
**规模：** XS

### Task 1：建立 Core 兼容矩阵

**说明：** 只记录 AI、Agent、Tool、Session 和 Config 的 TypeScript 行为。

**验收标准：**
- [ ] Message/event、Provider、工具、Session 和配置各有兼容分类。
- [ ] CLI/TUI 行为单独标记为后续范围。
- [ ] 每项必须兼容行为有 fixture 或现有测试来源。

**验证：** 人工对照原 Pi 的 `packages/ai`、`packages/agent`、`packages/coding-agent` 和测试目录。

**依赖：** Task 0
**文件：** `docs/compatibility-matrix.md`
**规模：** S

### Task 2：提取 Core fixtures

**说明：** 准备不含秘密的标准消息、Provider stream、工具调用和 Session fixture。

**验收标准：**
- [ ] 包含文本、thinking、tool call、tool result、error 和 abort。
- [ ] 包含线性 Session、parentId 分支和损坏尾记录。
- [ ] fixture 不包含个人路径、API key 或真实 Session。

**验证：** 运行原 Pi 对应特定测试，再人工审阅 fixture。

**依赖：** Task 1
**文件：** `testdata/model/events.jsonl`、`testdata/provider/streams.jsonl`、`testdata/session/branched.jsonl`、`docs/compatibility-matrix.md`
**规模：** M

## 检查点 A：Core baseline

- [ ] Task 0-2 完成。
- [ ] 首批 Provider 和 Session 读写边界已确认。
- [ ] fixture 可以在不访问真实 API 的情况下运行。

## 阶段 1：Package scaffold 和 CI/CD

### Task 3：建立 Core package 结构

**说明：** 建立与原 Pi package 边界一致的 Go 目录，并将 CLI bootstrap 归入 coding-agent。

**验收标准：**
- [ ] 存在 `packages/ai/model`、`packages/ai/provider`、`packages/ai/protocol`。
- [ ] 存在 `packages/agent/loop`、`event`、`tool`、`queue`。
- [ ] 存在 `packages/storage/session`、`config` 和 `packages/coding-agent`。
- [ ] CLI help/version 行为不变。

**验证：** `gofmt -l .`、`go test ./...`、`go build ./packages/coding-agent/cmd/pi`。

**依赖：** 检查点 A
**文件：** `packages/ai/**/doc.go`、`packages/agent/**/doc.go`、`packages/storage/**/doc.go`、`packages/coding-agent/cmd/pi/main.go`、`packages/coding-agent/cli/*`
**规模：** M

### Task 4：建立 Core CI/CD 门禁

**说明：** 保持 CI 在 Core 开发最早阶段生效，验证所有 package、race 和跨平台构建。

**验收标准：**
- [ ] push/PR 执行 gofmt、vet、test、race 和 coverage。
- [ ] darwin/linux amd64/arm64 构建通过。
- [ ] v* tag 的 release workflow 只在发布阶段执行。

**验证：** 本地运行同等命令；提交 PR 后等待全部 required checks 通过。

**依赖：** Task 3
**文件：** `.github/workflows/ci.yml`、`.github/workflows/release.yml`、`README.md`
**规模：** M

## 检查点 B：Package 和 CI

- [ ] Task 3-4 通过 PR 合并。
- [ ] `main` 的 required checks 全部通过。
- [ ] Core package 之间没有反向依赖。

## 阶段 2：`packages/ai` 公共契约

### Task 5：定义标准 Model 和 Message

**说明：** 在 `packages/ai/model` 定义 Provider、Message、ContentBlock、ToolCall、ToolResult、Usage、Error 和 StopReason。

**验收标准：**
- [ ] 所有 Core fixture 可以解码为 typed struct。
- [ ] round-trip 保留 ID、时间、content 顺序、usage 和错误字段。
- [ ] 未知版本和非法 content type 返回稳定错误。

**验证：** 先确认 round-trip 测试 RED；再运行 `go test ./packages/ai/model`。

**依赖：** 检查点 B、Task 2
**文件：** `packages/ai/model/message.go`、`packages/ai/model/message_test.go`、`packages/ai/model/error.go`、`testdata/model/events.jsonl`
**规模：** M

### Task 6：定义标准 Stream 和 Agent Event

**说明：** 在 `packages/ai/model` 和 `packages/agent/event` 定义 Provider 与 Agent 的稳定事件。

**验收标准：**
- [ ] 支持 text/thinking/toolcall start、delta、end、done、error。
- [ ] 支持 agent、turn、message、tool execution 生命周期。
- [ ] content index、tool call ID 和事件顺序 round-trip 不变。

**验证：** 先确认事件测试 RED；再运行 `go test ./packages/ai/model ./packages/agent/event`。

**依赖：** Task 5
**文件：** `packages/ai/model/event.go`、`packages/ai/model/event_test.go`、`packages/agent/event/event.go`、`packages/agent/event/event_test.go`
**规模：** M

### Task 7：实现 faux Provider

**说明：** 在 `packages/ai/provider` 提供脚本化内存 Provider，供所有 Agent 测试使用。

**验收标准：**
- [ ] 可按脚本发送文本、thinking、tool call、error 和 done。
- [ ] context 取消返回 aborted，不泄漏 goroutine。
- [ ] 测试只断言标准事件和最终消息，不 mock 内部调用。

**验证：** 先确认 faux stream 测试 RED；再运行 `go test ./packages/ai/provider`。

**依赖：** Task 6
**文件：** `packages/ai/provider/provider.go`、`packages/ai/provider/faux.go`、`packages/ai/provider/provider_test.go`、`packages/ai/model/message.go`
**规模：** M

## 检查点 C：AI contract

- [ ] Task 5-7 通过 PR 合并。
- [ ] message/event schema v1 冻结。
- [ ] faux Provider 可驱动文本和 tool call fixture。

## 阶段 3：`packages/ai` Provider core

### Task 8：实现共享 HTTP/SSE/JSON stream

**说明：** 实现不绑定具体 Provider 的取消、超时、HTTP 错误和流分块解析。

**验收标准：**
- [ ] 任意网络分块、CRLF/LF、多个 data 行和流结束都能解析。
- [ ] 非 2xx、超时、取消和 malformed stream 映射为稳定错误。
- [ ] payload debug 自动脱敏。

**验证：** 先确认 `httptest.Server` 测试 RED；再运行 `go test ./packages/ai/protocol`。

**依赖：** Task 6
**文件：** `packages/ai/protocol/stream.go`、`packages/ai/protocol/stream_test.go`、`packages/ai/protocol/http.go`、`packages/ai/protocol/http_test.go`
**规模：** M

### Task 9：实现 OpenAI-compatible Completions

**验收标准：**
- [ ] text、thinking、tool call、usage、stop reason 映射为标准事件。
- [ ] 支持自定义 base URL、headers 和兼容配置。
- [ ] malformed/aborted stream 保留部分消息。

**验证：** 先确认 HTTP fixture 测试 RED；再运行 `go test ./packages/ai/protocol -run Completions`。

**依赖：** Task 8
**文件：** `packages/ai/protocol/openai_completions.go`、`packages/ai/protocol/openai_completions_test.go`、`testdata/provider/openai-completions.jsonl`
**规模：** M

### Task 10：实现 OpenAI Responses

**验收标准：**
- [ ] input/output、reasoning、function call、usage 和 response ID 正确映射。
- [ ] 取消和错误符合标准事件契约。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./packages/ai/protocol -run Responses`。

**依赖：** Task 8
**文件：** `packages/ai/protocol/openai_responses.go`、`packages/ai/protocol/openai_responses_test.go`、`testdata/provider/openai-responses.jsonl`
**规模：** M

### Task 11：实现 Anthropic Messages

**验收标准：**
- [ ] text、thinking、tool use、usage、stop reason 正确映射。
- [ ] replay context 和 tool result payload 与 fixture 一致。
- [ ] 部分流和错误保留已接收内容。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./packages/ai/protocol -run Anthropic`。

**依赖：** Task 8
**文件：** `packages/ai/protocol/anthropic.go`、`packages/ai/protocol/anthropic_test.go`、`testdata/provider/anthropic.jsonl`
**规模：** M

### Task 12：实现 Google Generative AI

**验收标准：**
- [ ] text、thinking、function call、图片输入和 usage 正确映射。
- [ ] safety/error response 有稳定错误类型。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./packages/ai/protocol -run Google`。

**依赖：** Task 8
**文件：** `packages/ai/protocol/google.go`、`packages/ai/protocol/google_test.go`、`testdata/provider/google.jsonl`
**规模：** M

### Task 13：实现 API Key store 和模型目录

**验收标准：**
- [ ] stored credential 优先于环境 fallback，logout 后恢复环境解析。
- [ ] 文件权限为 `0600`，并发修改使用文件锁和原子写入。
- [ ] Provider/model pattern 查询和动态目录刷新可取消。

**验证：** 先确认临时 HOME 测试 RED；再运行 `go test -race ./packages/ai/provider`。

**依赖：** Task 5、Task 9-12
**文件：** `packages/ai/provider/auth.go`、`packages/ai/provider/auth_test.go`、`packages/ai/provider/catalog.go`、`packages/ai/provider/catalog_test.go`
**规模：** M

### Task 14：实现一个 OAuth 纵向切片

**验收标准：**
- [ ] 本地 fixture 可以完成 login、refresh、logout。
- [ ] 并发请求只执行一次 token refresh。
- [ ] refresh 失败保留凭据并要求重新登录。

**验证：** 先确认 OAuth 状态机测试 RED；再运行 `go test -race ./packages/ai/provider -run OAuth`。

**依赖：** Task 13、对应 Provider protocol
**文件：** `packages/ai/provider/oauth.go`、`packages/ai/provider/oauth_test.go`、`packages/ai/provider/auth.go`、OAuth adapter
**规模：** M

## 检查点 D：AI core

- [ ] Task 8-14 通过 PR 合并。
- [ ] 首批 Provider 契约测试通过，不依赖真实 API。
- [ ] 一个 API Key Provider 和一个 OAuth Provider 具备标准 auth/stream contract。

## 阶段 4：`packages/agent` Agent core

### Task 15：定义 Tool contract 和参数校验

**验收标准：**
- [ ] Tool descriptor、JSON Schema、ToolResult 和错误类型可序列化。
- [ ] 缺少参数、错误类型和额外字段按规则拒绝。

**验证：** 先确认 schema 测试 RED；再运行 `go test ./packages/agent/tool`。

**依赖：** Task 5、Task 6
**文件：** `packages/agent/tool/tool.go`、`packages/agent/tool/tool_test.go`、`packages/ai/model/tool.go`
**规模：** S

### Task 16：实现 read 工具和最小 Agent loop

**说明：** 完成第一条核心纵向链路：Provider -> Agent -> read -> tool result -> continuation。

**验收标准：**
- [ ] read 成功和失败都生成标准 tool result。
- [ ] Agent 按 fixture 事件顺序继续下一 turn。
- [ ] 最终 transcript 可交给 Session 保存。

**验证：** 先确认 Agent loop 测试 RED；再运行 `go test ./packages/agent/...`。

**依赖：** Task 7、Task 15
**文件：** `packages/agent/loop/loop.go`、`packages/agent/loop/loop_test.go`、`packages/agent/tool/read.go`、`packages/agent/tool/read_test.go`
**规模：** M

### Task 17：实现 write、edit、bash

**验收标准：**
- [ ] write 支持新建/覆盖、取消和错误。
- [ ] edit 对零/多匹配失败，失败时原文件不变。
- [ ] bash 捕获 stdout/stderr/exit code，支持超时和进程取消。

**验证：** 先确认临时目录和受控子进程测试 RED；再运行 `go test ./packages/agent/tool`。

**依赖：** Task 15、Task 16
**文件：** `packages/agent/tool/write.go`、`packages/agent/tool/edit.go`、`packages/agent/tool/bash.go`、对应测试文件
**规模：** M

### Task 18：实现工具并行/串行调度

**验收标准：**
- [ ] parallel 模式按完成顺序发 event、按来源顺序写 transcript。
- [ ] sequential 工具使当前批次串行执行。
- [ ] `go test -race` 无共享状态竞态。

**验证：** 先确认调度测试 RED；再运行 `go test -race ./packages/agent`。

**依赖：** Task 17
**文件：** `packages/agent/loop/scheduler.go`、`packages/agent/loop/scheduler_test.go`、`packages/agent/tool/tool.go`
**规模：** M

### Task 19：实现 abort、continue、steering、follow-up

**验收标准：**
- [ ] abort 同时取消 Provider 和工具，保留 partial assistant message。
- [ ] continue 只从 user/toolResult 状态继续。
- [ ] steering/follow-up 队列按 one-at-a-time/all 规则工作。

**验证：** 先确认状态机测试 RED；再运行 `go test -race ./packages/agent/loop ./packages/agent/queue`。

**依赖：** Task 18
**文件：** `packages/agent/loop/runtime.go`、`packages/agent/loop/runtime_test.go`、`packages/agent/queue/queue.go`、`packages/agent/queue/queue_test.go`
**规模：** M

## 检查点 E：Agent core

- [ ] Task 15-19 通过 PR 合并。
- [ ] faux Provider 能完成文本、read、write、edit、bash 和 continuation。
- [ ] abort、continue、steering、follow-up 有确定状态机测试。
- [ ] `go test -race ./packages/agent/...` 通过。

## 阶段 5：`packages/storage` Storage core

### Task 20：实现 Session JSONL store

**验收标准：**
- [ ] 可以读取旧版 Session 并保留 parentId、message order 和 metadata。
- [ ] 写入使用临时文件 + rename，失败不覆盖旧文件。
- [ ] 损坏尾记录可恢复有效前缀并返回诊断信息。

**验证：** 先确认旧 Session fixture 测试 RED；再运行 `go test ./packages/storage/session`。

**依赖：** Task 5、Task 16
**文件：** `packages/storage/session/store.go`、`packages/storage/session/store_test.go`、`testdata/session/branched.jsonl`、`testdata/session/corrupt-tail.jsonl`
**规模：** M

### Task 21：实现 Session tree、resume 和 fork

**验收标准：**
- [ ] 可以计算活动 parentId 路径。
- [ ] fork 创建新 Session，tree 切换不修改未选分支。
- [ ] resume/continue 可以追加 Agent transcript。

**验证：** 先确认 tree/fork 测试 RED；再运行 `go test ./packages/storage/session -run 'Tree|Fork|Resume'`。

**依赖：** Task 20
**文件：** `packages/storage/session/tree.go`、`packages/storage/session/tree_test.go`、`packages/storage/session/store.go`
**规模：** M

### Task 22：实现 config、context files 和 trust

**验收标准：**
- [ ] 全局/项目配置和环境变量优先级确定。
- [ ] `AGENTS.md`/`CLAUDE.md` 加载顺序与项目规则一致。
- [ ] trust 决策可保存，非交互模式不显示交互提示。

**验证：** 先确认临时 HOME/project fixture 测试 RED；再运行 `go test ./packages/storage/config`。

**依赖：** Task 0、Task 20
**文件：** `packages/storage/config/config.go`、`packages/storage/config/config_test.go`、`packages/storage/config/context.go`、`packages/storage/config/context_test.go`
**规模：** M

## Core checkpoint

- [ ] Task 20-22 通过 PR 合并。
- [ ] `packages/ai`、`packages/agent`、`packages/storage` 不依赖 TUI、扩展或 server。
- [ ] 一个 faux Provider 可以完成 prompt -> stream -> tool -> continuation -> Session。
- [ ] 旧版 Session 可以读取、追加、恢复和 fork。
- [ ] `go test ./...`、`go test -race ./...`、`go vet ./...`、CI cross-build 全部通过。
- [ ] Core API 文档和兼容矩阵已更新。
- [ ] 人工批准进入 UI/外围阶段。

## Core 之后的待办队列

### Deferred A：coding-agent CLI

- [ ] print mode 组装 Core。
- [ ] JSONL/RPC mode 组装 Core。
- [ ] login、model、resume、compact、export 命令。

### Deferred B：TUI

- [ ] Bubble Tea model、编辑器、resize、粘贴和 Escape。
- [ ] 流式消息、工具进度、主题、Markdown 和图片回退。
- [ ] TUI golden 和 tmux 测试。

### Deferred C：外围能力

- [ ] 子进程 JSON-RPC 扩展协议和宿主。
- [ ] 其余 Provider，每个 Provider 单独一张任务卡。
- [ ] SQLite index 和 server 包。
- [ ] parity、迁移、发布 smoke test 和 TypeScript 退役。

每个 Deferred 项目在开始前必须建立独立 PR 和详细任务卡，不得把它们混入 Core PR。
