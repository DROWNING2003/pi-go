# Pi 全量 Go 重写计划

> 前置规格：[spec.md](./spec.md)。必须先由项目负责人确认 Spec 的假设、兼容边界和成功标准，才能开始执行本计划。工作流固定为 `SPECIFY -> PLAN -> TASKS -> IMPLEMENT`，每个阶段都需要人工确认。

## 一、目标

将当前 Pi 项目完整重写为纯 Go 应用，最终产物是一个独立的 Go CLI，包含：

- 交互式终端 TUI
- Agent 状态机和工具调用
- 多 LLM Provider
- API Key、OAuth 和凭据存储
- Session、恢复、分支和上下文压缩
- `read`、`write`、`edit`、`bash` 工具
- print、JSONL/RPC 和 interactive 模式
- 命令、快捷键、主题、补全和图片显示
- 扩展机制和发布用的跨平台二进制

当前 TypeScript 实现暂时保留，作为行为基准和迁移参照。只有 Go 版本通过兼容性、安全性和发布验收后，才退役 TypeScript 运行时。

当前项目规模约为 903 个 TypeScript 源文件、335 个测试文件、21 万行代码。因此这不是简单的语言翻译，而是一次保持用户行为的系统重构。

## 二、兼容目标

### 必须保持的行为

- 交互式启动、输入、多行编辑、流式输出和中断
- 文本、thinking、图片、tool call、tool result 和错误事件
- 工具调用、并行/串行执行、工具失败和工具取消
- `read`、`write`、`edit`、`bash` 的主要行为
- Session JSONL、恢复、继续、fork、tree、compaction 和导出
- API Key、环境变量、凭据存储、OAuth 登录和 token 刷新
- 模型选择、Provider 切换、token/费用统计
- `/login`、`/logout`、`/model`、`/resume`、`/tree`、`/compact`、`/settings`、`/export`、`/quit` 等命令
- print、JSONL/RPC 和 interactive 三种运行模式
- 终端 resize、快捷键、粘贴、补全、主题和退出时的终端清理

### 可以变化的部分

- TypeScript 源码级 API
- 内部包名和模块结构
- ANSI 转义序列的具体实现
- 未公开、已废弃或没有测试覆盖的内部行为
- TypeScript 扩展是否直接兼容

任何有意删除的能力，都必须先从兼容矩阵中标记为“明确变更”，不能在重写过程中默认丢弃。

## 三、技术栈

### 3.1 运行时和语言

| 领域 | 选择 | 用途 |
|---|---|---|
| 主语言 | Go 1.26.1 | CLI、Agent、Provider、Session、TUI 全部使用 Go；`go.mod` 固定 toolchain |
| 模块管理 | Go Modules | 依赖和版本管理 |
| 可执行文件 | `cmd/pi/main.go` | 生成 `pi` 命令 |
| 最低平台 | macOS、Linux；Windows 后续支持 | 第一阶段优先保证 Unix 终端行为 |
| 并发模型 | goroutine、channel、`context.Context` | 流式响应、工具执行、取消和事件分发 |

Go 版本固定为 1.26.1，并在 `go.mod` 中声明 toolchain。CI、开发环境和发布构建使用同一版本，避免工具链差异造成 TUI 或标准库行为变化。第三方依赖必须先评审，再以明确版本写入 `go.mod`/`go.sum`。

### 3.2 TUI

| 组件 | 技术选择 | 用途 |
|---|---|---|
| TUI 框架 | Charmbracelet Bubble Tea | Model/Update/View 事件循环和状态管理 |
| 样式 | Charmbracelet Lip Gloss | 颜色、边框、布局和主题 |
| 终端底层 | `golang.org/x/term` | raw mode、终端尺寸和终端状态恢复 |
| 键盘解析 | Bubble Tea 输入事件 + 自定义 key map | Ctrl、Alt、Shift、方向键和 Kitty keyboard protocol |
| 文本宽度 | `github.com/mattn/go-runewidth` | 中英文、emoji、组合字符和 ANSI 宽度计算 |
| Markdown | `github.com/charmbracelet/glamour` 或独立 renderer | Markdown、代码块和主题渲染 |
| 代码高亮 | `github.com/alecthomas/chroma` | 工具输出和 Markdown 代码块高亮 |
| 图片 | Kitty/iTerm2 协议适配器 | 支持图片终端；不支持时显示文本回退 |

不直接从零实现终端渲染器。先使用 Bubble Tea 建立应用状态和交互流，只有现有 TUI 行为无法满足时，才在明确边界内补充底层 ANSI 代码。

### 3.3 HTTP、Provider 和流式协议

| 组件 | 技术选择 | 用途 |
|---|---|---|
| HTTP | Go 标准库 `net/http` | Provider 请求、代理、超时和连接复用 |
| 请求取消 | `context.Context` | 用户 Escape、超时和进程退出时取消请求 |
| SSE | `github.com/r3labs/sse/v2` 或受控的标准库解析器 | Anthropic、OpenAI 等 SSE 流 |
| JSON | `encoding/json` | Provider payload、事件、Session 和 RPC |
| JSON Schema | `github.com/santhosh-tekuri/jsonschema/v6` | 工具参数验证 |
| 重试 | 自定义有限重试策略 | 只对可重试的网络错误执行，禁止无条件重试请求 |
| Provider 抽象 | 自定义 Go interface | 统一模型目录、鉴权、stream 和 complete |

Provider 分成两层：

```text
Provider 适配器：模型目录、认证、base URL、能力声明
协议客户端：OpenAI Completions、OpenAI Responses、Anthropic、Google 等 wire protocol
```

OpenAI-compatible Provider 共用协议客户端，但每个 Provider 保留独立的模型元数据、鉴权和兼容性配置。

第一批实现顺序：

1. OpenAI-compatible Completions
2. OpenAI Responses
3. Anthropic Messages
4. Google Generative AI
5. OpenRouter 和本地兼容服务
6. 其余 Provider 按实际用户需求迁移

### 3.4 Agent Runtime

| 组件 | 技术选择 | 用途 |
|---|---|---|
| Agent 状态 | 自定义不可变快照/受控可变状态 | 管理消息、模型、工具和运行状态 |
| 事件流 | Go typed event + channel | 统一驱动 TUI、print 和 RPC |
| 流式累积 | 显式状态机 | 累积 text、thinking、tool call 和错误事件 |
| 取消 | `context.WithCancel` | 取消请求和工具执行 |
| 工具并发 | goroutine + `errgroup` | 并行工具调用和统一错误收集 |
| 事件同步 | 单一 event loop + 有序 transcript writer | 保证 UI 事件和 Session 写入顺序 |
| Schema | JSON Schema | 工具参数校验和 RPC 工具描述 |

Agent 必须与 TUI 解耦。TUI 只能订阅 Agent 事件并发送用户动作，不能直接修改 Agent 内部状态。

### 3.5 Session、配置和凭据

| 组件 | 技术选择 | 用途 |
|---|---|---|
| Session 格式 | 继续支持现有 JSONL | 迁移期间直接读取旧会话 |
| Session 写入 | 标准库 `os` + 临时文件 rename | 原子写入，防止进程中断损坏会话 |
| Session 索引 | 文件系统扫描 + 可选 SQLite 索引 | 第一阶段减少迁移复杂度，后续优化检索 |
| SQLite | `modernc.org/sqlite` | 纯 Go、无需 CGO；用于索引、缓存或可选后端 |
| 配置 | `encoding/json` | 全局和项目级 JSON 配置 |
| 路径 | `os.UserConfigDir`、`os.UserHomeDir` | 跨平台配置和 session 目录 |
| 凭据 | 文件存储，权限 `0600` | API Key 和 OAuth token 的持久化 |
| 文件锁 | `github.com/gofrs/flock` | 防止多进程同时刷新凭据或写 Session |
| 密钥日志 | 自定义 redaction 层 | 禁止 API Key、OAuth token 出现在日志和错误中 |

现有 Session JSONL 先作为兼容格式长期支持。任何新格式必须提供导入、导出、dry-run 和失败恢复机制。

### 3.6 CLI 和 RPC

| 组件 | 技术选择 | 用途 |
|---|---|---|
| CLI 参数 | `github.com/spf13/cobra` + `github.com/spf13/pflag` | 子命令、短参数和帮助信息 |
| JSONL/RPC | 标准输入输出 + `encoding/json` | 进程集成和测试驱动 |
| 日志 | `log/slog` | 结构化日志；默认不污染 stdout |
| 配置目录 | `PI_CODING_AGENT_DIR` 等环境变量 | 保持已有环境变量兼容 |
| 退出码 | 自定义稳定退出码表 | 区分参数、认证、网络、工具和内部错误 |

stdout 只输出产品协议内容；诊断日志写入 stderr 或显式日志文件，确保 RPC 不被日志污染。

### 3.7 测试和质量工具

| 领域 | 技术选择 | 用途 |
|---|---|---|
| 单元测试 | Go 标准库 `testing` | 纯函数、解析器和状态机 |
| 断言 | `github.com/stretchr/testify` | 提升测试可读性 |
| HTTP 测试 | `httptest.Server` | Provider fixture 和错误场景 |
| TUI 测试 | Bubble Tea model 测试 + golden 文件 | 键盘输入、状态转移和渲染结果 |
| 模糊测试 | Go 原生 fuzzing | SSE、部分 JSON、ANSI 宽度和 Session 恢复 |
| 并发检测 | `go test -race` | 凭据刷新、事件流和工具并发 |
| 静态检查 | `go vet`、`staticcheck`、`golangci-lint` | 代码质量和常见错误 |
| 覆盖率 | `go test -coverprofile` | 核心状态机和安全代码的覆盖率门禁 |
| 集成测试 | faux provider + fixture server | 不依赖真实 API Key 和付费服务 |
| 交互测试 | tmux | 启动、输入、resize、Escape、恢复和退出 |

不在自动化测试中调用真实 Provider，不把真实 API Key 写入 fixture、日志或 CI。真实 Provider 只用于发布前受控的人工 smoke test。

### 3.8 开发方法：测试驱动开发

整个 Go 重写采用严格的 RED-GREEN-REFACTOR 循环：

```text
RED：先写一个描述外部行为的测试，并确认它因为功能尚未实现而失败
GREEN：只实现让当前测试通过的最小代码
REFACTOR：清理命名、重复和边界，确认测试仍然通过
```

每个任务必须遵守以下规则：

1. 在生产代码之前提交对应测试或 fixture。
2. 记录 RED 阶段的失败原因，证明测试确实覆盖待实现行为。
3. GREEN 阶段只实现当前验收行为，不提前加入未测试的抽象。
4. REFACTOR 后运行该包测试；跨包契约变化再运行相关集成测试。
5. Bug 必须先添加能够稳定复现问题的回归测试。
6. 不允许通过跳过、删除或弱化断言让测试变绿。
7. 测试断言输入、输出、状态和持久化结果，不绑定内部函数调用顺序。
8. 优先使用真实实现，其次是内存 fake；只在外部网络等不可控边界使用 stub/mock。

测试分层：

| 层级 | 占比目标 | 内容 | 资源约束 |
|---|---:|---|---|
| 单元测试 | 约 80% | parser、serializer、状态机、参数校验、文本宽度 | 单进程、无网络、无数据库 |
| 集成测试 | 约 15% | fixture Provider、Session 文件、工具子进程、RPC | 只允许 localhost 和临时目录 |
| E2E 测试 | 约 5% | 完整 CLI/TUI 用户流程和发布 smoke test | 只覆盖关键路径 |

各模块的 TDD 顺序：

- Provider：先写 HTTP fixture 和期望的标准事件，再实现 payload 和流解析。
- Agent：先写完整事件序列和最终 transcript，再实现状态转移和工具调度。
- 工具：先写临时目录/子进程场景，再实现成功、失败、超时和取消。
- Session：先准备旧版 JSONL fixture，再实现读取、写入、恢复和迁移。
- TUI：先输入 Bubble Tea message 并断言 model 状态，再实现 Update；最后增加小型 golden frame。
- Bug 修复：先增加失败的 regression test，再修改实现。

每个任务的完成标准：

- RED 失败已确认，且失败原因是缺少目标行为，而不是测试写错或环境失败。
- GREEN 测试通过。
- REFACTOR 后测试仍通过。
- 没有 skip、禁用测试或降低断言强度。
- 新行为有测试名称清楚描述其预期结果。
- 涉及并发的代码通过 `go test -race`。

### 3.9 扩展机制

默认选择：**子进程 JSON-RPC 扩展协议**，而不是 Go plugin。

理由：

- Go plugin 对 Go 版本、操作系统和构建环境有较强限制。
- 子进程协议可以支持 Go、TypeScript、Python 等扩展语言。
- 扩展崩溃可以被主进程隔离和重启。
- 协议可以单独版本化，不绑定 Pi 内部 Go 类型。

扩展协议至少包含：

- `initialize`
- tool 注册和调用
- command 注册和调用
- event 订阅
- UI notification
- 超时、取消和错误返回
- 协议版本和能力协商

是否必须兼容当前 TypeScript 扩展，需要在开始 Phase 5 前明确。如果必须兼容，增加 Node 子进程 adapter；不承诺直接加载 `.ts` 模块。

### 3.10 构建和发布

| 组件 | 技术选择 | 用途 |
|---|---|---|
| 构建 | `go build` | 生成静态或尽量少依赖的二进制 |
| 发布自动化 | GitHub Actions | 多平台构建、测试和发布 |
| 多平台构建 | Go `GOOS`/`GOARCH` | darwin-arm64、darwin-amd64、linux-amd64、linux-arm64 |
| 校验 | SHA256SUMS | 发布包完整性校验 |
| 版本 | ldflags 注入版本和 commit | `pi --version` 和诊断信息 |
| 安装 | shell installer + 直接下载 | 保持 CLI 安装体验 |
| 容器 | 多阶段 Dockerfile | 可选的隔离运行方式 |

第一阶段不引入 CGO。这样可以降低跨平台构建和发布复杂度；SQLite 优先使用 `modernc.org/sqlite`。

## 四、目录结构

建议的 Go 项目结构：

```text
cmd/pi/main.go
internal/
  agent/       Agent loop、队列、工具调度、事件
  cli/         参数解析、退出码、启动流程
  config/      settings、context、trust、环境变量
  export/      HTML/JSONL 导出
  model/       消息、内容块、模型、usage、错误
  provider/    Provider interface、模型目录、认证
  protocol/    OpenAI、Anthropic、Google、SSE 客户端
  rpc/         JSONL/RPC server 和 client
  session/     JSONL、索引、分支、compaction
  tool/        read、write、edit、bash、schema
  tui/         Bubble Tea model、页面、组件、主题
  terminal/    raw mode、resize、图片和 ANSI 辅助
  extension/   子进程 JSON-RPC 扩展协议
pkg/           只有确实需要对外复用的稳定包
scripts/       构建、fixture、发布和迁移脚本
testdata/      Provider、Session、TUI golden fixtures
```

## 五、任务依赖和执行方式

详细任务卡位于 [tasks.md](./tasks.md)，每项包含验收标准、验证命令、依赖、预计文件和规模。实现时以任务卡为准，本计划只描述阶段目标和依赖关系。

### 5.1 主依赖图

```text
Task 0 Spec 审批
  -> Task 1-2 兼容矩阵和 fixture
  -> Task 3 Go 骨架
  -> Task 4 初始 CI/CD 门禁
  -> Task 5-7 消息/事件契约、faux Provider
  -> Task 8-10 print + Session + read 工具纵向切片
  -> Task 11-17 四个工具、队列、取消、Session tree
  -> Task 18 HTTP/SSE 公共客户端
       -> Task 19-22 首批 Provider 协议
       -> Task 23-25 凭据、OAuth、模型目录
  -> Task 26-27 RPC Agent
  -> Task 28-31 TUI 外壳、编辑器、消息、Agent 对话
  -> Task 32-37 命令、Session UI、compaction、渲染、图片
  -> Task 38-43 扩展、剩余 Provider、SQLite、server
  -> Task 44-46 迁移、parity、安全和故障加固
  -> Task 47-48 发布和回滚
  -> Task 49 审批首个 TypeScript 退役批次
```

### 5.2 纵向切片

优先交付以下可运行切片，不采用“先写完所有 Provider，再写 Agent，最后接 UI”的横向方式：

1. faux Provider + print 文本回答 + 标准事件。
2. Session 写入/恢复 + read 工具 + Agent 二次调用。
3. 四个内置工具 + abort/continue + steering/follow-up。
4. 一个 API Key Provider + 一个 OAuth Provider + 模型选择。
5. RPC 完整对话。
6. TUI prompt + 流式消息 + 工具 + Session。
7. 命令、主题、扩展、剩余 Provider 和迁移。

每个切片结束时程序都必须可构建、可测试，并提供一个新增的完整用户行为。

### 5.3 检查点

| 检查点 | Tasks | 审批内容 |
|---|---|---|
| A | 0-2 | Spec、兼容矩阵和 fixture |
| B | 3-7 | message/event schema v1 |
| C | 8-10 | 最小 Agent 纵向切片 |
| D | 11-14 | 四个内置工具和并发调度 |
| E | 15-17 | Agent 队列、取消和 Session tree |
| F | 18-21 | HTTP/SSE、OpenAI、Anthropic 协议 |
| G | 22-25 | Google、凭据、OAuth 和模型目录 |
| H | 26-28 | RPC 和 TUI 外壳 |
| I | 29-31 | 可完成对话的 TUI |
| J | 32-37 | 核心用户功能对齐 |
| K | 38-43 | 扩展、剩余 Provider 和外围包 |
| Final | 44-49 | 迁移、parity、发布和首个退役批次审批 |

每个检查点必须完成对应测试命令和人工审查，不允许在检查点失败时继续堆叠后续实现。

### 5.4 并行开发边界

冻结 message/event schema 后可以并行：

- Task 11、12、13：write、edit、bash 工具。
- Task 19、20、21、22：不同 wire protocol。
- Task 23：凭据存储可与 Provider 协议并行。
- Task 26：RPC framing 可与早期 TUI shell 并行，但 Agent RPC 连接必须等待 Task 27。
- Task 32、34、36、37：补全、Session UI、Markdown/主题、图片协议。
- Task 41 模板：每个 Provider adapter 独立实例化执行。

必须串行：

- Spec -> 兼容矩阵 -> 公共 schema。
- 公共 schema -> faux Provider -> 第一个 Agent 切片。
- Agent 事件和 Session 格式变更。
- OAuth store -> OAuth Provider。
- 扩展协议 -> 扩展宿主 -> 扩展示例。
- 迁移 -> parity -> 发布 -> TypeScript 退役。

需要协调：

- 任何共享 schema、CLI flag、RPC version、Session format 或扩展协议变化。
- `go.mod`/`go.sum` 和 CI 修改。
- 同一 package 中的并行任务。

## 六、实施阶段

### 阶段 0：冻结行为基线

- 记录 CLI 参数、slash commands、快捷键、Session 路径、环境变量、Provider、认证、工具和输出模式。
- 使用 faux provider 生成文本、thinking、tool call、abort 和错误事件 fixture。
- 保存代表性 Session JSONL、配置和 TUI 状态快照。
- 建立兼容矩阵，标记“必须兼容”“第一版可暂缺”“明确改变”。

验收：每项必须兼容的行为都有 fixture、自动化测试或明确的手工测试步骤。

### 阶段 1：Go 基础和公共协议

- 先为 model/message JSON round-trip、未知字段和版本错误编写失败测试。
- 创建 `go.mod`、`cmd/pi`、CI、格式化和静态检查。
- 定义 model、message、content block、usage、error 和 stop reason。
- 定义 Agent lifecycle event 和 Provider stream event。
- 定义 JSON 序列化规则和版本号。
- 实现配置、环境变量、`AGENTS.md`/`CLAUDE.md` 和 project trust。

验收：Go 二进制可以执行 `pi --help`，所有公共 fixture 可以 round-trip。

### 阶段 2：HTTP、Provider 和认证

- 每个 Provider 先建立 `httptest.Server` fixture，并确认标准事件契约测试失败。
- 实现 HTTP 超时、取消、SSE、JSON streaming、错误和 payload debug。
- 实现 OpenAI-compatible Completions。
- 实现 OpenAI Responses。
- 实现 Anthropic Messages。
- 实现 Google Generative AI。
- 按 Provider 优先级迁移其他服务。
- 实现 credential store、环境变量 fallback、logout 和 redaction。
- 实现 OAuth 登录、token 刷新、并发刷新和失败恢复。

验收：至少一个 API Key Provider 和一个 OAuth Provider 可以从 Go 端完成真实流程；测试本身仍使用 fixture server。

### 阶段 3：Agent 和工具

- 先用 fake Provider 编写 Agent 事件顺序、最终 transcript 和取消行为的失败测试。
- 先用临时目录和受控子进程编写工具成功、失败、超时和取消测试。
- 实现工具描述、JSON Schema 参数校验和工具结果。
- 实现 `read`、`write`、`edit`、`bash`。
- 实现 Agent loop、上下文转换和流式消息累积。
- 实现工具 preflight、hook、并行/串行执行和 terminate。
- 实现 abort、continue、steering、follow-up 和 idle settlement。
- 实现 Session JSONL 读取、原子写入、损坏恢复、resume、fork 和 tree。
- 实现 compaction、HTML 导出和 JSONL 导出。

验收：文本、工具调用、工具失败、中断、steering、follow-up 和恢复场景的事件顺序与基线一致。

### 阶段 4：CLI、RPC 和 TUI

- 先为 CLI 退出码、RPC JSONL framing 和 Bubble Tea model 状态转移编写失败测试。
- 实现 print mode。
- 实现 JSONL/RPC mode。
- 使用 Bubble Tea 建立主交互页面。
- 实现终端 raw mode、resize、滚动、流式消息、tool progress、loader 和 overlay。
- 实现多行编辑、粘贴、光标、Escape、中断和 autocomplete。
- 实现 slash commands、model selector、login、resume、tree、settings、compact、export。
- 使用 Lip Gloss 实现主题；使用 Glamour/Chroma 实现 Markdown 和代码高亮。
- 实现 Kitty/iTerm2 图片协议和不支持图片终端时的文本回退。

验收：用户可以启动 Go 版、登录、选择模型、发送请求、执行工具、取消请求、恢复 Session 并继续对话。

### 阶段 5：扩展和迁移

- 冻结并实现版本化子进程 JSON-RPC 扩展协议。
- 提供一个 Go 扩展示例和一个 TypeScript/Python 扩展示例。
- 实现扩展超时、取消、崩溃隔离和错误显示。
- 实现配置和 Session dry-run 迁移。
- 对旧版和 Go 版执行相同 fixture，比较事件、Session 和退出码。
- 记录所有不兼容项和迁移方式。

验收：扩展协议有版本号、能力协商和失败隔离；迁移失败不会覆盖原始 Session 或凭据。

### 阶段 6：发布和退役

- 执行全量 parity、security、fuzz、race、性能和故障测试。
- 测试网络断开、Provider 返回脏数据、终端中断、Session 损坏、大输出和并发工具。
- 发布 macOS/Linux 多架构二进制、checksum、安装脚本和回滚说明。
- 使用干净机器验证 `--help`、`--version`、print、RPC 和 interactive。
- 保留 TypeScript 版一个明确的过渡周期。
- 达到迁移指标后再移除 TypeScript 运行时。

## 七、关键验收标准

- 每项新增行为都有先失败后通过的测试记录。
- 所有“必须兼容”的行为通过自动测试或记录过的手工验证。
- Provider 不会在流式错误、取消或部分 JSON 时静默丢失状态。
- API Key 和 OAuth token 不出现在日志、错误、Session 或 RPC 输出中。
- 工具执行支持取消、超时、错误和并发控制。
- TUI 不会破坏终端 raw mode、光标状态或颜色状态。
- 旧 Session 可以被 Go 版读取、恢复、分支和继续。
- JSONL/RPC stdout 不混入日志。
- Go 二进制可在支持平台上独立运行，不依赖 Node.js。
- 迁移和回滚流程在真实代表性 Session 上验证通过。

## 八、风险和应对

| 风险 | 影响 | 应对 |
|---|---|---|
| 兼容目标持续增加 | 高 | 先冻结兼容矩阵，新增目标必须单独确认 |
| Provider 之间细节差异很大 | 高 | 统一事件模型，Provider 使用 fixture server 做契约测试 |
| OAuth 刷新造成凭据丢失 | 高 | 文件锁、原子写入、并发刷新测试和日志脱敏 |
| TUI 在不同终端表现不一致 | 高 | Bubble Tea + golden 测试 + tmux 手工测试 |
| TypeScript 扩展无法直接运行 | 高 | 使用子进程 JSON-RPC；需要时增加 Node adapter |
| Session 迁移导致历史损坏 | 高 | 只读导入、dry-run、备份、原子写入和回滚 |
| 重写周期过长 | 高 | 按“一个 Provider + 四个工具 + Session + TUI”切纵向可运行版本 |
| 并行开发产生协议不一致 | 中 | 先冻结 model/event/RPC schema，再拆分 Provider 和 UI 任务 |

## 九、需要先确认的问题

1. 第一版必须支持哪些 Provider？建议先支持 OpenAI、Anthropic、Google 和 OpenAI-compatible 服务。
2. 是否必须直接兼容现有 TypeScript 扩展？建议改为子进程 JSON-RPC，不直接加载 `.ts` 模块。
3. 现有 Session JSONL 是否需要永久兼容？建议永久只读兼容，写入格式短期保持一致。
4. 第一版是否只支持 macOS/Linux？建议是，Windows 放到第二阶段。
5. 是否需要保留 server 包，还是只重写本地 CLI？
6. Go 版是否需要实现所有现有 Provider，还是先按用户实际使用情况分批迁移？

## 十、第一步

不要先写 Go 代码，也不要先重写 TUI。第一步应当完成以下内容：

1. 审阅并批准 `docs/spec.md`，解决或明确延后其中的 Open Questions。
2. 根据批准后的 Spec 更新本计划和 `docs/tasks.md`。

Spec 获得批准后，再执行以下内容：

1. 确认兼容矩阵和 Provider 优先级。
2. 创建 Go module 和 `cmd/pi`。
3. 定义 model/message/event JSON schema。
4. 用 faux provider 和 fixture server 跑通“用户 prompt -> 流式回复 -> Session 写入”。
5. 再接入 `read` 工具，形成第一个完整 Agent 纵向切片。
