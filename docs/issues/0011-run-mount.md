# 0011 挂载运行

**里程碑** M10

**依赖** 0003

## 目标

`gear run` 把阶段加载成本机镜像，再按镜像里的 `GEAR_` 或 `DAPPER_` 环境变量把当前目录挂进容器。这对应 Dapper 的 bind。不实现 cp。

## 范围

- 加载标签是 `gear-run`，不覆盖 `gear-debug`。
- 一次 `docker image inspect` 读取 `Config.WorkingDir` 和 `Config.Env`。
- 同一项先用 `GEAR_`，没有再用 `DAPPER_`。
- `SOURCE` 是容器内挂载点，也是 `-w`。都没写时用 `WORKDIR`。结果为空或 `/` 时失败，并且不 `docker run`。
- `CP` 是宿主机目录，相对当前工作目录，默认 `.`。绝对路径直接使用。
- `RUN_ARGS` 按空白拆开，插在镜像名之前。不支持引号。
- `ENV` 是要转发的变量名，每个变成 `-e NAME`。
- `DOCKER_SOCKET=true` 时挂载 `/var/run/docker.sock`。
- 总是传入 `GEAR_UID`、`GEAR_GID`、`DAPPER_UID`、`DAPPER_GID`。GEAR 不在宿主机上改属主。
- 不读 `OUTPUT`。
- `--volume`、`--env`、`--entrypoint` 只加在 `docker run` 上。`gear build` 和 `gear debug` 没有这三个标志，也不读这些变量。
- 未写 `--entrypoint` 时保留镜像入口。后面的参数替换 `CMD`。
- 标准输出是终端时加 `-t`。`Exec` 替换当前进程。
- 非法 `--volume` 在调用 docker 之前失败。

## 实现要点

```text
docker buildx build --load -t gear-run --file Dockerfile --target build .
docker image inspect -f '{"workdir":{{json .Config.WorkingDir}},"env":{{json .Config.Env}}}' gear-run
docker run --rm -i -t -v <cwd>:<source> -w <source> -e GEAR_UID=... gear-run ci
```

## 验收标准

- [x] `GEAR_SOURCE` 盖过同次 inspect 里的 `DAPPER_SOURCE`。
- [x] 只有 `DAPPER_SOURCE` 时挂到该路径。
- [x] 没有 source 时用 `WORKDIR`。
- [x] `RUN_ARGS`、转发的环境变量、四条 uid/gid 出现在 `docker run` 里。
- [x] 命令行 `--volume` 排在 `RUN_ARGS` 之后。省略 `--entrypoint` 时不传该参数；写了就覆盖。
- [x] 挂载点为 `/` 时不执行 `docker run`。
- [x] 非法 `--volume` 不调用 docker。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-30 用假 `Runner` 验证，未启动 Docker：

- `TestMountGearSourceOverridesDapper` 核对 `GEAR_SOURCE`、`GEAR_RUN_ARGS`、`GEAR_ENV` 盖过 `DAPPER_`，以及 socket、uid/gid、追加的 `--volume` 和 `--entrypoint`。
- `TestMountDapperSourceAndWorkdir` 核对只有 `DAPPER_SOURCE` 时的挂载，以及没有 source 时用 `WORKDIR`，并且不传 `--entrypoint`。
- `TestMountAbsoluteCP` 核对绝对路径的 `GEAR_CP`。
- `TestMountRejectsRootDestination` 确认挂载点为 `/` 时没有 `docker run`。
- `TestMountRejectsVolumeBeforeDocker` 确认非法 `--volume` 不调用 docker。
