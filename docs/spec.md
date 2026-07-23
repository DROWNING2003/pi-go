# Spec：Pi 纯 Go 重写

## 1. 假设

以下假设需要在实现前由项目负责人确认：

1. 目标是最终用纯 Go 运行时替换当前 TypeScript CLI，而不是只重写 TUI。
2. 第一版正式支持 macOS 和 Linux，Windows 暂不作为首发平台。
3. 现有 Session JSONL 是重要的用户数据，Go 版至少要能读取、恢复和继续旧 Session。
4. 第一版优先支持 OpenAI、Anthropic、Google 和 OpenAI-compatible 服务，其他 Provider 分批迁移。
5. 不要求 Go 直接加载现有 TypeScript 模块；扩展改用版本化的子进程 JSON-RPC 协议。
6. 当前 TypeScript 实现在迁移期间继续作为行为基准，不在 Go 版尚未通过验收时删除。
7. 工具默认仍然拥有当前 Pi 的本地文件和进程权限；沙箱不属于第一版重写目标，但必须保留风险说明和取消机制。

如果以上任一假设不成立，应先更新本 Spec 和 `docs/plan.md`，再开始实现。

## 2. Objective

### 2.1 Core-first 产品目标

第一阶段先交付可独立测试和复用的 Core，不先实现完整 TUI、扩展或 server。Core 的完成链路为：

```text
packages/ai -> packages/agent -> packages/storage
```

`packages/coding-agent` 只作为薄组装入口保留；TUI、扩展和 server 进入 Core checkpoint 之后的独立阶段。

### 2.2 产品目标

构建一个独立的纯 Go `pi` CLI，保留当前 Pi 的核心用户工作流：

```text
启动 pi
  -> 选择/登录 Provider
  -> 输入 prompt
  -> 接收流式文本或 thinking
  -> 执行 read/write/edit/bash 工具
  -> 中断、steering 或 follow-up
  -> 保存 Session
  -> 下次恢复并继续
```

### 2.3 用户

主要用户是需要在终端中使用 coding agent 的开发者。用户关心：

- 启动速度和跨平台独立运行
- 流式输出、工具调用和中断是否可靠
- 旧 Session 是否安全可恢复
- Provider、模型和凭据是否易于切换
- TUI 是否能适应不同终端尺寸和中英文输入
- 出错时是否能获得可诊断的信息，而不是静默丢失状态

### 2.4 最终必须支持的功能

- Interactive TUI、print mode、JSONL/RPC mode
- 多行编辑、粘贴、文件引用、slash command 补全
- 文本、thinking、图片、tool call、tool result 和错误流式事件
- `read`、`write`、`edit`、`bash` 工具
- 工具参数校验、超时、取消、错误、并行/串行模式
- abort、continue、steering、follow-up
- Session JSONL、resume、fork、tree、compaction、HTML/JSONL export
- 模型切换、thinking level、token usage 和 cost
- API Key、环境变量、持久化凭据、logout 和 OAuth（按 Provider 优先级）
- `/login`、`/logout`、`/model`、`/resume`、`/new`、`/session`、`/tree`、`/fork`、`/compact`、`/settings`、`/export`、`/quit`
- 主题、快捷键、终端 resize、终端状态恢复和图片终端回退
- 子进程 JSON-RPC 扩展，包含 tool、command、event 和能力协商

## 3. Tech Stack

### 3.1 核心

- **语言和工具链**：Go 1.26.1，使用 `go 1.26.1`/`toolchain go1.26.1` 固定开发和 CI 版本
- **依赖管理**：Go Modules；第三方依赖在评审后以 `go.mod`/`go.sum` 锁定明确版本，禁止提交未审核的 `@latest` 解析结果
- **入口**：`packages/coding-agent/cmd/pi/main.go`
- **并发**：goroutine、channel、`context.Context`
- **日志**：标准库 `log/slog`
- **JSON/JSONL**：标准库 `encoding/json`

### 3.2 Core package 技术边界

- **AI**：`packages/ai/model`、`packages/ai/provider`、`packages/ai/protocol`
- **Agent**：`packages/agent/loop`、`packages/agent/event`、`packages/agent/tool`、`packages/agent/queue`
- **Storage**：`packages/storage/session`、`packages/storage/config`
- **组装层**：`packages/coding-agent`
- **后续阶段**：`packages/tui`、`packages/extension`、`packages/server`

Core 包不能依赖 TUI、扩展或 server；TUI 只能订阅 Agent event。

### 3.3 TUI

- **TUI 框架**：Bubble Tea
- **样式和主题**：Lip Gloss
- **终端 raw mode/尺寸**：`golang.org/x/term`
- **终端字符宽度**：`github.com/mattn/go-runewidth`
- **Markdown**：Glamour 或受控的内部 renderer
- **代码高亮**：Chroma
- **图片**：Kitty/iTerm2 graphics protocol 适配器

不从零实现完整的终端渲染框架。Bubble Tea 负责状态循环，Lip Gloss 负责样式；只在协议兼容性测试证明必要时补充底层 ANSI 处理。

### 3.4 Provider 和网络

- **HTTP**：标准库 `net/http`
- **取消和超时**：`context.Context`
- **流式响应**：SSE/JSON stream 解析器
- **工具参数**：JSON Schema 校验
- **Provider 分层**：Provider 适配器 + 共享 wire protocol 客户端
- **协议首批顺序**：OpenAI-compatible Completions、OpenAI Responses、Anthropic Messages、Google Generative AI

### 3.5 存储和配置

- **Session**：继续读取现有 JSONL；临时文件 + rename 原子写入
- **配置**：JSON 文件，兼容全局/项目配置和现有环境变量
- **凭据**：`0600` 文件存储、文件锁、日志脱敏
- **可选索引**：`modernc.org/sqlite`，第一版不强制依赖数据库
- **路径**：`os.UserConfigDir`、`os.UserHomeDir` 和项目目录

### 3.6 CLI、扩展和构建

- **CLI**：Cobra + PFlag
- **RPC**：stdin/stdout 上的版本化 JSONL
- **扩展**：子进程 JSON-RPC；不使用 Go plugin
- **CI**：GitHub Actions 在首个 Go 骨架后立即建立，所有后续实现必须通过质量门禁
- **构建**：`go build`、`GOOS`/`GOARCH`、GitHub Actions
- **CD**：`v*` tag 触发受控 GitHub Release，发布前重复测试并生成 SHA256SUMS
- **发布**：跨平台二进制、版本 ldflags、SHA256SUMS、安装脚本

## 4. Commands

以下命令从 Go 项目根目录执行。

### 开发

```bash
# 格式化
gofmt -w ./packages

# 静态检查
go vet ./...
staticcheck ./...

# 运行全部单元和集成测试
go test ./...

# 运行竞态检测
go test -race ./...

# 运行测试并生成覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# 构建本地二进制
go build -trimpath -ldflags "-X main.version=dev" -o ./bin/pi ./packages/coding-agent/cmd/pi

# 启动当前开发版 CLI
./bin/pi
```

项目应在实现阶段提供等价的 Makefile 或脚本命令，但底层命令必须保持可直接执行。

### 运行模式

```bash
# 交互模式
./bin/pi

# 继续最近 Session
./bin/pi --continue

# 选择历史 Session
./bin/pi --resume

# print 模式
./bin/pi --print "Summarize this repository"

# JSONL/RPC 模式
./bin/pi --mode rpc

# 指定 Provider 和模型
./bin/pi --provider anthropic --model claude-sonnet-4-6

# 离线运行
./bin/pi --offline

# 查看帮助和版本
./bin/pi --help
./bin/pi --version
```

### 发布验证

```bash
GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/pi-darwin-arm64 ./packages/coding-agent/cmd/pi
GOOS=darwin GOARCH=amd64 go build -trimpath -o dist/pi-darwin-amd64 ./packages/coding-agent/cmd/pi
GOOS=linux GOARCH=amd64 go build -trimpath -o dist/pi-linux-amd64 ./packages/coding-agent/cmd/pi
GOOS=linux GOARCH=arm64 go build -trimpath -o dist/pi-linux-arm64 ./packages/coding-agent/cmd/pi
shasum -a 256 dist/*
```

## 5. Project Structure

```text
packages/ai/model/           标准消息、内容块、usage、错误
packages/ai/provider/        Provider interface、模型目录、认证
packages/ai/protocol/        OpenAI、Anthropic、Google、SSE 客户端
packages/agent/loop/         Agent loop 和状态机
packages/agent/event/        Agent lifecycle event
packages/agent/tool/         read、write、edit、bash 和 schema
packages/agent/queue/        abort、continue、steering、follow-up
packages/storage/session/    JSONL、恢复、分支和原子写入
packages/storage/config/     settings、context files、trust、凭据
packages/coding-agent/       CLI 组装层
packages/tui/                后续 Bubble Tea TUI
packages/extension/          后续 JSON-RPC 扩展
packages/server/             后续 server 模式
testdata/model/              Message 和 event fixture
testdata/provider/           Provider stream fixture
testdata/session/             旧版、分支和损坏 Session fixture
scripts/                     parity、迁移、构建和发布
docs/                        Spec、Plan、Tasks 和兼容矩阵
```

依赖方向必须保持单向：

```text
model
  -> protocol/provider
  -> agent/tool/session
  -> cli/rpc/tui
```

`packages/tui` 不得直接调用 Provider；`packages/ai/provider` 不得依赖 TUI；CLI 只负责组装 Core 依赖和启动模式。

## 6. Code Style

### 6.1 Go 示例

```go
package session

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

// Load reads a session without mutating the source file.
func Load(ctx context.Context, path string) (Session, error) {
    if err := ctx.Err(); err != nil {
        return Session{}, err
    }

    data, err := os.ReadFile(filepath.Clean(path))
    if err != nil {
        return Session{}, fmt.Errorf("read session: %w", err)
    }

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return Session{}, fmt.Errorf("decode session: %w", err)
    }
    return session, nil
}
```

### 6.2 约定

- 遵循 `gofmt`；导出类型和函数使用 GoDoc 注释。
- 包名使用小写单词，不使用下划线。
- 错误使用 `%w` 包装上下文；调用方使用 `errors.Is`/`errors.As` 判断类型。
- 外部输入必须在边界校验；路径、Provider payload、RPC 消息和工具参数不能直接信任。
- API Key、OAuth token、cookie 和完整认证 header 不得写入日志、Session 或错误文本。
- Context 通过函数参数传递，不存储在长生命周期结构体中。
- Agent 事件使用明确的 typed struct，不使用 `map[string]any` 作为核心内部协议。
- 代码先写测试，再写最小实现；测试名称描述用户可观察行为。
- 测试不依赖真实网络、真实 API Key 或本机已有 Session。

## 7. Testing Strategy

### 7.1 TDD 规则

每个行为遵循：

```text
RED -> GREEN -> REFACTOR
```

- RED：先写测试并确认由于目标行为不存在而失败。
- GREEN：写最小实现让测试通过。
- REFACTOR：清理设计，测试必须保持通过。
- Bug：先写失败的 regression test，再修复。
- 不允许 skip、删除测试或降低断言来消除失败。

### 7.2 测试分层

| 层级 | 目标比例 | 测试内容 |
|---|---:|---|
| 单元 | 约 80% | parser、serializer、状态机、参数校验、文本宽度 |
| 集成 | 约 15% | `httptest.Server` Provider、临时目录 Session、工具进程、RPC |
| E2E | 约 5% | 关键 CLI/TUI 流程、发布 smoke test |

### 7.3 必须覆盖的行为

- JSONL message/event round-trip 和未知字段处理
- SSE 分块、部分 JSON tool arguments、Provider 错误和取消
- Provider 能力声明、thinking、图片、tool call 和 usage
- Agent 事件顺序、并发工具、工具失败、abort、steering、follow-up
- `read`、`write`、`edit`、`bash` 的路径、输出、权限、超时和取消
- Session 读取、原子写入、损坏恢复、resume、fork、tree 和 compaction
- 凭据读写、文件权限、OAuth refresh、并发 refresh 和日志脱敏
- RPC framing、stdout 不混入日志、稳定退出码
- Bubble Tea 输入、resize、焦点、粘贴、滚动、主题和终端清理
- 旧版 TypeScript fixture 与 Go 输出的兼容性

### 7.4 测试命令

每个实现任务至少运行：

```bash
gofmt -w <changed-files>
go test ./path/to/changed/package
```

跨包或并发变更运行：

```bash
go test ./...
go test -race ./...
```

发布前运行：

```bash
go vet ./...
staticcheck ./...
go test ./... -coverprofile=coverage.out
```

## 8. Boundaries

### Always：必须做

- 先更新 Spec/Plan/Task，再实现新的公共行为或架构决策。
- 先写失败测试，再写实现。
- 为外部输入做 schema、路径、尺寸和权限校验。
- 使用 `context.Context` 支持请求、工具和扩展取消。
- 测试真实用户可观察的状态、输出、事件和持久化结果。
- 保证 Session 原子写入和旧数据可恢复。
- 对凭据和日志执行脱敏。
- 每个阶段结束运行对应测试和验证命令。

### Ask first：必须先确认

- 删除当前功能或 Provider。
- 改变 Session JSONL 格式或凭据存储格式。
- 改变现有 CLI 参数、退出码、RPC schema 或扩展协议。
- 引入或升级外部依赖，包括 Bubble Tea、Lip Gloss、Cobra 等；必须先确定明确版本并审查 `go.mod`/`go.sum`，特别关注 CGO 和原生数据库依赖。
- 修改 CI、发布流程、支持平台或最低 Go 版本。
- 允许 Go 版直接执行 TypeScript 扩展。
- 将 TUI、Agent 或 Provider 合并成无法独立测试的模块。

### Never：禁止做

- 提交 API Key、OAuth token、cookie 或个人 Session。
- 为了通过测试删除、跳过或弱化失败测试。
- 使用真实 Provider、真实凭据或付费 API 作为普通自动化测试依赖。
- 在 RPC stdout 写入未声明的日志或调试信息。
- 未经确认修改或删除原有 TypeScript 实现。
- 使用 `git reset --hard`、`git checkout .`、`git clean -fd` 或覆盖其他会话的改动。
- 在未备份和未提供 dry-run 的情况下迁移用户 Session 或凭据。

## 9. Success Criteria

### 功能验收

- 用户可以在无 Node.js 运行时的机器上启动 Go `pi`。
- 用户可以登录至少一个 API Key Provider 和一个 OAuth Provider。
- 用户可以发送 prompt，看到流式文本/ thinking，并完成工具调用。
- 用户可以取消请求、发送 steering/follow-up，并继续 Session。
- Go 版可以读取现有代表性 Session，并支持 resume、fork、tree 和 compaction。
- print、RPC 和 interactive 使用同一套 Agent runtime。
- TUI 在窄终端、宽终端、resize、中英文输入、粘贴和 Escape 下不破坏终端状态。

### 质量验收

- 所有必须兼容项有自动测试或记录过的手工测试。
- 每个新增行为都有 RED -> GREEN -> REFACTOR 记录。
- `go test ./...`、`go test -race ./...`、`go vet ./...` 和 `staticcheck ./...` 通过。
- 关键包覆盖率达到项目确定的门槛，且不通过删除测试达标。
- Provider、Session、RPC、凭据和工具没有已知数据丢失或凭据泄漏路径。
- 跨平台二进制可以通过 `--help`、`--version`、print、RPC 和 interactive smoke test。
- 有迁移 dry-run、备份和回滚文档，并在代表性 Session 上验证。
- TypeScript 运行时只在 Go 版通过所有迁移门槛后退役。

## 10. Open Questions

1. 第一版必须发布哪些 Provider？
2. 是否要求当前所有 TypeScript 扩展继续可用？如果要求，是否接受 Node adapter 而不是纯 Go 扩展？
3. Session JSONL 是否需要永久写兼容，还是只要求永久读兼容？
4. 第一版是否只支持 macOS/Linux？
5. 是否需要同时重写当前 `server` 包？
6. 首发版本对 OAuth Provider 的最低要求是什么？
7. 是否接受 Go 版第一阶段不支持图片生成，只支持图片输入和终端图片回退？
8. 最低测试覆盖率门槛设为多少？建议核心包行覆盖率不低于 80%，但以行为覆盖为主。

## 11. Spec 审批门槛

在进入实现前，项目负责人需要确认：

- [ ] Objective 和用户范围正确。
- [ ] 技术栈和首批 Provider 正确。
- [ ] Session、扩展和 RPC 兼容边界正确。
- [ ] Success Criteria 可接受且可测试。
- [ ] Open Questions 已有决定，或明确标记为后续决策。

确认 Spec 后，才进入 `docs/plan.md` 的实现阶段；任何范围或架构变化先更新本文件。
