# 0002 导出构建阶段

**里程碑** M1

**依赖** 0001

## 目标

`gear build` 把 Dockerfile 的一个阶段导出到目录。

## 范围

- `--file` / `-f` 默认 `Dockerfile`。
- `--target` 是阶段名，有值才传给 buildx。
- `--output` / `-o` 默认 `.`，对应 `--output type=local,dest=<dir>`。
- `--build-arg` 可重复。
- `--secret`、`--cache-from`、`--cache-to`、`--platform` 有值才追加。
- 构建上下文默认 `.`，命令末尾可以再写一个路径。
- 退出码沿用 `docker buildx`。

## 实现要点

argv 为：

```text
docker buildx build --file Dockerfile --target <stage> --output type=local,dest=<dir> <context>
```

`--file` 省略时就是 `Dockerfile`。可选参数插在 `--output` 和上下文之间。`Runner.Run` 把 `exec.ExitError` 转成 `ExitError`。

## 验收标准

- [x] 默认 argv 是 `buildx build --file Dockerfile --target result --output type=local,dest=. .`。
- [x] `--file examples/Dockerfile` 覆盖默认文件。
- [x] 两个 `--build-arg` 都会出现；secret、cache、platform 只在有值时出现。
- [x] buildx 退出码 2 时，`Build` 返回 `ExitCode() == 2`。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-29 用假 `Runner` 验证，未启动 Docker：

- `TestBuildArgvDefaults`、`TestBuildArgvOptionalFlags` 核对 argv。
- `TestBuildRequiresBuildxAndKeepsExitCode` 核对退出码 2，以及 `buildx version` 成功后才执行构建。
- `./bin/gear build --help` 显示 `--file` 默认 `Dockerfile`，`--output` 默认 `.`。
