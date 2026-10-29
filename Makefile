.PHONY: help build test image dist dist-image clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
LDFLAGS := -s -w -X main.VERSION=$(VERSION)

# 全平台是 GEAR 能编译的范围：Linux 与 macOS 的 amd64、arm64。
# 单平台：make dist PLATFORM=linux/amd64
PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
DIST_PLATFORMS := $(if $(PLATFORM),$(PLATFORM),$(PLATFORMS))

.DEFAULT_GOAL := build

help: ## 列出可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## 在本机编译 bin/gear
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/gear .

test: ## 运行测试
	go test ./...

image: ## 用 docker buildx 导出一个 Linux 二进制到 bin/gear
	mkdir -p bin
	docker buildx build \
		--file Dockerfile \
		--target result \
		$(if $(PLATFORM),--platform $(PLATFORM),) \
		--build-arg VERSION=$(VERSION) \
		--output type=local,dest=bin \
		.

dist: ## 打包。默认全平台；PLATFORM=darwin/arm64 只打一个
	rm -rf dist
	mkdir -p dist
	@set -eu; \
	for spec in $(DIST_PLATFORMS); do \
		os=$${spec%/*}; \
		arch=$${spec#*/}; \
		stage=$$(mktemp -d); \
		echo "==> $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" -o $$stage/gear .; \
		tar -C $$stage -czf dist/gear_$(VERSION)_$${os}_$${arch}.tar.gz gear; \
		rm -rf $$stage; \
	done

dist-image: ## 在容器里交叉编译并导出到 dist/
	rm -rf dist
	mkdir -p dist
	docker buildx build \
		--file Dockerfile \
		--target dist \
		--build-arg VERSION=$(VERSION) \
		--build-arg PLATFORMS="$(DIST_PLATFORMS)" \
		--output type=local,dest=dist \
		.

clean: ## 删除 bin/ 和 dist/
	rm -rf bin dist
