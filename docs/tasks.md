# Pi Go 重写任务清单

> 前置规格：[spec.md](./spec.md)。任何实现任务都必须遵循 RED -> GREEN -> REFACTOR。

## 执行规则

- 每个任务限制在一次专注会话内完成，通常不超过 5 个文件。
- 先运行新测试并确认 RED，再写最小实现达到 GREEN，最后 REFACTOR。
- 每个任务只运行相关测试；跨包契约或并发变化再运行 `go test ./...` 或 `go test -race ./...`。
- 不允许跳过、删除或弱化测试。
- 每个检查点必须人工确认后才能进入下一组任务。

## 阶段 0：规格和基线

### Task 0：批准重写规格

**说明：** 确认首发范围、兼容边界和成功标准，不修改源码。

**验收标准：**
- [ ] `docs/spec.md` 中 7 项假设均已确认或修改。
- [ ] 首批 Provider、支持平台、Session 和扩展兼容策略已有结论。
- [ ] 测试覆盖率门槛和 server 包范围已有结论。

**验证：** 人工审阅 `docs/spec.md`，勾选其“Spec 审批门槛”。

**依赖：** 无
**文件：** `docs/spec.md`
**规模：** XS

### Task 1：建立 TypeScript 行为兼容矩阵

**说明：** 将 CLI、事件、Session、工具、Provider 和 TUI 行为分为必须兼容、首版暂缺和明确改变。

**验收标准：**
- [ ] 每个公开 CLI 参数和 slash command 都有分类。
- [ ] 每种事件、Session 操作和内置工具都有验证来源。
- [ ] 所有有意变化都有迁移说明占位。

**验证：** 人工对照现有 README、CLI help 和测试目录审阅矩阵。

**依赖：** Task 0
**文件：** `docs/compatibility-matrix.md`
**规模：** S

### Task 2：提取兼容 fixture

**说明：** 从现有 faux provider 和 Session 测试中提取不含凭据的输入输出 fixture。

**验收标准：**
- [ ] 至少包含文本、thinking、tool call、error 和 abort 事件。
- [ ] 至少包含线性、分支和 compaction Session。
- [ ] fixture 中不包含真实路径、凭据或个人会话。

**验证：** `npm run check`；运行产生 fixture 的现有特定测试。

**依赖：** Task 1
**文件：** `testdata/providers/events.jsonl`、`testdata/sessions/linear.jsonl`、`testdata/sessions/branched.jsonl`、相关 TypeScript fixture 测试
**规模：** M

## 检查点 A：规格与基线

- [ ] Task 0-2 已完成。
- [ ] TypeScript `npm run check` 通过。
- [ ] fixture 已人工检查，不含秘密和机器专属数据。
- [ ] 人工批准进入 Go 实现。

## 阶段 1：第一个可运行纵向切片

### Task 3：创建 Go 工程和 CLI 骨架

**说明：** 创建固定 Go 1.26.1 toolchain 的最小工程，支持 help 和 version。

**验收标准：**
- [ ] `go.mod` 固定 Go 1.26.1 toolchain 和审核后的 Cobra/PFlag 版本。
- [ ] `pi --help` 返回 0，未知参数返回稳定的非零退出码。
- [ ] `pi --version` 输出注入的版本。

**验证：** 先确认 CLI 测试 RED；再运行 `go test ./internal/cli ./cmd/pi` 和 `go build -o bin/pi ./cmd/pi`。

**依赖：** 检查点 A
**文件：** `go.mod`、`go.sum`、`cmd/pi/main.go`、`internal/cli/root.go`、`internal/cli/root_test.go`
**规模：** M

### Task 4：建立初始 CI/CD 门禁

**说明：** 在任何核心行为实现前建立 GitHub Actions，持续验证格式、静态检查、测试、竞态和跨平台构建，并为 tag 发布准备可回滚的 Release workflow。

**验收标准：**
- [ ] push 和 pull request 自动执行 gofmt、go vet、go test、race test 和覆盖率产物。
- [ ] macOS/Linux 的 amd64/arm64 交叉构建全部成功。
- [ ] v* tag 在重复质量门禁后发布二进制和 SHA256SUMS，工作流不包含真实凭据。

**验证：** 本地执行与 CI 相同的命令；推送后使用 `gh run watch` 确认 CI 成功，并检查 Release workflow 只在 tag 上触发。

**依赖：** Task 3
**文件：** `.github/workflows/ci.yml`、`.github/workflows/release.yml`、`README.md`
**规模：** M

### Task 5：定义消息 JSON 契约

**说明：** 定义 text、thinking、image、tool call、tool result、usage 和 stop reason。

**验收标准：**
- [ ] TypeScript fixture 可以解码为 typed Go struct。
- [ ] 编码后保留 ID、时间、content 顺序、usage 和错误字段。
- [ ] 未知版本和非法 content type 返回可识别错误。

**验证：** 先确认 round-trip 测试 RED；再运行 `go test ./internal/model`。

**依赖：** Task 4
**文件：** `internal/model/message.go`、`internal/model/message_test.go`、`internal/model/errors.go`、`testdata/providers/events.jsonl`
**规模：** M

### Task 6：定义流式事件契约

**说明：** 定义 Provider stream event 和 Agent lifecycle event 的类型、顺序和 JSON 表达。

**验收标准：**
- [ ] 支持 text/thinking/toolcall 的 start、delta、end 和 done/error。
- [ ] 支持 agent、turn、message 和 tool execution 生命周期。
- [ ] 事件 fixture round-trip 不改变顺序或 content index。

**验证：** 先确认事件契约测试 RED；再运行 `go test ./internal/model`。

**依赖：** Task 5
**文件：** `internal/model/event.go`、`internal/model/event_test.go`、`internal/model/message.go`
**规模：** M

### Task 7：实现 faux Provider 流

**说明：** 提供可脚本化的内存 Provider，用于后续 Agent 测试，不访问网络。

**验收标准：**
- [ ] 可以按顺序发送文本、thinking、tool call 和 error。
- [ ] 收到 context 取消后停止并返回 aborted。
- [ ] 测试可以检查最终标准消息，不需要 mock 内部调用。

**验证：** 先确认 faux stream 测试 RED；再运行 `go test ./internal/provider/faux`。

**依赖：** Task 6
**文件：** `internal/provider/provider.go`、`internal/provider/faux/faux.go`、`internal/provider/faux/faux_test.go`
**规模：** M

## 检查点 B：公共契约

- [ ] Task 3-7 已完成。
- [ ] `go test ./internal/model ./internal/provider/faux ./internal/cli` 通过。
- [ ] message/event schema 人工冻结为 v1。
- [ ] 后续分支不得自行改变 schema。

### Task 8：跑通 print 文本对话

**说明：** 用 faux Provider 完成“prompt -> 流式事件 -> stdout 文本”的第一个端到端切片。

**验收标准：**
- [ ] `pi --print` 接收一个 prompt 并输出完整文本。
- [ ] Provider 错误写入 stderr，stdout 不混入诊断日志。
- [ ] 成功、错误和取消使用稳定退出码。

**验证：** 先确认 CLI 集成测试 RED；再运行 `go test ./internal/agent ./internal/cli` 和 `go build ./cmd/pi`。

**依赖：** 检查点 B
**文件：** `internal/agent/runner.go`、`internal/agent/runner_test.go`、`internal/cli/print.go`、`internal/cli/print_test.go`
**规模：** M

### Task 9：保存并恢复线性 Session

**说明：** 在 print 切片中加入现有 JSONL 兼容的原子 Session 写入和恢复。

**验收标准：**
- [ ] prompt 和 assistant message 以兼容 JSONL 写入临时 Session。
- [ ] 中断写入不会覆盖已有有效 Session。
- [ ] `--continue` 可以读取最近 Session 并追加一次对话。

**验证：** 先确认旧 fixture 读取和中断写入测试 RED；再运行 `go test ./internal/session ./internal/cli`。

**依赖：** Task 8
**文件：** `internal/session/store.go`、`internal/session/store_test.go`、`internal/cli/session.go`、`internal/cli/session_test.go`
**规模：** M

### Task 10：加入 read 工具的完整 Agent 循环

**说明：** 用 faux Provider 跑通“模型发出 tool call -> read 执行 -> tool result -> 模型继续回答”。

**验收标准：**
- [ ] read 参数通过 JSON Schema 校验。
- [ ] 成功和失败都生成标准 tool result 并写入 Session。
- [ ] 事件顺序与冻结 fixture 一致。

**验证：** 先确认工具循环测试 RED；再运行 `go test ./internal/tool ./internal/agent ./internal/session`。

**依赖：** Task 9
**文件：** `internal/tool/read.go`、`internal/tool/read_test.go`、`internal/agent/tools.go`、`internal/agent/tools_test.go`、`internal/session/store.go`
**规模：** M

## 检查点 C：最小 Agent

- [ ] Task 8-10 已完成。
- [ ] `go test ./...` 通过。
- [ ] faux Provider 能完成一次文本对话和一次 read 工具循环。
- [ ] Session 可由 Go 版继续读取。

## 阶段 2：工具、队列和 Session

### Task 11：实现 write 工具

**说明：** 完成“实现 write 工具”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 支持新建和覆盖文本文件，并返回规范化结果。
- [ ] 非法参数、父目录缺失和取消产生 tool error。

**验证：** 先确认临时目录测试 RED；再运行 `go test ./internal/tool -run Write`。

**依赖：** Task 10
**文件：** `internal/tool/write.go`、`internal/tool/write_test.go`、`internal/tool/schema.go`
**规模：** S

### Task 12：实现 edit 工具

**说明：** 完成“实现 edit 工具”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 唯一 oldText 被精确替换，多处或零处匹配时失败。
- [ ] 多项不重叠编辑按原始文件坐标应用并原子写入。
- [ ] 失败时原文件保持不变。

**验证：** 先确认替换边界测试 RED；再运行 `go test ./internal/tool -run Edit`。

**依赖：** Task 11
**文件：** `internal/tool/edit.go`、`internal/tool/edit_test.go`、`internal/tool/schema.go`
**规模：** M

### Task 13：实现 bash 工具

**说明：** 完成“实现 bash 工具”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 捕获 stdout、stderr、退出码和超时。
- [ ] context 取消时终止进程树并返回 aborted。
- [ ] 大输出按已批准的限制截断并保留诊断信息。

**验证：** 先确认受控子进程测试 RED；再运行 `go test ./internal/tool -run Bash`。

**依赖：** Task 10
**文件：** `internal/tool/bash.go`、`internal/tool/bash_test.go`、`internal/tool/process_unix.go`、`internal/tool/schema.go`
**规模：** M

### Task 14：实现并行和串行工具调度

**说明：** 完成“实现并行和串行工具调度”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] parallel 模式按完成顺序发事件、按来源顺序写 transcript。
- [ ] 任一 sequential 工具使当前批次串行执行。
- [ ] `go test -race` 不报告共享状态竞态。

**验证：** 先确认调度事件测试 RED；再运行 `go test -race ./internal/agent -run ToolExecution`。

**依赖：** Task 11、Task 12、Task 13
**文件：** `internal/agent/scheduler.go`、`internal/agent/scheduler_test.go`、`internal/agent/tools.go`
**规模：** M

## 检查点 D：内置工具

- [ ] Task 11-14 已完成。
- [ ] `go test -race ./internal/agent ./internal/tool` 通过。
- [ ] 四个内置工具的成功、失败、取消和超时均有测试。

### Task 15：实现 abort 和 continue

**说明：** 完成“实现 abort 和 continue”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] abort 取消 Provider 和正在执行的工具。
- [ ] 部分 assistant message 保留并标记 aborted。
- [ ] continue 只允许从 user/toolResult 状态继续。

**验证：** 先确认状态机测试 RED；再运行 `go test -race ./internal/agent -run 'Abort|Continue'`。

**依赖：** Task 14
**文件：** `internal/agent/runtime.go`、`internal/agent/runtime_test.go`、`internal/agent/runner.go`
**规模：** M

### Task 16：实现 steering 和 follow-up 队列

**说明：** 完成“实现 steering 和 follow-up 队列”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] steering 在当前工具批次完成后进入下一 turn。
- [ ] follow-up 只在无工具和 steering 时发送。
- [ ] one-at-a-time/all 模式和清空队列行为有测试。

**验证：** 先确认队列测试 RED；再运行 `go test -race ./internal/agent -run Queue`。

**依赖：** Task 15
**文件：** `internal/agent/queue.go`、`internal/agent/queue_test.go`、`internal/agent/runtime.go`
**规模：** M

### Task 17：实现 Session 分支和 tree

**说明：** 完成“实现 Session 分支和 tree”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 读取旧版 parentId 树并计算活动路径。
- [ ] fork 创建新 Session，tree 切换不丢失原分支。
- [ ] 部分损坏的尾记录可诊断并恢复有效前缀。

**验证：** 先确认分支 fixture 测试 RED；再运行 `go test ./internal/session -run 'Tree|Fork|Recover'`。

**依赖：** Task 9、Task 16
**文件：** `internal/session/tree.go`、`internal/session/tree_test.go`、`internal/session/store.go`、`testdata/sessions/branched.jsonl`
**规模：** M

## 检查点 E：Agent 状态

- [ ] Task 15-17 已完成。
- [ ] `go test -race ./internal/agent ./internal/session` 通过。
- [ ] abort、continue、steering、follow-up、fork 和 tree 行为与矩阵一致。

## 阶段 3：真实 Provider 和认证

### Task 18：实现共享 HTTP/SSE 客户端

**说明：** 完成“实现共享 HTTP/SSE 客户端”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 正确处理任意网络分块、CRLF/LF、多个 data 行和流结束。
- [ ] 超时、context 取消、非 2xx 和格式错误映射为标准错误。
- [ ] debug payload 自动脱敏。

**验证：** 先确认 `httptest.Server` 测试 RED；再运行 `go test ./internal/protocol/httpstream`。

**依赖：** 检查点 B
**文件：** `internal/protocol/httpstream/client.go`、`internal/protocol/httpstream/client_test.go`、`internal/protocol/httpstream/sse.go`、`internal/protocol/httpstream/sse_test.go`
**规模：** M

### Task 19：实现 OpenAI-compatible Completions

**说明：** 完成“实现 OpenAI-compatible Completions”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 文本、thinking、tool call、usage 和 stop reason 映射为标准事件。
- [ ] 支持自定义 base URL、headers 和关键兼容配置。
- [ ] malformed/aborted stream 保留部分消息。

**验证：** 先确认 HTTP fixture 契约测试 RED；再运行 `go test ./internal/protocol/openai -run Completions`。

**依赖：** Task 18
**文件：** `internal/protocol/openai/completions.go`、`internal/protocol/openai/completions_test.go`、`testdata/providers/openai-completions.jsonl`
**规模：** M

### Task 20：实现 OpenAI Responses

**说明：** 完成“实现 OpenAI Responses”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] Responses input/output、reasoning、function call 和 usage 正确映射。
- [ ] response ID 和缓存相关字段按 Spec 保存。
- [ ] 错误和取消符合标准事件契约。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./internal/protocol/openai -run Responses`。

**依赖：** Task 18
**文件：** `internal/protocol/openai/responses.go`、`internal/protocol/openai/responses_test.go`、`testdata/providers/openai-responses.jsonl`
**规模：** M

### Task 21：实现 Anthropic Messages

**说明：** 完成“实现 Anthropic Messages”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] text、thinking、tool use、cache usage 和 stop reason 正确映射。
- [ ] replay context 和 tool result payload 与 fixture 一致。
- [ ] 部分流和错误保留已接收内容。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./internal/protocol/anthropic`。

**依赖：** Task 18
**文件：** `internal/protocol/anthropic/messages.go`、`internal/protocol/anthropic/messages_test.go`、`testdata/providers/anthropic-messages.jsonl`
**规模：** M

## 检查点 F：首批协议

- [ ] Task 18-21 已完成。
- [ ] `go test ./internal/protocol/...` 通过。
- [ ] 三种协议对同一标准场景产生兼容事件。
- [ ] 人工审查所有 outbound payload fixture。

### Task 22：实现 Google Generative AI

**说明：** 完成“实现 Google Generative AI”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] text、thinking、function call、图片输入和 usage 正确映射。
- [ ] 非流式 function call 被转换为完整 toolcall 事件。
- [ ] safety/error response 有稳定错误类型。

**验证：** 先确认 fixture 测试 RED；再运行 `go test ./internal/protocol/google`。

**依赖：** Task 18
**文件：** `internal/protocol/google/generative.go`、`internal/protocol/google/generative_test.go`、`testdata/providers/google-generative.jsonl`
**规模：** M

### Task 23：实现 API Key 凭据存储

**说明：** 完成“实现 API Key 凭据存储”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] stored credential 优先于环境 fallback，logout 后恢复环境解析。
- [ ] 文件权限为 `0600`，并发修改使用文件锁和原子写入。
- [ ] 日志和错误不包含 secret。

**验证：** 先确认临时 HOME 测试 RED；再运行 `go test -race ./internal/config ./internal/provider/auth`。

**依赖：** Task 3
**文件：** `internal/provider/auth/store.go`、`internal/provider/auth/store_test.go`、`internal/config/paths.go`、`internal/config/paths_test.go`
**规模：** M

### Task 24：实现一个 OAuth 纵向切片

**说明：** 先实现 Spec 选定的首个 OAuth Provider，验证登录、刷新和 logout 架构。

**验收标准：**
- [ ] 登录状态机可用本地 fixture 完成，不依赖真实账户。
- [ ] 并发请求只执行一次 token refresh。
- [ ] refresh 失败保留凭据并要求重新登录，不回退到其他 secret。

**验证：** 先确认 OAuth 状态机测试 RED；再运行 `go test -race ./internal/provider/auth -run OAuth`。

**依赖：** Task 23、对应协议任务
**文件：** `internal/provider/auth/oauth.go`、`internal/provider/auth/oauth_test.go`、`internal/provider/auth/store.go`、对应 Provider adapter
**规模：** M

### Task 25：实现模型目录和 Provider 选择

**说明：** 完成“实现模型目录和 Provider 选择”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 可以按 Provider、模型 ID 和 pattern 查询模型。
- [ ] 动态目录刷新可取消，失败保留最后已知目录。
- [ ] CLI `--provider`、`--model` 和 `--list-models` 使用同一目录。

**验证：** 先确认目录测试 RED；再运行 `go test ./internal/provider ./internal/cli -run Model`。

**依赖：** Task 19-24
**文件：** `internal/provider/catalog.go`、`internal/provider/catalog_test.go`、`internal/cli/models.go`、`internal/cli/models_test.go`
**规模：** M

## 检查点 G：真实模型切片

- [ ] Task 22-25 已完成。
- [ ] 一个 API Key Provider 和一个 OAuth Provider 通过本地契约测试。
- [ ] 受控人工 smoke test 可以获得一次真实流式回答，且不记录凭据。
- [ ] `go test -race ./internal/provider/... ./internal/protocol/...` 通过。

## 阶段 4：RPC 和 TUI 纵向切片

### Task 26：实现 JSONL/RPC framing

**说明：** 完成“实现 JSONL/RPC framing”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 只按 LF 分帧，支持 request ID、错误和取消。
- [ ] stdout 只包含协议 JSONL，日志仅写 stderr。
- [ ] malformed request 不终止后续有效 request。

**验证：** 先确认 framing 测试 RED；再运行 `go test ./internal/rpc`。

**依赖：** Task 8、Task 16
**文件：** `internal/rpc/protocol.go`、`internal/rpc/protocol_test.go`、`internal/rpc/server.go`、`internal/rpc/server_test.go`
**规模：** M

### Task 27：实现 RPC 完整 Agent 对话

**说明：** 完成“实现 RPC 完整 Agent 对话”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] RPC 可以启动 prompt、接收流式事件、执行工具和 abort。
- [ ] Session ID 和最终状态可查询。
- [ ] 事件顺序与 print 和 Agent 测试一致。

**验证：** 先确认进程内集成测试 RED；再运行 `go test -race ./internal/rpc -run Agent`。

**依赖：** Task 26、Task 17、Task 25
**文件：** `internal/rpc/agent.go`、`internal/rpc/agent_test.go`、`internal/rpc/server.go`
**规模：** M

### Task 28：建立 Bubble Tea TUI 外壳

**说明：** 完成“建立 Bubble Tea TUI 外壳”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 启动、resize、quit 和终端恢复映射为确定的 model 状态。
- [ ] 空界面包含 header、message viewport、editor 占位和 footer。
- [ ] 单元测试无需真实终端。

**验证：** 先确认 model 测试 RED；再运行 `go test ./internal/tui -run Shell`。

**依赖：** Task 3；Bubble Tea/Lip Gloss 版本需先审核
**文件：** `internal/tui/model.go`、`internal/tui/model_test.go`、`internal/tui/view.go`、`internal/tui/view_test.go`、`go.mod`
**规模：** M

## 检查点 H：交互外壳

- [ ] Task 26-28 已完成。
- [ ] RPC Agent 对话通过。
- [ ] TUI shell 在 80x24 和 120x40 golden 下无越界。
- [ ] 人工批准继续添加交互组件。

### Task 29：实现多行编辑器和粘贴

**说明：** 完成“实现多行编辑器和粘贴”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 输入、移动、删除、换行、提交和 Unicode 宽度正确。
- [ ] bracketed paste 和大粘贴不会触发意外提交。
- [ ] 自定义 key map 驱动行为，不硬编码业务快捷键。

**验证：** 先确认 editor 状态测试 RED；再运行 `go test ./internal/tui/editor`。

**依赖：** Task 28
**文件：** `internal/tui/editor/model.go`、`internal/tui/editor/model_test.go`、`internal/tui/editor/keys.go`、`internal/tui/editor/keys_test.go`
**规模：** M

### Task 30：渲染流式消息和工具事件

**说明：** 完成“渲染流式消息和工具事件”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] TUI 消费 Agent 事件，不直接访问 Provider。
- [ ] text、thinking、tool progress、error 和 abort 状态稳定更新。
- [ ] 动态内容不会改变 editor/footer 固定布局。

**验证：** 先确认 event-to-view 测试 RED；再运行 `go test ./internal/tui -run Message`。

**依赖：** Task 27、Task 28
**文件：** `internal/tui/messages.go`、`internal/tui/messages_test.go`、`internal/tui/model.go`、`internal/tui/view.go`
**规模：** M

### Task 31：连接 TUI Prompt 和 Agent

**说明：** 完成“连接 TUI Prompt 和 Agent”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] Enter 提交 prompt 并流式显示回复。
- [ ] Escape 取消当前运行，queued input 恢复到 editor。
- [ ] 连续两次 Ctrl+C 或 `/quit` 正确恢复终端并退出。

**验证：** 先确认集成测试 RED；再运行 `go test -race ./internal/tui -run Agent`，并用 tmux 完成一次 faux 对话。

**依赖：** Task 29、Task 30
**文件：** `internal/tui/app.go`、`internal/tui/app_test.go`、`internal/tui/model.go`、`cmd/pi/main.go`
**规模：** M

## 检查点 I：可用 TUI

- [ ] Task 29-31 已完成。
- [ ] 用户可以在 TUI 完成 faux Provider 对话和工具调用。
- [ ] tmux 验证启动、resize、提交、Escape 和退出。
- [ ] `go test -race ./internal/tui/... ./internal/agent/...` 通过。

## 阶段 5：用户功能补齐

### Task 32：实现 slash command 和文件补全

**说明：** 完成“实现 slash command 和文件补全”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] `/` 显示命令补全，`@` 和 Tab 补全项目文件。
- [ ] ignore 规则、相对路径和 home 路径行为符合矩阵。
- [ ] 大目录扫描可取消且不阻塞输入循环。

**验证：** 先确认补全测试 RED；再运行 `go test ./internal/tui/autocomplete`。

**依赖：** Task 29
**文件：** `internal/tui/autocomplete/provider.go`、`internal/tui/autocomplete/provider_test.go`、`internal/tui/editor/model.go`
**规模：** M

### Task 33：实现 login、model 和 settings UI

**说明：** 完成“实现 login、model 和 settings UI”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] `/login`、`/logout` 和 `/model` 驱动统一 Provider/Auth API。
- [ ] `/settings` 可修改 thinking、theme 和 queue 模式。
- [ ] overlay 的焦点、取消和窄终端布局有测试。

**验证：** 先确认 selector 测试 RED；再运行 `go test ./internal/tui -run 'Login|Model|Settings'`。

**依赖：** Task 24、Task 25、Task 31
**文件：** `internal/tui/login.go`、`internal/tui/model_select.go`、`internal/tui/settings.go`、`internal/tui/selectors_test.go`、`internal/tui/model.go`
**规模：** M

### Task 34：实现 Session 命令和 tree UI

**说明：** 完成“实现 Session 命令和 tree UI”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] `/resume`、`/new`、`/session`、`/fork` 和 `/tree` 操作同一 Session API。
- [ ] tree 搜索、选择和取消不修改未选分支。
- [ ] 切换 Session 后 TUI 与 Agent 状态同步。

**验证：** 先确认 Session UI 测试 RED；再运行 `go test ./internal/tui -run 'Session|Tree'`。

**依赖：** Task 17、Task 31
**文件：** `internal/tui/session.go`、`internal/tui/tree.go`、`internal/tui/session_test.go`、`internal/tui/model.go`
**规模：** M

### Task 35：实现 compaction 和 export

**说明：** 完成“实现 compaction 和 export”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 手动和阈值触发 compaction，原始历史仍保留在 Session tree。
- [ ] HTML 和 JSONL 导出不包含凭据。
- [ ] compaction 失败不破坏现有 Session。

**验证：** 先确认旧 Session fixture 测试 RED；再运行 `go test ./internal/session ./internal/export`。

**依赖：** Task 17、Task 25
**文件：** `internal/session/compact.go`、`internal/session/compact_test.go`、`internal/export/export.go`、`internal/export/export_test.go`
**规模：** M

### Task 36：实现 Markdown、代码高亮和主题

**说明：** 完成“实现 Markdown、代码高亮和主题”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] Markdown、代码块和 ANSI 宽度在窄/宽终端不越界。
- [ ] dark/light 和自定义主题可加载、校验和热更新。
- [ ] golden fixture 足够小且经过人工审查。

**验证：** 先确认 renderer 测试 RED；再运行 `go test ./internal/tui/render`。

**依赖：** Task 30；Glamour/Chroma 版本需先审核
**文件：** `internal/tui/render/markdown.go`、`internal/tui/render/markdown_test.go`、`internal/tui/render/theme.go`、`internal/tui/render/theme_test.go`、`go.mod`
**规模：** M

### Task 37：实现终端图片显示和回退

**说明：** 完成“实现终端图片显示和回退”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 正确识别 Kitty/iTerm2 能力并输出对应协议。
- [ ] 不支持图片的终端显示稳定文本占位。
- [ ] 尺寸限制防止图片破坏 viewport。

**验证：** 先确认编码和 fallback 测试 RED；再运行 `go test ./internal/terminal -run Image`。

**依赖：** Task 30
**文件：** `internal/terminal/image.go`、`internal/terminal/image_test.go`、`internal/tui/messages.go`
**规模：** M

## 检查点 J：核心功能对齐

- [ ] Task 32-37 已完成。
- [ ] 必须支持的 slash commands 均在兼容矩阵中通过。
- [ ] Session、compaction、export、主题和图片回退通过。
- [ ] tmux 在至少两种终端尺寸完成真实 Provider 对话。

## 阶段 6：扩展、其余 Provider 和外围包

### Task 38：定义扩展 JSON-RPC v1

**说明：** 完成“定义扩展 JSON-RPC v1”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 协议包含 initialize、capability、tool、command、event、cancel 和 error。
- [ ] 未知能力和版本不匹配返回明确错误。
- [ ] 协议文档和 Go typed message 使用同一 fixture。

**验证：** 先确认协议 round-trip 测试 RED；再运行 `go test ./internal/extension`。

**依赖：** Task 26
**文件：** `internal/extension/protocol.go`、`internal/extension/protocol_test.go`、`docs/extension-protocol.md`、`testdata/extensions/protocol.jsonl`
**规模：** M

### Task 39：实现扩展子进程宿主

**说明：** 完成“实现扩展子进程宿主”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 可以启动扩展、注册 tool/command 并处理调用。
- [ ] timeout、cancel、崩溃和脏 stdout 被隔离并诊断。
- [ ] 主进程退出时清理扩展进程。

**验证：** 先确认受控 helper process 测试 RED；再运行 `go test -race ./internal/extension -run Host`。

**依赖：** Task 38
**文件：** `internal/extension/host.go`、`internal/extension/host_test.go`、`internal/extension/process_unix.go`、`testdata/extensions/helper.go`
**规模：** M

### Task 40：提供扩展示例

**说明：** 完成“提供扩展示例”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 一个 Go 示例和一个 TypeScript 示例实现相同 echo tool。
- [ ] 示例只依赖公开协议，不导入 `internal` 包。
- [ ] 文档包含启动、超时和版本错误示例。

**验证：** 构建 Go 示例；运行协议集成测试驱动两个示例。

**依赖：** Task 39
**文件：** `examples/extensions/go-echo/main.go`、`examples/extensions/typescript-echo/index.ts`、`examples/extensions/README.md`、扩展集成测试
**规模：** M

### Task 41：迁移一个额外 Provider 的标准任务模板

**说明：** 每个剩余 Provider 单独复制本任务，不允许把多个独立 Provider 合并进一次实现。顺序由兼容矩阵决定。

**验收标准：**
- [ ] Provider 有独立模型目录、认证解析、能力声明和 fixture。
- [ ] 标准文本、工具、错误和取消契约测试通过。
- [ ] Provider 未通过测试前不会出现在公开模型列表。

**验证：** `go test ./internal/provider/<provider> ./internal/protocol/<protocol>`。

**依赖：** Task 18、Task 23、Task 25
**文件：** Provider adapter、adapter test、catalog、fixture，最多 5 个文件
**规模：** M（每个 Provider 单独执行）

Provider 待实例化队列（执行前复制 Task 41 模板，生成一张完整任务卡）：

- [ ] Azure OpenAI
- [ ] Amazon Bedrock
- [ ] Google Vertex
- [ ] GitHub Copilot
- [ ] Mistral
- [ ] OpenRouter
- [ ] Cloudflare AI Gateway
- [ ] Cloudflare Workers AI
- [ ] OpenCode
- [ ] ZAI
- [ ] MiniMax
- [ ] Moonshot/Kimi
- [ ] Groq
- [ ] Cerebras
- [ ] xAI
- [ ] DeepSeek
- [ ] Together
- [ ] Fireworks
- [ ] 兼容矩阵中的其他单个 Provider

### Task 42：实现 SQLite 可选索引后端

**说明：** 完成“实现 SQLite 可选索引后端”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] JSONL 仍是 Session 数据源，索引可以删除后重建。
- [ ] 并发索引和进程中断不损坏 Session。
- [ ] 禁用 SQLite 时所有核心功能仍可用。

**验证：** 先确认临时数据库测试 RED；再运行 `go test -race ./internal/session/index`。

**依赖：** Task 17；`modernc.org/sqlite` 版本需先审核
**文件：** `internal/session/index/sqlite.go`、`internal/session/index/sqlite_test.go`、`internal/session/index/index.go`、`go.mod`、`go.sum`
**规模：** M

### Task 43：重写 server 包

**说明：** 仅在 Spec 确认 server 属于范围后执行。

**验收标准：**
- [ ] server 使用与 CLI 相同的 Agent、Provider 和 Session 接口。
- [ ] 请求取消、认证边界和错误响应有集成测试。
- [ ] 不复制 Agent 状态机或 Provider 实现。

**验证：** 先确认 `httptest.Server` 测试 RED；再运行 `go test -race ./internal/server`。

**依赖：** Task 25、Task 27、Task 35
**文件：** `internal/server/server.go`、`internal/server/server_test.go`、`cmd/pi-server/main.go`、`cmd/pi-server/main_test.go`
**规模：** M

## 检查点 K：完整能力

- [ ] Task 38-43 中所有已批准范围完成。
- [ ] 每个公开 Provider 有独立契约测试。
- [ ] 扩展崩溃不会终止主进程。
- [ ] server/SQLite 的范围决策已落实。

## 阶段 7：迁移、加固和发布

### Task 44：实现迁移 dry-run 和备份

**说明：** 完成“实现迁移 dry-run 和备份”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] dry-run 报告配置、凭据和 Session 的兼容状态，不写文件。
- [ ] 正式迁移先备份，失败时原文件不变。
- [ ] 重复执行迁移保持幂等。

**验证：** 先确认临时 HOME fixture 测试 RED；再运行 `go test ./internal/migrate`。

**依赖：** Task 23、Task 35、Task 38
**文件：** `internal/migrate/migrate.go`、`internal/migrate/migrate_test.go`、`internal/cli/migrate.go`、`internal/cli/migrate_test.go`
**规模：** M

### Task 45：建立跨实现 parity runner

**说明：** 完成“建立跨实现 parity runner”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 同一 fixture 可驱动 TypeScript 和 Go，比较标准事件、Session 和退出码。
- [ ] 时间戳、随机 ID 等非确定字段通过明确规则归一化。
- [ ] 差异报告定位到事件和字段，不只返回 pass/fail。

**验证：** 先确认故意差异 fixture 测试 RED；再运行 parity runner 的自测命令。

**依赖：** Task 2、Task 27、Task 44
**文件：** `scripts/parity/main.go`、`scripts/parity/main_test.go`、`scripts/parity/normalize.go`、`scripts/parity/normalize_test.go`
**规模：** M

### Task 46：执行一个安全或故障加固目标

**说明：** 本任务是单目标模板。每次只选择下面队列中的一个目标，先复制成完整任务卡，再添加对应 fuzz/regression 测试和最小修复。

**验收标准：**
- [ ] 选定目标有一个先失败后通过的 fuzz 或 regression test。
- [ ] 修复不泄漏凭据、不降低现有边界，并通过目标包的 race test。
- [ ] 兼容矩阵或风险表记录该目标的结果。

**验证：** 运行目标包测试、目标 fuzz 时间窗口和 `go test -race ./path/to/package`。

**依赖：** Task 39、Task 44、Task 45
**文件：** 一个 fuzz/regression 文件和对应实现文件，单次不超过 5 个文件
**规模：** M（每个目标单独执行）

安全和故障待实例化队列：

- [ ] SSE 任意分块和超长事件
- [ ] 部分 JSON tool arguments
- [ ] Session 尾损坏和原子恢复
- [ ] ANSI/Unicode 宽度
- [ ] 凭据 redaction
- [ ] 工具路径边界
- [ ] 工具输出限制
- [ ] 子进程和扩展进程清理
- [ ] Agent/Session/credential 并发竞态

### Task 47：建立跨平台发布流水线

**说明：** 完成“建立跨平台发布流水线”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 构建 darwin/linux 的 amd64/arm64 二进制。
- [ ] 版本和 commit 通过 ldflags 注入，生成 SHA256SUMS。
- [ ] CI 先执行 vet、staticcheck、test 和 race gate。

**验证：** 本地 dry-run 构建四个目标；检查二进制 version 和 checksum。

**依赖：** Task 46；修改 CI 前需人工确认
**文件：** `.github/workflows/go-release.yml`、`scripts/build-go-release.sh`、`docs/releasing-go.md`
**规模：** M

### Task 48：执行发布 smoke test 和回滚演练

**说明：** 完成“执行发布 smoke test 和回滚演练”对应的单一行为切片，并以本任务的验收标准作为实现边界。

**验收标准：**
- [ ] 干净环境验证 help、version、print、RPC 和 interactive。
- [ ] Node.js 不存在时 Go 二进制仍可运行。
- [ ] 迁移失败和用户主动回滚均可恢复旧 TypeScript 版本和 Session。

**验证：** 按 `docs/releasing-go.md` 在临时目录和 tmux 中记录结果。

**依赖：** Task 47
**文件：** `docs/go-smoke-test.md`、`docs/go-migration.md`、`docs/go-rollback.md`
**规模：** M

### Task 49：批准 TypeScript 退役批次

**说明：** 这是纯审批任务，不修改或删除源码。批准后，每个删除批次必须另建不超过 5 个文件的完整任务卡。

**验收标准：**
- [ ] 兼容矩阵中所有必须项通过，剩余差异均有迁移文档。
- [ ] Go 发布经过约定过渡周期且无阻断性数据丢失问题。
- [ ] 首个删除批次、回滚点和验证命令获得明确批准。

**验证：** 发布门禁评审并检查首个删除批次任务卡；本任务不执行删除。

**依赖：** Task 48、人工批准
**文件：** `docs/compatibility-matrix.md`、首个删除批次任务卡
**规模：** S

## 最终检查点

- [ ] Spec、Plan 和兼容矩阵均为最新状态。
- [ ] 所有新增行为都有 RED -> GREEN -> REFACTOR 记录。
- [ ] `go test ./...`、`go test -race ./...`、`go vet ./...`、`staticcheck ./...` 全部通过。
- [ ] 自动 parity、tmux TUI 和干净环境 smoke test 通过。
- [ ] 迁移、备份和回滚经过实际演练。
- [ ] TypeScript 退役获得单独人工批准。
