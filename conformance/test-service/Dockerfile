ARG REGISTRY=docker.io
FROM ${REGISTRY}/golang:1.25-alpine AS builder

WORKDIR /build

# Копируем SDK
COPY sdk-go/ ./sdk-go/

# Копируем conformance test service
COPY conformance/test-service/go.mod conformance/test-service/go.sum ./conformance/test-service/
WORKDIR /build/conformance/test-service
RUN go mod download

COPY conformance/test-service/main.go .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app .

ARG REGISTRY=docker.io
FROM ${REGISTRY}/alpine:3.21
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
