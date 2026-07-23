# pi-go

Pi 的纯 Go 重写。不依赖 Node.js，单一二进制，支持多 Provider、工具调用、Session 持久化。

## 快速开始

```bash
# 构建
go build -trimpath -o ./bin/pi ./packages/coding-agent/cmd/pi

# 设置 API key
export DEEPSEEK_API_KEY=sk-xxx

# 对话
./bin/pi --print "解释 Go 的 goroutine"

# 切换模型
./bin/pi --model openai/gpt-4o --print "Hello"

# 指定系统提示
./bin/pi --system "用中文回答" --print "What is Docker?"
```

## 支持的 Provider

| Provider | 环境变量 | API 协议 |
|---|---|---|
| DeepSeek | `DEEPSEEK_API_KEY` | OpenAI Completions |
| OpenAI | `OPENAI_API_KEY` | OpenAI Completions / Responses |
| Anthropic | `ANTHROPIC_API_KEY` | Anthropic Messages |
| Google | `GOOGLE_API_KEY` | Gemini Generate |

更多 OpenAI-compatible 提供商可通过兼容层自动适配（Together、Groq、Fireworks、Cerebras、xAI 等）。

## 命令行

```
Usage: pi [options] [message...]

Options:
  --help       Show help
  --version    Show version
  --print      Non-interactive print mode
  --model      Model reference (e.g. deepseek/deepseek-chat)
  --provider   Provider override
  --system     System prompt override
  --workspace  Workspace directory (default: current dir)
```

## 架构

```
packages/
  ai/
    model/       标准消息、ContentBlock、Usage、StopReason
    provider/    Provider Registry、凭据存储、Compat 检测
    protocol/    HTTP/SSE 客户端 + 4 种 Provider 协议适配器
  agent/
    event/       Agent 生命周期事件
    loop/        Agent 循环（流式响应 → 工具调用 → 继续）
    tool/        read / write / edit / bash 工具
    queue/       Abort / Steering / Follow-up 队列
  storage/
    session/     JSONL Session 原子读写、fork、损坏恢复
    config/      全局/项目配置、AGENTS.md 加载、Trust
  coding-agent/
    cli/         CLI 入口（print 模式）
    cmd/pi/      main.go
```

依赖方向：`model → provider/protocol → agent/tool → loop → cli`

## 开发

```bash
gofmt -w . && go vet ./...
go test ./... -count=1
go test -race ./...
go build -trimpath -o ./bin/pi ./packages/coding-agent/cmd/pi
```

## 文档

- [规格说明](docs/spec.md)
- [实施计划](docs/plan.md)
- [任务清单](docs/tasks.md)
- [兼容矩阵](docs/compatibility-matrix.md)
