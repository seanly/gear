# 0005 多平台打包

**里程碑** M4

**依赖** 0004

## 目标

一次打出 GEAR 能运行的全部平台，也能只打其中一个。

## 范围

全平台是 Linux 与 macOS 的 amd64、arm64：

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

`make dist` 默认打这四份。`PLATFORM=linux/amd64` 只打一份。包是 `dist/gear_<version>_<os>_<arch>.tar.gz`，里面的文件名是 `gear`。

`make dist-image` 在 `golang:1.25` 容器里做同样的交叉编译，再由 `dist` 阶段导出。`make image PLATFORM=linux/arm64` 仍只导出一个 Linux 二进制到 `bin/gear`。

## 实现要点

`CGO_ENABLED=0`，用 `GOOS` 和 `GOARCH` 交叉编译。macOS 包也在 Linux 构建容器里编出来，不依赖 `--platform darwin/arm64`。

Dockerfile 的 `packages` 阶段按 `PLATFORMS` 循环，`dist` 阶段 `FROM scratch` 只放这些 tar.gz。

## 验收标准

- [x] `make dist` 生成四份包：linux/amd64、linux/arm64、darwin/amd64、darwin/arm64。
- [x] `make dist PLATFORM=darwin/arm64` 只生成 macOS arm64 这一份。
- [x] 包内文件名是 `gear`，文件类型与目标平台一致。
- [x] `make dist-image` 使用 `--target dist` 和同样的平台列表。

## 完成情况

2026-10-29 在 darwin/arm64 上用本机 Go 交叉编译，未启动 Docker：

- `make dist` 生成 `dist/gear_0.0.0-dev_{linux,darwin}_{amd64,arm64}.tar.gz`。
- `make dist PLATFORM=darwin/arm64` 只留下 `gear_0.0.0-dev_darwin_arm64.tar.gz`。
- 包内都是 `gear`。`file` 结果：Linux amd64 为 ELF x86-64，Linux arm64 为 ELF aarch64，macOS amd64 为 Mach-O x86_64，macOS arm64 为 Mach-O arm64。
- `make dist-image` 会拉取 `golang:1.25`，本次未实际拉取镜像。
