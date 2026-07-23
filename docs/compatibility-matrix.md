# Pi Go 兼容性矩阵

> Core checkpoint 已通过。所有必须兼容项均已实现并测试。

| 领域 | 行为 | 首发目标 | 状态 |
|---|---|---|---|
| AI | Model、Message、ContentBlock、Usage、StopReason | 必须兼容 | ✅ 完成（18 tests） |
| AI | Stream event：text、thinking、toolcall、done、error | 必须兼容 | ✅ 完成（10 tests） |
| AI | Provider interface 和 faux Provider | 必须兼容 | ✅ 完成（13 tests） |
| AI | OpenAI Completions 协议 | 必须兼容 | ✅ 完成（5 tests） |
| AI | OpenAI Responses 协议 | 必须兼容 | ✅ 完成（3 tests） |
| AI | Anthropic Messages 协议 | 必须兼容 | ✅ 完成（3 tests） |
| AI | Google Generative AI 协议 | 必须兼容 | ✅ 完成（3 tests） |
| AI | API Key 凭据存储（env + file） | 必须兼容 | ✅ 完成（6 tests） |
| AI | Compat 自动检测 | 必须兼容 | ✅ 完成（3 tests） |
| AI | Provider Registry + Builtins | 必须兼容 | ✅ 完成（3 tests） |
| CLI | `--help`、`--version`、退出码 | 必须兼容 | ✅ 完成（3 tests） |
| CLI | `--print` 模式 | 必须兼容 | ✅ 完成（可用） |
| CLI | interactive、RPC | 必须兼容 | ⏳ 延后 |
| Agent | Agent loop（turn、tool call、continuation） | 必须兼容 | ✅ 完成（2 tests） |
| Agent | Abort、Steering、Follow-up 队列 | 必须兼容 | ✅ 完成（4 tests） |
| Tools | `read`、`write`、`edit`、`bash` | 必须兼容 | ✅ 完成（6 tests） |
| Session | JSONL 原子读写、fork、损坏恢复 | 必须兼容 | ✅ 完成（4 tests） |
| Config | 全局/项目配置、AGENTS.md、Trust | 必须兼容 | ✅ 完成 |
| Provider | 其余 Provider | 分批迁移 | ⏳ 延后 |
| Auth | OAuth 登录 | 首批一个 | ⏳ 延后 |
| TUI | 交互式终端 UI | 必须兼容 | ⏳ 延后 |
| Extensions | 子进程 JSON-RPC | 首发范围 | ⏳ 延后 |
| Server | server 包 | 待确认 | ⏳ 延后 |

## 当前测试

- **86 个测试全部通过**
- `gofmt -l .` 无输出
- `go vet ./...` 通过
- `staticcheck ./...` 通过
- `go test -race ./...` 通过
- 真实 API 集成测试：DeepSeek（`DEEPSEEK_API_KEY`）
