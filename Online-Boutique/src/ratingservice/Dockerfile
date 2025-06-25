FROM golang:1.24.4-alpine AS builder
RUN apk add --no-cache ca-certificates git
RUN apk add build-base
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -gcflags="-N -l" -o /ratingservice .

FROM alpine AS release
RUN apk add --no-cache ca-certificates
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.25 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /src
COPY --from=builder /ratingservice /src/server
EXPOSE 8080
ENTRYPOINT ["/src/server"]