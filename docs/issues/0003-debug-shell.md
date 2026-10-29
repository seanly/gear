# 0003 进入构建阶段

**里程碑** M2

**依赖** 0001

## 目标

`gear debug` 加载一个有 shell 的阶段，并进入该环境。`scratch` 的 `result` 阶段没有 shell，调试时 `--target` 指向 `build`。

## 范围

- 用 `docker buildx build --load -t gear-debug` 构建该阶段。
- 再执行 `docker run --rm -i --entrypoint /bin/sh gear-debug`。
- 标准输出是终端时追加 `-t`。
- 额外参数传给 shell。没有参数时进入交互 shell。
- 构建失败时不启动 `docker run`。
- `--file` 默认 `Dockerfile`，其余构建参数与 `gear build` 相同，但不做目录导出。

## 实现要点

先 `EnsureBuildx`，再 `Run` 加载镜像。加载成功后 `Exec` 替换当前进程，这样交互 shell 拿到终端和 docker 的退出码。

```text
docker buildx build --load -t gear-debug --file Dockerfile --target build .
docker run --rm -i -t --entrypoint /bin/sh gear-debug
```

## 验收标准

- [x] 加载 argv 含 `--load -t gear-debug --file Dockerfile --target build`。
- [x] 终端上的运行 argv 是 `run --rm -i -t --entrypoint /bin/sh gear-debug`。
- [x] 非终端省略 `-t`。
- [x] 额外参数追加在镜像名之后。
- [x] 构建返回错误时不出现 `docker run`。

## 完成情况

2026-10-29 用假 `Runner` 验证，未启动 Docker：

- `TestDebugArgv` 核对加载参数、终端下的 `-t`，以及非终端省略 `-t`。
- `TestDebugLoadsThenRuns` 确认先 `buildx version`，再加载，最后 `run`。
- `TestDebugStopsWhenBuildFails` 确认构建失败后没有 `run`。
- `./bin/gear debug --help` 列出与构建共用的 `--file`、`--target`、`--build-arg`、`--secret`、`--cache-from`、`--cache-to`、`--platform`。
