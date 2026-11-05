# 构建阶段使用宿主架构编译器，交叉编译目标 Linux 可执行文件。
ARG BASE_REGISTRY=docker.io/library
FROM --platform=$BUILDPLATFORM ${BASE_REGISTRY}/golang:1.24.5-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/bootstrap ./cmd/bootstrap

FROM ${BASE_REGISTRY}/alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 app && adduser -D -u 10001 -G app app \
    && mkdir -p /app/uploads && chown app:app /app/uploads
WORKDIR /app
COPY --from=build /out/ /app/bin/
COPY backend/migrations/ /app/migrations/
COPY frontend/ /app/frontend/
COPY --chmod=755 deploy/entrypoint.sh /app/entrypoint.sh
ENV APP_ADDR=:8080 FRONTEND_DIR=/app/frontend UPLOAD_DIR=/app/uploads TZ=Asia/Shanghai
USER app
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
