# 0008 打印阶段依赖

**里程碑** M7

**依赖** 0001

## 目标

`gear show targets` 读本地 Dockerfile，按文件顺序列出阶段，以及阶段之间的 `FROM`、`COPY`、`ADD`、`RUN --mount` 依赖。

## 范围

- 子命令是 `gear show targets`。`show` 下面只有 `targets`。
- `--file` / `-f` 默认 `Dockerfile`，与 `build` / `debug` 共用。不接受其它构建参数。
- 不调用 `docker`，不导入 `dockerfile2llb`。
- 解析使用 `github.com/moby/buildkit` v0.34.1 的 `parser` 和 `instructions`。
- 模块 Go 版本升到 1.26.8。根目录 Dockerfile 的基础镜像改为 `golang:1.26`。
- 标准输出每个关系一行，制表符分隔：阶段、关系、对象。
- `base` 是外部镜像。`from` 是前面的阶段。`copy` / `mount` 是对应指令里的阶段引用。无名阶段写成 `#0` 这种序号。阶段序号解析成该阶段的名字。
- BuildKit v0.34.1 不解析 `ADD --from`，该写法按未知 flag 失败。
- 阶段名比对大小写不敏感，输出保持文件里的写法。
- 文件里写了默认值、并且在当前作用域可见的 `ARG` 会展开。没有默认值的 `${VAR}` 和 `TARGETARCH` 这类构建期变量保持原文。不读 `--build-arg`。
- `ONBUILD` 里的 `COPY` / `ADD` / `RUN` 计入当前阶段。
- `--from` 像镜像引用（含 `:`、`/` 或 `@`）且不是已有阶段名时，不记成阶段依赖。对不上的名字仍打印。
- `INCLUDE` 和其他未知指令让解析失败。缺文件、没有 `FROM`、解析失败都退出非 0。

## 实现要点

`parser.Parse` 得到 AST，`instructions.Parse` 得到阶段和 `FROM` 之前的 meta `ARG`。`FROM` 之前的 `ARG` 用于各条 `FROM`；阶段内的 `ARG` 只作用于该阶段之后的指令。阶段内重新声明、且不写默认值的 `ARG` 继承 meta `ARG` 的默认值。`ONBUILD` 的触发指令再解析一次，依赖记在当前阶段上。

## 验收标准

- [x] `examples/Dockerfile` 与根 `Dockerfile` 的行符合上面的格式。
- [x] 无名阶段是 `#0`，`COPY --from=0` 解析成该阶段名。
- [x] `FROM` 前一阶段、大小写不同的阶段名、`--from` 不在第一个 flag 都正确。
- [x] 注释、续行、heredoc 内的假 `FROM` 被忽略。
- [x] 带默认值的 `ARG` 被展开，无默认值的 `${VAR}` 保留原文。
- [x] `ONBUILD COPY --from` 计入依赖。`INCLUDE` 或未知指令失败。
- [x] 缺文件与无 `FROM` 失败，缺文件的错误含 `gear show targets --file <Dockerfile>`。
- [x] `gear show targets --help` 显示 `--file` 默认 `Dockerfile`，以及 `gear show targets` 和 `gear show targets --file examples/Dockerfile`。
- [x] `go.mod` 的 Go 版本是 1.26.8。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-30 用字符串和仓库里的 Dockerfile 验证，未启动 Docker：

- `TestShowTargetsExamples`、`TestShowTargetsRootDockerfile` 核对 `examples/Dockerfile` 与根 `Dockerfile`。
- `TestShowTargetsSyntax` 覆盖无名阶段、序号、大小写、flag 顺序、注释、续行、三种 heredoc、`# escape=`、`ARG` 默认值、未设置变量、阶段作用域、meta `ARG` 重新声明、`ONBUILD` 和 `--mount`。
- `TestShowTargetsErrors` 覆盖没有 `FROM`、`INCLUDE`、`ADD --from`（BuildKit v0.34.1 报 `unknown flag`）和缺文件。
- `go test ./...` 通过。`go run . show targets --help` 显示 `--file` 默认 `Dockerfile` 和两条例子。`go.mod` 是 Go 1.26.8，直接依赖 `github.com/moby/buildkit v0.34.1`。
