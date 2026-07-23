# pi-go

Pi 的纯 Go 重写项目。目标是在不依赖 Node.js 的情况下提供完整的 coding agent CLI、Agent runtime、多 Provider、Session、工具调用、RPC 和交互式 TUI。

项目当前先实现 Core：`packages/ai`、`packages/agent`、`packages/storage`。TUI、扩展和 server 延后。TypeScript 版本仍是行为兼容基准，Go 版本通过迁移门禁前不会替代现有运行时。

## 文档

- [规格说明](docs/spec.md)
- [实施计划](docs/plan.md)
- [任务清单](docs/tasks.md)

## 技术栈

- Go 1.26.1
- Bubble Tea + Lip Gloss
- `net/http` + SSE
- JSONL Session 和 RPC
- TDD：RED -> GREEN -> REFACTOR
- GitHub Actions CI/CD

## 开发

```bash
gofmt -w ./cmd ./internal
go vet ./...
go test ./...
go test -race ./...
go build -trimpath -o ./bin/pi ./packages/coding-agent/cmd/pi
./bin/pi --help
```

## CI/CD

每次 push 和 pull request 都执行：

1. `gofmt` 检查
2. `go vet ./...`
3. `go test ./...`
4. `go test -race ./...`
5. macOS/Linux 的 amd64/arm64 交叉构建

推送 `v*` tag 后，Release workflow 会重新执行质量门禁、构建四个平台二进制、生成 SHA256SUMS 并发布 GitHub Release。

## License

MIT
