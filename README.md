# GEAR

GEAR（Generic Execution And Runtime）用 `docker buildx` 把 Dockerfile 的一个阶段导出到宿主机目录。

构建写在普通 Dockerfile 的阶段里。产物由一个 `scratch` 阶段声明，GEAR 调用 `docker buildx build --output type=local,dest=<dir>` 把该阶段写到目录。需要进入构建环境时，对有 shell 的阶段使用 `gear debug`。

只支持 Linux 和 macOS。只调用本机的 `docker`，并且构建必须走 `docker buildx`。没有 buildx 插件时直接失败。

## 用法

```dockerfile
FROM alpine:3.20 AS build
RUN mkdir -p /out && echo gear > /out/hello.txt

FROM scratch AS result
COPY --from=build /out /
```

```sh
gear build --target result
gear build --target result --file examples/Dockerfile
gear debug --target build
```

`--file` / `-f` 默认是当前目录的 `Dockerfile`。`--target` 是阶段名。`--output` / `-o` 默认是 `dist`，阶段根写到这个目录；要换目录再写 `--output`。构建上下文默认是当前目录 `.`，也可以在命令末尾再写一个路径。`gear debug` 后面的参数传给容器里的 `/bin/sh`；没有参数时进入交互 shell。标准输出是终端时才会加上 `-t`。

Docker 守护进程没有 registry mirror 时，可以给这一次构建单独指定 Docker Hub 镜像站：

```sh
gear build --target result --registry-mirror=mirror.example.com
```

GEAR 会创建一个名为 `gear-mirror-example-com` 的 buildx builder，把 `docker.io` 的拉取转到该主机，并且不改默认 builder，也不改守护进程配置。环境变量 `GEAR_REGISTRY_MIRROR` 作用相同。

退出码沿用 `docker buildx` 的退出码。

## 构建

```sh
make build   # 本机二进制 bin/gear
make test
make image   # docker buildx 导出一个 Linux 二进制到 bin/gear
make dist    # 全平台包：linux/amd64、linux/arm64、darwin/amd64、darwin/arm64
make dist PLATFORM=darwin/arm64
make dist-image
```

`make dist` 把每个平台打成 `dist/gear_<version>_<os>_<arch>.tar.gz`，包里的文件名是 `gear`。全平台就是 Linux 和 macOS 的 amd64、arm64。`make dist-image` 在容器里做同样的交叉编译。`make image PLATFORM=linux/arm64` 只导出那一个 Linux 二进制。

## 文档

- [设计](docs/design.md)
- [Issue 索引](docs/issues/README.md)
- [给实现者的入口](AGENTS.md)
