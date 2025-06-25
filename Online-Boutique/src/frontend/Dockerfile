FROM golang:1.24.4-alpine AS builder
RUN apk add --no-cache ca-certificates git
RUN apk add build-base
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 构建前端服务，不是评分服务
RUN go build -gcflags="-N -l" -o /frontend .

FROM alpine AS release
RUN apk add --no-cache ca-certificates
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.25 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /src
# 复制前端服务二进制文件
COPY --from=builder /frontend /src/server
# 复制静态文件和模板
COPY --from=builder /src/static ./static
COPY --from=builder /src/templates ./templates
EXPOSE 8080
ENTRYPOINT ["/src/server"]
