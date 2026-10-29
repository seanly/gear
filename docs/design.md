# GEAR 设计

读者：实现本项目的 Go 工程师。

阅读顺序：本文 → [docs/issues/README.md](issues/README.md) → 各 issue。

## 是什么

GEAR 把多阶段 Dockerfile 的某个阶段导出到宿主机。模块是 `github.com/seanly/gear`。只支持 Linux 和 macOS。Go 1.26，CLI 用 `github.com/urfave/cli/v3`，日志用 `log/slog`。

构建写在普通 Dockerfile 里。`build` 阶段安装工具并生成产物，`result` 阶段 `FROM scratch` 后只 `COPY` 要导出的目录。`gear build --target result` 调用：

```text
docker buildx build --file Dockerfile --target result --output type=local,dest=dist .
```

`--file` / `-f` 默认是当前目录的 `Dockerfile`。要换文件时再写 `--file`。`--target` 是 Dockerfile 阶段名，与 BuildKit 的 `--target` 相同。`--output` / `-o` 默认是宿主机目录 `dist`，接收该阶段的根。构建上下文默认是当前目录。

`result` 没有 shell。调试时 `--target` 指向 `build`：

```text
docker buildx build --load -t gear-debug --file Dockerfile --target build .
docker run --rm -i --entrypoint /bin/sh gear-debug
```

标准输出是终端时，`docker run` 再加上 `-t`。

## 和 Dapper 的分工

Dapper 用 `Dockerfile.dapper` 描述一套工具镜像，再 `docker run` 进这套环境，适合反复改源码、进 shell、把目录挂进容器。

GEAR 不复制 Dapper 的 `Dockerfile.dapper`、bind/cp 和 `docker cp`。产物由 Dockerfile 的 `scratch` 阶段声明，GEAR 只负责把该阶段导出到目录。

## 目标与非目标

目标：

- `gear build` 把指定阶段导出到目录。
- `gear debug` 把指定阶段加载到本机引擎并进入 shell。
- `gear show targets` 列出 Dockerfile 的阶段和阶段依赖。
- 退出码沿用 `docker buildx`。
- 没有 `docker buildx` 时失败。

非目标：

- 不调用经典 `docker build`。
- 不支持 podman、nerdctl，也不提供 `--runtime`。
- 不管理镜像仓库发布，不在导出后改文件属主。

## 构建本仓库

`make build` 在本机编译 `bin/gear`。`make image` 用 `docker buildx` 构建根目录 Dockerfile 的 `result` 阶段，并把 `/gear` 导出为 `bin/gear`。`PLATFORM=linux/amd64` 或 `linux/arm64` 选择这一个 Linux 平台。

`make dist` 交叉编译全平台。全平台是 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`。每个包是 `dist/gear_<version>_<os>_<arch>.tar.gz`，里面的文件叫 `gear`。`PLATFORM=darwin/arm64` 只打一个。`make dist-image` 在构建容器里做同样的交叉编译，从 `dist` 阶段导出。版本来自 `git describe`，通过 `-X main.VERSION` 写入；没有 git 信息时是 `0.0.0-dev`。

## 运行时

`internal/gear` 的 `Runner` 只执行 `docker`。`gear build` 和 `gear debug` 先跑 `docker buildx version`，失败则退出。测试用假 `Runner` 核对 argv，不启动 Docker。

`--registry-mirror` 在守护进程没有 registry mirror 时，为这一次构建使用 Docker Hub 镜像站。GEAR 创建独立的 `docker-container` builder，并在 BuildKit 配置里把 `docker.io` 的 `mirrors` 设为该主机。默认 builder 和守护进程配置保持原样。
