# pi-ai

纯 Go 的 AI 核心库，对应 TypeScript `@earendil-works/pi-ai`。提供模型类型、Provider 注册、流式协议适配器和凭据管理。零外部依赖，自包含。

## 安装

```go
// go.mod
require github.com/DROWNING2003/pi-go/packages/ai v0.0.0

replace github.com/DROWNING2003/pi-go/packages/ai => ../path/to/pi-go/packages/ai
```

或直接用 import path（当 pi-go 是依赖时）：

```go
import ai "github.com/DROWNING2003/pi-go/packages/ai"
```

## 快速开始

### 1. 使用 Provider Registry

```go
import ai "github.com/DROWNING2003/pi-go/packages/ai"

func main() {
    // 创建 registry 并注册内置 Provider
    reg := ai.NewRegistry(nil)
    ai.RegisterBuiltins(reg)

    // 查找模型
    m := reg.ResolveModel("deepseek/deepseek-chat")
    if m == nil {
        panic("model not found")
    }

    // 解析 API key（从环境变量 DEEPSEEK_API_KEY）
    key := reg.ResolveAPIKeyForProvider("deepseek", "")
    if key == "" {
        panic("set DEEPSEEK_API_KEY")
    }
}
```

### 2. 流式对话

```go
client := ai.NewHTTPClient("https://api.deepseek.com", map[string]string{
    "Authorization": "Bearer " + key,
})

ctx := ai.Context{
    Messages: []json.RawMessage{
        json.RawMessage(`{"role":"user","content":"hello","timestamp":1}`),
    },
}

for evt := range ai.StreamChatCompletion(context.Background(), client, m, &ctx, nil) {
    switch evt.Type {
    case ai.StreamEventTextDelta:
        fmt.Print(evt.Delta)
    case ai.StreamEventDone:
        fmt.Println("\nDone:", evt.Message.StopReason)
    case ai.StreamEventError:
        fmt.Println("Error:", evt.Error.ErrorMessage)
    }
}
```

### 3. 构建消息

```go
// 用户消息
userMsg := ai.UserMessage{
    Role:    "user",
    Content: ai.UserContent{ai.NewTextContent("hello")},
    Timestamp: time.Now().UnixMilli(),
}

// Assistant 消息
asstMsg := ai.AssistantMessage{
    Role: "assistant",
    Content: []ai.ContentBlock{
        ai.NewTextContent("world"),
        ai.NewThinkingContent("let me think..."),
        ai.NewToolCallContent("call-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
    },
    API:    "openai-completions",
    Provider: "openai",
    Model:  "gpt-4o",
    Usage:  ai.Usage{TotalTokens: 10},
    StopReason: ai.StopReasonStop,
}
```

### 4. 测试用 Faux Provider

```go
faux := ai.NewFauxProvider()
faux.SetResponses(
    ai.FauxMessage{
        Message: &ai.AssistantMessage{
            Role: "assistant",
            Content: []ai.ContentBlock{ai.NewTextContent("hello")},
            StopReason: ai.StopReasonStop,
        },
    },
)

m := faux.GetModel()
for evt := range faux.Stream(context.Background(), m, &ai.Context{}, nil) {
    // 处理事件
}
```

## 包结构

```
packages/ai/
  ai.go             统一导出 API
  model/            核心类型（消息、事件、Usage、StopReason）
  provider/         Registry、凭据存储、Compat 检测、Faux Provider
  protocol/         HTTP/SSE 客户端 + 4 种 Provider 协议适配器
```

## 支持的 Provider

| Provider | API 协议 | 环境变量 |
|---|---|---|
| DeepSeek | OpenAI Completions | `DEEPSEEK_API_KEY` |
| OpenAI | OpenAI Completions / Responses | `OPENAI_API_KEY` |
| Anthropic | Anthropic Messages | `ANTHROPIC_API_KEY` |
| Google | Gemini Generate | `GOOGLE_API_KEY` |
| Faux (测试) | 内存脚本化 | 无 |

## API 参考

### 核心类型

| 类型 | 说明 |
|---|---|
| `ContentBlock` | text/thinking/image/toolCall 内容块 |
| `UserMessage` / `AssistantMessage` / `ToolResultMessage` | 三种消息类型 |
| `Usage` / `UsageCost` | Token 用量和费用 |
| `StopReason` | stop/length/toolUse/error/aborted |
| `StreamEvent` | Provider 流式事件（12 种） |
| `UnifiedModel` | 模型描述符 |
| `Context` | 对话上下文 |

### Registry

| 方法 | 说明 |
|---|---|
| `NewRegistry(store)` | 创建 registry |
| `RegisterBuiltins(r)` | 注册内置 Provider |
| `ResolveModel(ref)` | 按 `provider/model` 或 `model` 查找 |
| `ResolveAPIKeyForProvider(id, key)` | 解析 API key |
| `SetDispatcher(api, fn)` | 注册流式分发器 |

### 凭据

| 方法 | 说明 |
|---|---|
| `NewCredentialStore(dir)` | 创建凭据存储（0600 文件权限） |
| `CredentialStore.Save/Load/Delete` | 持久化凭据 |
| `ResolveAPIKey(store, id, envVars, key)` | 解析优先级：存储 > 环境变量 > 参数 |
| `DetectCompat(id, baseURL)` | 自动检测 Provider 兼容性 |

### 协议适配器

| 函数 | API |
|---|---|
| `StreamChatCompletion` | OpenAI Chat Completions |
| `StreamOpenAIResponses` | OpenAI Responses |
| `StreamAnthropicMessages` | Anthropic Messages |
| `StreamGoogleGenerate` | Google Gemini |

## 兼容性

与 TypeScript `@earendil-works/pi-ai` JSONL 格式完全兼容。所有消息和事件类型可 round-trip。
