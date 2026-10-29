# syntax=docker/dockerfile:1

FROM golang:1.25 AS build
WORKDIR /src
ENV CGO_ENABLED=0
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=0.0.0-dev
ARG TARGETOS
ARG TARGETARCH
RUN set -eu; \
    if [ -n "${TARGETOS}" ] && [ -n "${TARGETARCH}" ]; then \
      export GOOS="${TARGETOS}" GOARCH="${TARGETARCH}"; \
    fi; \
    go build -ldflags "-s -w -X main.VERSION=${VERSION}" -o /out/gear .

FROM scratch AS result
COPY --from=build /out/gear /gear

FROM build AS packages
ARG VERSION=0.0.0-dev
ARG PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"
RUN set -eu; \
    mkdir -p /packages; \
    for spec in ${PLATFORMS}; do \
      os=$${spec%/*}; \
      arch=$${spec#*/}; \
      stage=$$(mktemp -d); \
      CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
        go build -ldflags "-s -w -X main.VERSION=${VERSION}" -o $$stage/gear .; \
      tar -C $$stage -czf /packages/gear_${VERSION}_$${os}_$${arch}.tar.gz gear; \
      rm -rf $$stage; \
    done

FROM scratch AS dist
COPY --from=packages /packages/ /
