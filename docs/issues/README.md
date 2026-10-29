# Issues

实现顺序见 [docs/design.md](../design.md)。一项功能一份 issue。新增行为先加下一号 issue，再写代码。

## 里程碑

| 里程碑 | 内容 | 状态 |
| --- | --- | --- |
| M0 | 模块、CLI 骨架、Linux/macOS 限制、buildx 检查 | 已完成 |
| M1 | `gear build` 导出阶段 | 已完成 |
| M2 | `gear debug` 进入构建阶段 | 已完成 |
| M3 | Dockerfile 与 Makefile 构建二进制 | 已完成 |
| M4 | Linux 与 macOS 的多平台打包 | 已完成 |
| M5 | Docker Hub 镜像加速 | 已完成 |
| M6 | 默认导出到 `dist` | 已完成 |
| M7 | 打印 Dockerfile 阶段依赖 | 已完成 |
| M8 | 易读的阶段列表 | 已完成 |

## Issue

| 编号 | 标题 | 依赖 | 状态 |
| --- | --- | --- | --- |
| [0001](0001-project-skeleton.md) | 项目骨架 | 无 | 已完成 |
| [0002](0002-build-export.md) | 导出构建阶段 | 0001 | 已完成 |
| [0003](0003-debug-shell.md) | 进入构建阶段 | 0001 | 已完成 |
| [0004](0004-build-files.md) | 构建入口 | 0001 | 已完成 |
| [0005](0005-multi-platform-dist.md) | 多平台打包 | 0004 | 已完成 |
| [0006](0006-registry-mirror.md) | Docker Hub 镜像加速 | 0002 | 已完成 |
| [0007](0007-default-output-dist.md) | 默认导出到 dist | 0002 | 已完成 |
| [0008](0008-show-targets.md) | 打印阶段依赖 | 0001 | 已完成 |
| [0009](0009-readable-targets.md) | 易读的阶段列表 | 0008 | 已完成 |

## 约定

- 每个 issue 含目标、范围、实现要点、验收标准。做完后勾上验收，并写完成情况。
- 测试用假 `Runner` 核对传给 `docker` 的参数，不启动 Docker。
- 退出码沿用 `docker buildx`。`ExitError.ExitCode` 交给 `urfave/cli`。
