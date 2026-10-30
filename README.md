# GEAR

GEAR（Generic Execution And Runtime）用 `docker buildx` 处理 Dockerfile 的一个阶段。`gear build` 把该阶段导出到宿主机目录。`gear debug` 进入镜像里的文件系统。`gear run` 把当前目录挂进容器再运行。

只支持 Linux 和 macOS。只调用本机的 `docker`，并且构建必须走 `docker buildx`。没有 buildx 插件时直接失败。

## 用法

`--file` / `-f` 默认是当前目录的 `Dockerfile`。`--target` 是阶段名。构建上下文默认是当前目录 `.`，`gear build` 可以在命令末尾再写一个路径。

### 导出产物

产物由一个 `scratch` 阶段声明。`--output` / `-o` 默认是 `dist`。

```dockerfile
FROM alpine:3.20 AS build
RUN mkdir -p /out && echo gear > /out/hello.txt

FROM scratch AS result
COPY --from=build /out /
```

```sh
gear build --target result
gear build --target result --file examples/Dockerfile
```

### 进入镜像

`gear debug` 把阶段加载为本地镜像 `gear-debug`，再用 `/bin/sh` 进入。它不挂载当前目录，容器里看到的是构建时复制进去的文件。后面的参数传给 shell；没有参数时进入交互 shell。标准输出是终端时才会加上 `-t`。

```sh
gear debug --target build
```

### 挂载当前目录

`gear run` 对应 Dapper 的 bind。它先把阶段加载为 `gear-run`，再从镜像环境变量读出挂载方式，然后 `docker run -v` 把当前目录挂进去。容器里写到挂载点下的文件会出现在当前目录。不实现 Dapper 的 cp，也不读取 `GEAR_OUTPUT` / `DAPPER_OUTPUT`。

同一项先用 `GEAR_`，没有再用 `DAPPER_`。旧模版可以继续写 `DAPPER_`。

| 变量 | 作用 |
| --- | --- |
| `GEAR_SOURCE` | 容器内挂载点，也是工作目录。没写时用镜像 `WORKDIR`。结果为空或 `/` 时失败 |
| `GEAR_CP` | 宿主机目录，相对当前目录，默认 `.`。绝对路径直接使用 |
| `GEAR_RUN_ARGS` | 按空白拆开，加到 `docker run` 上，例如缓存目录的 `-v` |
| `GEAR_ENV` | 要从宿主机转发的变量名，空格分隔 |
| `GEAR_DOCKER_SOCKET` | 值为 `true` 时挂载 `/var/run/docker.sock` |

每次运行还会传入 `GEAR_UID`、`GEAR_GID`、`DAPPER_UID`、`DAPPER_GID`。GEAR 自己不改宿主机上的文件属主。

```dockerfile
ENV GEAR_SOURCE /src
ENV GEAR_RUN_ARGS -v /var/cache/m2:/root/.m2
ENV GEAR_ENV APP_NAME APP_ENV
WORKDIR /src
ENTRYPOINT ["bash", "./scripts/entry"]
CMD ["ci"]
```

```sh
gear run --file .gear/Dockerfile --target build ci
```

后面的参数替换镜像的 `CMD`。入口默认是镜像自己的 `ENTRYPOINT`。要临时多挂一个目录、多传一个变量或换入口：

```sh
gear run --target build \
  --volume /var/cache/npm:/var/cache/npm \
  --env APP_ENV=test \
  --entrypoint bash \
  ci
```

`--volume` 的值是 `host:container`，两侧都不能空。

### 额外构建上下文

`--build-context` 可重复，值是 `name=path`，原样传给 buildx。它是额外目录，Dockerfile 用 `COPY --from=name` 或 `RUN --mount=from=name` 才会读到，不改变命令末尾那个代码库上下文。名字为空或缺少 `=` 时直接失败。这个目录只进构建，构建不会把文件写回去。

```sh
gear build --file .gear/Dockerfile --target result \
  --build-context m2=/var/cache/m2 \
  .
```

同一台机器上要留下依赖下载，在 Dockerfile 里用 `RUN --mount=type=cache`。两台机器共享的是层缓存：

```sh
gear build --target result \
  --cache-from type=registry,ref=example.com/app:buildcache \
  --cache-to type=registry,ref=example.com/app:buildcache,mode=max
```

`type=cache` 的目录不会进这份层缓存。

### 查看阶段

`gear show targets` 只读 Dockerfile，不调用 Docker。每个阶段单独成组，依赖用 `FROM`、`COPY --from` 和 `RUN --mount from` 写在名字下面。文件里写了默认值的 `ARG` 会展开；没有默认值的变量保持原文。`INCLUDE`、`ADD --from` 和其他未知指令会失败。

```sh
gear show targets
gear show targets --file examples/Dockerfile
```

### 镜像加速

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
- [从 Dapper 模版迁移](docs/guides/from-dapper.md)
- [Issue 索引](docs/issues/README.md)
- [给实现者的入口](AGENTS.md)
