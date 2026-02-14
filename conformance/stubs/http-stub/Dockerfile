ARG REGISTRY=docker.io
FROM --platform=$BUILDPLATFORM ${REGISTRY}/golang:1.23-alpine AS builder

ARG TARGETOS TARGETARCH

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /http-stub .

ARG REGISTRY=docker.io
FROM ${REGISTRY}/alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /http-stub /http-stub

EXPOSE 8080
ENTRYPOINT ["/http-stub"]
