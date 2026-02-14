ARG REGISTRY=docker.io
FROM --platform=$BUILDPLATFORM ${REGISTRY}/golang:1.25-alpine AS builder

ARG TARGETOS TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /grpc-stub .

ARG REGISTRY=docker.io
FROM ${REGISTRY}/alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /grpc-stub /grpc-stub

EXPOSE 9090 8080
ENTRYPOINT ["/grpc-stub"]
