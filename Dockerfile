# 阶段一：从 web 源码构建 React 静态资源。
# 构建产物直接落在 internal/http/static/out，供下一阶段的 go:embed 读取，
# 避免镜像依赖本地未跟踪的旧构建产物。
FROM node:24-alpine AS web-builder

WORKDIR /web

# 使用仓库锁定的 pnpm 版本，保证 CI 与镜像安装同一套依赖树
RUN corepack enable && corepack prepare pnpm@11.22.0 --activate

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

# 构建后校验 index.html 引用的每个本地资源都真实存在，
# 防止把缺少 JS/CSS 的空壳前端打进镜像。
RUN set -eu; \
    out=/internal/http/static/out; \
    test -f "$out/index.html" || { echo "构建产物缺少 index.html"; exit 1; }; \
    grep -oE '(src|href)="/assets/[^"]+"' "$out/index.html" \
      | sed -E 's#^[a-z]+="/assets/(.*)"$#\1#' \
      | sort -u > /tmp/referenced-assets.txt; \
    test -s /tmp/referenced-assets.txt || { echo "index.html 未引用任何 assets 资源"; exit 1; }; \
    while read -r asset; do \
        test -f "$out/assets/$asset" || { echo "index.html 引用的资源缺失: $asset"; exit 1; }; \
        echo "已校验前端资源: $asset"; \
    done < /tmp/referenced-assets.txt

# 阶段二：Go 1.26.5 builder；仅复制 cmd/internal，历史 PHP 源码不会进入运行镜像。
FROM golang:1.26.5-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=mod \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

# 用本次构建的前端资源覆盖仓库中的占位目录，确保 embed 的一定是当前源码产物
RUN rm -rf internal/http/static/out
COPY --from=web-builder /internal/http/static/out ./internal/http/static/out

RUN go build -trimpath -ldflags="-s -w" -o /out/vmqfox-api ./cmd/vmqfox-api

# 阶段三：精简运行时仍保留 CA、时区和 wget，分别用于 HTTPS 通知、时间处理和健康检查。
FROM alpine:3.22 AS runtime

ARG TZ=Asia/Shanghai
ENV TZ=${TZ}

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S vmqfox \
    && adduser -S -G vmqfox vmqfox

COPY --from=builder /out/vmqfox-api /usr/local/bin/vmqfox-api
COPY entrypoint.sh /usr/local/bin/vmqfox-entrypoint
RUN chmod 0555 /usr/local/bin/vmqfox-entrypoint /usr/local/bin/vmqfox-api

USER vmqfox:vmqfox
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/vmqfox-entrypoint"]
CMD ["/usr/local/bin/vmqfox-api"]
