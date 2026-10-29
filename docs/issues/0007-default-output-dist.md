# 0007 默认导出到 dist

**里程碑** M6

**依赖** 0002

## 目标

`gear build` 省略 `--output` 时，把阶段根写到宿主机的 `dist` 目录。

## 范围

- `--output` / `-o` 默认 `dist`。
- 显式 `--output` 仍覆盖默认值。
- 不按 `--target` 改目录。`dist` 只是宿主机路径。

## 实现要点

CLI 默认值和空的 `Request.Output` 都是 `dist`，对应 `--output type=local,dest=dist`。

## 验收标准

- [x] 默认 argv 含 `--output type=local,dest=dist`。
- [x] `--output out` 仍是 `dest=out`。
- [x] `gear build --help` 显示 `--output` 默认 `dist`。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-30 用假 `Runner` 验证，未启动 Docker：

- `TestBuildArgvDefaults`、`TestBuildRequiresBuildxAndKeepsExitCode`、`TestBuildCreatesMirrorBuilder` 的默认 argv 是 `type=local,dest=dist`。
- `TestBuildArgvOptionalFlags` 在 `--output out` 时仍是 `dest=out`。
- `go run . build --help` 显示 `--output` 默认 `dist`。
