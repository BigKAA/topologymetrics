FROM harbor.kryukov.lan/docker/golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /http-stub .

FROM harbor.kryukov.lan/docker/alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /http-stub /http-stub

EXPOSE 8080
ENTRYPOINT ["/http-stub"]
