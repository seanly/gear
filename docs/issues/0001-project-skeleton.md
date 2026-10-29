# 0001 项目骨架

**里程碑** M0

**依赖** 无

## 目标

建立 `github.com/seanly/gear`，提供 `build` 与 `debug` 两个子命令的 CLI 骨架，并限制只在 Linux 和 macOS 上编译。

## 范围

- Go 1.25，`github.com/urfave/cli/v3`，`log/slog`。
- `main.go` 注册 `build` 和 `debug`。
- `internal/gear` 的 `Runner` 只执行 `docker`。
- 非 linux/darwin 编译失败。
- 构建前执行 `docker buildx version`，失败则退出。

## 实现要点

- `NewDockerRunner` 用 `exec.LookPath("docker")`。
- `EnsureBuildx` 调用 `docker buildx version`。
- `unsupported.go` 使用 `//go:build !linux && !darwin`，用字符串赋值给 `int` 让其他系统编译失败。
- `ExitError` 实现 `ExitCode()`，让 CLI 沿用 docker 的退出码。

## 验收标准

- [x] 模块路径是 `github.com/seanly/gear`，`go` 版本是 1.25.0。
- [x] `gear --help` 列出 `build` 和 `debug`。
- [x] Linux 与当前 darwin 可以编译。
- [x] `GOOS=windows go build` 失败，错误包含 `gear supports only linux and darwin`。
- [x] 假 `Runner` 在 `buildx version` 失败时，`Build` 返回包含 `docker buildx is required` 的错误。

## 完成情况

2026-10-29 在 darwin/arm64 上验证：

- `go test ./...`：`internal/gear` 通过。
- `go build -o ./bin/gear .` 成功，`./bin/gear --help` 列出 `build` 和 `debug`。
- `GOOS=linux GOARCH=amd64 go build` 成功。
- `GOOS=windows go build` 失败：`cannot use "gear supports only linux and darwin" (untyped string constant) as int value`。
- `TestBuildRequiresBuildxAndKeepsExitCode` 覆盖缺少 buildx 的失败路径。
