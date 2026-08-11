# Go 1.26.5 builder；仅复制 cmd/internal，历史 PHP 源码不会进入运行镜像。
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

RUN go build -trimpath -ldflags="-s -w" -o /out/vmqfox-api ./cmd/vmqfox-api

# 精简运行时仍保留 CA、时区和 wget，分别用于 HTTPS 通知、时间处理和健康检查。
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