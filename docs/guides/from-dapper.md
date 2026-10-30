# 从 Dapper 模版迁移

读者：要把 `Dockerfile.dapper` 和 `scripts/entry` 迁到 GEAR 的人。

`gear build` 不读取 `DAPPER_SOURCE`、`DAPPER_OUTPUT`、`DAPPER_ENV`、`DAPPER_RUN_ARGS`。导出写在普通 Dockerfile 的阶段里。要像 Dapper 的 bind 那样把当前目录挂进容器，用 `gear run`。`gear run` 不实现 cp。

## 三路输入

一次构建里有三件彼此独立的事。

- `--file` 决定读哪一份 Dockerfile。模版放在代码库的 `.gear/Dockerfile` 时写 `--file .gear/Dockerfile`。这只选择指令文本。
- 命令末尾的位置参数是默认构建上下文，默认是 `.`，也就是代码库。`COPY .` 和不带 `--from` 的 `ADD` 读的都是它。Dockerfile 放在 `.gear/` 里，上下文仍然是仓库根，受仓库根的 `.dockerignore` 约束。
- `--build-context name=path` 是额外目录，可以重复。Dockerfile 写了 `COPY --from=name` 或 `RUN --mount=from=name` 才会用到。不写就不影响 `COPY .`。

```sh
gear build --file .gear/Dockerfile --target result .
```

末尾的 `.` 是代码库。`COPY . /src` 复制仓库根，其中包含 `.gear`，除非 `.dockerignore` 排除了它。

位置参数写成 `.gear` 时，`COPY .` 只看见模版目录，仓库里的源码进不去。要用代码库时，位置参数保持仓库根。把代码库做成具名上下文也可以：`--build-context src=.`，Dockerfile 写 `COPY --from=src`。默认上下文直接放仓库根更直接。

多个目录各用一个名字：

```sh
gear build --file .gear/Dockerfile --target result \
  --build-context m2=/var/cache/m2 \
  --build-context npm=/var/cache/npm \
  .
```

同一个名字写两次不会合并成一个目录，后写的覆盖同名上下文。值原样交给 buildx，可以是本地目录、Git URL、HTTP 压缩包、`docker-image://` 或 `oci-layout://`。

`--build-context` 是单项输入。目录进入构建，构建不把文件写回该目录。`RUN --mount=type=bind,from=m2` 默认只读。加上 `rw` 后，只有这一条 `RUN` 里面能写；指令结束就丢掉，不进入镜像层，也不修改宿主机上的那个路径。Maven 新下载的包不会出现在 `/var/cache/m2`。

要从构建里拿出文件，写到挂载点以外的普通目录，由 `scratch` 阶段 `COPY` 出来，再由 `gear build --output` 导出。多次构建之间要留下依赖缓存，用 `RUN --mount=type=cache`。那份缓存在 BuildKit 自己的存储里，不是 `--build-context` 指向的目录。

## 和 Dapper 变量的对应

`DAPPER_SOURCE` 和 `DAPPER_CP` 是默认构建上下文加上 `COPY .`。源码不再绑进一个长期运行的容器。

`DAPPER_OUTPUT ./dist` 是名为 `result` 的 `scratch` 阶段。`gear build --target result` 把该阶段的根写到宿主机，`--output` 默认就是 `dist`。

`DAPPER_ENV` 里的 `APP_NAME`、`APP_ENV` 等写成 Dockerfile 的 `ARG`，调用时加 `--build-arg APP_NAME=...`。这些值在 `RUN` 里就是原来的环境变量。GEAR 不扫描宿主机环境再自动转发。

`#@MAVEN_SETTINGS@` 和 `#@DOT_NETRC_FILE@` 用已有的 `--secret`：

```sh
gear build --target result \
  --secret id=maven_settings,src=settings.xml
```

```dockerfile
RUN --mount=type=secret,id=maven_settings,target=/root/.m2/settings.xml \
    mvn -B package
```

工具镜像的层缓存仍然有效。跨机构建再加上 `--cache-from` 和 `--cache-to`。这是层缓存，不是 Maven 或 npm 的目录缓存。

`gear debug --target build` 进入该阶段构建完成后的文件系统。它不把宿主机源码再挂进去，也不是 bind 或 cp。

`gear build` 导出后不改文件属主。`gear run` 会传入 `GEAR_UID`、`GEAR_GID`、`DAPPER_UID`、`DAPPER_GID`，脚本可以自己 `chown`。

## gear run

Dapper 的 bind 是 `docker run -v`。`gear run` 加载阶段后，用 `docker image inspect` 读出镜像环境。同一项先用 `GEAR_`，没有再用 `DAPPER_`。旧模版可以继续写 `DAPPER_`。

- `GEAR_SOURCE` / `DAPPER_SOURCE` 是容器内挂载点和工作目录。都没写时用 `WORKDIR`。
- `GEAR_CP` / `DAPPER_CP` 是宿主机目录，默认当前目录。
- `GEAR_RUN_ARGS` / `DAPPER_RUN_ARGS` 原样加到 `docker run`，缓存目录的 `-v` 写在这里。
- `GEAR_ENV` / `DAPPER_ENV` 是要从宿主机转发的变量名。
- `GEAR_DOCKER_SOCKET` / `DAPPER_DOCKER_SOCKET` 为 `true` 时挂载 Docker socket。
- `GEAR_OUTPUT` 和 `DAPPER_OUTPUT` 不读。文件直接写在挂载目录里。

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

`--volume` 和 `--env` 只在要临时多加一项时使用。`--entrypoint` 覆盖镜像入口。挂载点解析成 `/` 时 `gear run` 失败，避免盖住镜像里的程序。

## 依赖缓存和脚本

Dapper 模版用 `DAPPER_RUN_ARGS` 把宿主机上的依赖目录挂进 `docker run`。GEAR 的构建没有这一步。依赖缓存写在 `RUN` 上：

```dockerfile
RUN --mount=type=cache,target=/root/.m2/repository \
    mvn -B -e -U clean package -Dmaven.test.skip=true
```

Node 模版里按 `APP_GROUP`、`APP_NAME`、`APP_ENV` 把 `node_modules` 打成 tar 再写回宿主机缓存目录的做法，改成缓存挂载。脚本不再读写宿主机上的 npm 缓存目录。`CACHE_ENABLED` 不再表示宿主机目录在不在。

```dockerfile
RUN --mount=type=cache,target=/var/cache/npm \
    bash ./scripts/build
```

`scripts/entry` 按第一个参数转发到 `scripts/ci` 或 `scripts/build`。在 GEAR 里，跑哪一段就是 `RUN` 哪条命令。仓库里的 `./build.sh` 可以留着，由 `RUN bash ./build.sh` 调用。

脚本如果仍然生成 `.dapper/Dockerfile`，这个文件必须被 `result` 阶段复制出来，才会出现在宿主机上。例如再写一行 `COPY --from=build /src/.dapper/Dockerfile /Dockerfile`，或让脚本把该文件写进 `dist/`。

## JDK 模版

`.gear/Dockerfile`：

```dockerfile
# syntax=docker/dockerfile:1
FROM maven:3.9-eclipse-temurin-21 AS build
ARG APP_NAME
ARG APP_ENV
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/root/.m2/repository \
    --mount=type=secret,id=maven_settings,target=/root/.m2/settings.xml \
    bash -lc 'mkdir -p dist && if [ -f ./build.sh ]; then bash ./build.sh; else mvn -B -e -U clean package -Dmaven.test.skip=true; fi'
FROM scratch AS result
COPY --from=build /src/dist /
```

```sh
gear build --file .gear/Dockerfile --target result \
  --build-arg APP_NAME=demo --build-arg APP_ENV=test \
  --secret id=maven_settings,src=settings.xml \
  --build-context m2=/var/cache/m2 \
  --build-context npm=/var/cache/npm \
  .
```

上面的 `m2` 和 `npm` 只有在 Dockerfile 里写了 `--from=m2` 或 `--from=npm` 时才会被读到。依赖是否跨构建保留，由 `type=cache` 决定，不由这两个目录决定。
