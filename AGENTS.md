# pi-go Agent Instructions

## Git and Pull Requests

- Never commit directly to `main`.
- Every change must be made on a short-lived branch and submitted through a pull request.
- The user is the reviewer and merger. Do not merge, approve, close, or delete the pull request unless the user explicitly requests it.
- Do not force-push shared branches.
- Keep one logical change per pull request. Split unrelated fixes, formatting, dependency changes, and features.
- Before creating or updating a pull request, inspect `git status`, `git diff`, and the staged diff. Do not include unrelated files or user changes.
- Use descriptive branch names such as `feature/<short-name>`, `fix/<short-name>`, or `chore/<short-name>`.
- Use descriptive commit messages with the format `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, or `chore:`. Commits on a PR branch are allowed; direct commits to `main` are not.

## Pull Request Description

Every pull request must clearly state:

- **修改内容**：具体改了哪些文件和行为。
- **修改原因**：为什么需要修改，以及对应的 Spec/Plan/Task。
- **验证内容**：执行了哪些测试、静态检查、构建或手工验证命令，以及结果。
- **影响范围**：可能影响哪些模块、用户流程、协议、Session 或发布流程。
- **未修改内容**：明确列出有意不触碰的模块或已知不属于本 PR 的问题。
- **风险和回滚**：已知风险、迁移影响和回滚方式。

PR 标题应说明变更类型和目的，例如：

```text
feat: add normalized agent event contract
fix: preserve partial messages on provider abort
chore: update CI action runtimes
```

## Required Workflow

1. Read `docs/spec.md`, `docs/plan.md`, and the relevant task in `docs/tasks.md`.
2. Create a short-lived branch from the latest `main`.
3. Write a failing test before implementing behavior changes.
4. Implement the smallest change that makes the test pass, then refactor with tests green.
5. Run the required local checks for the affected task.
6. Create or update a PR with the full modification summary above.
7. Wait for required GitHub Actions checks to pass.
8. Stop and let the user review and merge the PR.

## Quality Gates

Before requesting review, run the checks required by the affected task. The baseline checks are:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build -trimpath -o ./bin/pi ./cmd/pi
```

A non-empty `gofmt -l` result is a failure. Do not skip tests, weaken assertions, or hide CI failures.

## Scope and Safety

- Do not commit API keys, OAuth tokens, cookies, private sessions, or local environment files.
- Do not modify or delete the original TypeScript implementation unless a reviewed task explicitly requires it.
- Do not change Session formats, RPC schemas, extension protocols, CI branch protection, or release behavior without a dedicated task and PR description.
- Do not use destructive Git commands such as `git reset --hard`, `git checkout .`, `git clean -fd`, or force-pushes.
- Treat generated files, lockfiles, GitHub Actions, and dependency changes as reviewed code.
- If unrelated user changes are present, leave them untouched and exclude them from the PR.

## Documentation

Keep `docs/spec.md`, `docs/plan.md`, `docs/tasks.md`, and `docs/compatibility-matrix.md` consistent. Update the relevant document before changing scope or architecture. Every feature PR must reference its task and update documentation when behavior or public interfaces change.
