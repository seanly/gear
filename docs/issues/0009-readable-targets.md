# 0009 易读的阶段列表

**里程碑** M8

**依赖** 0008

## 目标

`gear show targets` 的输出按阶段分组，用 Dockerfile 里的写法表示依赖，读的时候不用对照列名。

## 范围

- 每个阶段先写名字，空一行再写下个阶段。
- 依赖缩进两格。`FROM` 外部镜像和 `FROM` 前一阶段都写成 `FROM <name>`。
- `COPY --from` 写成 `COPY --from <stage>`。`RUN --mount` 的 `from` 写成 `RUN --mount from <stage>`。
- `ONBUILD` 里的依赖在同一阶段下，行首加 `ONBUILD`。
- 解析规则、失败条件和 `--file` 不变。

## 实现要点

[examples/Dockerfile](../../examples/Dockerfile) 的输出：

```text
build
  FROM alpine:3.20

result
  FROM scratch
  COPY --from build
```

## 验收标准

- [x] `examples/Dockerfile` 与根 `Dockerfile` 按上面的格式分组。
- [x] 无名阶段、序号、大小写、`ARG`、`ONBUILD`、`RUN --mount` 仍正确，只是换成这种写法。
- [x] 缺文件、没有 `FROM`、`INCLUDE`、`ADD --from` 的失败不变。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-30 用字符串和仓库里的 Dockerfile 验证，未启动 Docker：

- `TestShowTargetsExamples` 和 `TestShowTargetsRootDockerfile` 核对分组输出。
- `TestShowTargetsSyntax` 覆盖无名阶段、序号、大小写、`ARG`、`ONBUILD COPY --from` 和 `RUN --mount from`。
- `TestShowTargetsErrors` 仍覆盖没有 `FROM`、`INCLUDE`、`ADD --from` 和缺文件。
- `go test ./internal/gear` 通过。`go run . show targets --file examples/Dockerfile` 打印阶段名和缩进的 `FROM`、`COPY --from`。
