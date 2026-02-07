FROM harbor.kryukov.lan/homelab/golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /grpc-stub .

FROM harbor.kryukov.lan/homelab/alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /grpc-stub /grpc-stub

EXPOSE 9090 8080
ENTRYPOINT ["/grpc-stub"]
