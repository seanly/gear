# 0006 Docker Hub 镜像加速

**里程碑** M5

**依赖** 0002

## 目标

Docker 守护进程没有 registry mirror 时，`gear build` 和 `gear debug` 仍可以把 Docker Hub 的拉取转到指定镜像站。

## 范围

- `--registry-mirror`，也可由 `GEAR_REGISTRY_MIRROR` 提供。
- 值是镜像站主机，例如 `mirror.example.com`。`https://` 会去掉，`http://` 按明文登记。
- 只影响 `docker.io`。已经写了其他仓库地址的 `FROM` 保持原样。
- 不修改用户的默认 buildx builder，也不改 Docker 守护进程配置。

## 实现要点

`docker buildx build` 没有镜像加速参数。GEAR 在需要时创建名为 `gear-<host>` 的 `docker-container` builder，并写入：

```toml
[registry."docker.io"]
  mirrors = ["mirror.example.com"]
```

构建命令加上 `--builder gear-mirror-example-com`。builder 镜像从镜像站拉取：`mirror.example.com/moby/buildkit:buildx-stable-1`，这样创建 builder 本身也不走 Docker Hub。同名 builder 已存在时直接复用。

## 验收标准

- [x] `--registry-mirror=mirror.example.com` 生成的 BuildKit 配置把 `docker.io` 指到该主机。
- [x] 创建命令使用 `docker-container`，builder 镜像来自该镜像站。
- [x] 构建 argv 带 `--builder gear-mirror-example-com`。
- [x] builder 已存在时不再执行 `buildx create`。
- [x] 带路径的值会被拒绝。
- [x] 测试不启动 Docker。

## 完成情况

2026-10-29 用假 `Runner` 验证，未创建 builder，也未启动 Docker：

- `TestParseMirror` 接受 `https://mirror.example.com/`，拒绝带路径的值，并为 `http://` 写上 `http = true`。
- `TestBuildCreatesMirrorBuilder` 核对 toml、`buildx create` 和带 `--builder` 的构建 argv。
- `TestBuildReusesMirrorBuilder` 核对已有 builder 时不会再次创建。
- 另外确认镜像站 `https://mirror.example.com/v2/library/golang/manifests/1.25` 与 `moby/buildkit:buildx-stable-1` 都返回 200。本次没有实际执行 `docker buildx create`。
