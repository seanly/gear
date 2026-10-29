# 0004 构建入口

**里程碑** M3

**依赖** 0001

## 目标

用 Dockerfile 和 Makefile 构建 `gear` 二进制。

## 范围

- `Dockerfile`：`build` 阶段编译，`result` 阶段 `FROM scratch` 只放二进制。
- `Makefile`：`build` 在本机编译，`image` 用 `docker buildx` 把 `result` 导出到 `bin/gear`，另有 `test` 和 `clean`。
- `.dockerignore` 排除 `bin/` 和 `.git`。
- 版本通过 `-ldflags -X main.VERSION` 写入。

## 实现要点

`make image` 直接调用 `docker buildx`，这样仓库里还没有 `gear` 时也能构建：

```text
docker buildx build --file Dockerfile --target result --output type=local,dest=bin .
```

`result` 里的文件是 `/gear`，导出后是 `bin/gear`。这是 Linux 二进制。`make build` 编的是当前系统的二进制，两者都写到 `bin/gear`。

## 验收标准

- [x] `make build` 生成可执行的 `bin/gear`。
- [x] `make test` 通过。
- [x] Dockerfile 含 `build` 与 `result` 两个阶段，`result` 为 `scratch`。
- [x] `make image` 的参数是 `docker buildx build --target result --output type=local,dest=bin`。

## 完成情况

2026-10-29 在 darwin/arm64 上验证：

- `make build` 生成 `bin/gear`，`./bin/gear --version` 打印 `0.0.0-dev`（目录还不是 git 仓库，版本回退到默认值）。
- `make test` 通过。
- `make image` 会拉取 `golang:1.25` 并导出 Linux 二进制，本次未实际拉取镜像。
