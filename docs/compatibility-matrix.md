# Pi Go 兼容性矩阵

> 本文件是 Task 1 的工作入口。进入 Go 核心实现前，必须由项目负责人确认每项的范围和验收来源。

| 领域 | 行为 | 首发目标 | 验收来源 | 状态 |
|---|---|---|---|---|
| CLI | `--help`、`--version`、退出码 | 必须兼容 | 现有 CLI 行为 + Go 单测 | 基线骨架已建立 |
| CLI | interactive、print、RPC | 必须兼容 | Spec + 集成测试 | 待基线 |
| Agent | 文本和 thinking 流 | 必须兼容 | faux provider fixture | 待基线 |
| Agent | tool call、tool result、abort | 必须兼容 | faux provider fixture | 待基线 |
| Tools | `read`、`write`、`edit`、`bash` | 必须兼容 | 工具测试 | 待基线 |
| Session | JSONL 读取、恢复、分支 | 必须兼容 | 旧 Session fixture | 待基线 |
| Provider | OpenAI、Anthropic、Google、OpenAI-compatible | 首发范围 | Provider fixture | 待基线 |
| Provider | 其余 Provider | 分批迁移 | 单 Provider 任务卡 | 待确认 |
| Auth | API Key、环境变量、logout | 必须兼容 | 凭据测试 | 待基线 |
| Auth | OAuth | 首批一个，随后扩展 | OAuth 状态机测试 | 待确认 |
| TUI | 编辑器、resize、paste、Escape | 必须兼容 | Bubble Tea/golden/tmux | 待基线 |
| TUI | 图片生成 | 暂不纳入首发 | Spec Open Question 7 | 待确认 |
| Extensions | 直接加载 TypeScript | 明确不兼容 | 改用 JSON-RPC | 已决定 |
| Extensions | 子进程 Go/TypeScript/Python | 首发范围 | Extension protocol fixture | 待实现 |
| Server | server 包 | 待确认 | Spec Open Question 5 | 待确认 |
| Storage | SQLite 索引 | 可选 | Spec + Session tests | 待确认 |

## 状态定义

- **待基线**：需要从 TypeScript 行为或现有测试提取 fixture。
- **待确认**：需要项目负责人决定是否属于首发范围。
- **已决定**：兼容边界已经明确，可以按计划实现。
- **完成**：自动测试和人工验证均通过。
