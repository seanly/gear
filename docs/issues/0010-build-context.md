# 0010 额外构建上下文

**里程碑** M9

**依赖** 0002

## 目标

`gear build` 和 `gear debug` 可以把多个具名目录传给 `docker buildx build --build-context`。

## 范围

- `--build-context` 可重复，值是 `name=path`。
- 空字符串跳过。没有 `=`，或 `=` 左侧为空，则失败，并且不调用 docker。
- 不检查路径是否存在。本地目录、Git URL、HTTP 压缩包、`docker-image://`、`oci-layout://` 都原样交给 buildx。
- 同一个名字写两次不合并，原样按顺序传给 buildx。
- 不代替命令末尾的默认构建上下文。
- 不解析 `DAPPER_RUN_ARGS`，也不把构建写入回写到该目录。

## 实现要点

`Request.BuildContexts` 保存这些值。`commonArgs` 对每个非空值追加 `--build-context`。`BuildArgv` 和 `DebugBuildArgv` 都走 `commonArgs`。

校验在组 argv 并调用 docker 之前完成。

## 验收标准

- [x] 两个 `--build-context` 都出现，顺序与传入一致。
- [x] 省略时 argv 里没有 `--build-context`。空字符串被跳过。
- [x] `gear debug` 的加载 argv 同样带上 `--build-context`。
- [x] 缺少 `=` 或名字为空时返回错误，并且不调用 docker。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-30 用假 `Runner` 验证，未启动 Docker：

- `TestBuildContextArgv` 核对两个上下文的顺序，并确认空字符串被跳过。
- `TestBuildContextOmitted` 确认省略时 argv 不含 `--build-context`。
- `TestDebugBuildContext` 核对 `gear debug` 的加载 argv。
- `TestBuildContextRejectedBeforeDocker` 对 `nopath` 和 `=onlyvalue` 返回错误，`build` 与 `debug` 都不调用 docker。
