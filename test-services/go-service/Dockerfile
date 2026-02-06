# Build context: корень проекта (dephealth/)
# docker build -t ... -f test-services/go-service/Dockerfile .

FROM golang:1.25-alpine AS builder

WORKDIR /build

# Копируем SDK (для replace directive)
COPY sdk-go/ ./sdk-go/

# Зависимости
COPY test-services/go-service/go.mod test-services/go-service/go.sum ./test-services/go-service/
WORKDIR /build/test-services/go-service
RUN go mod download

# Исходники
COPY test-services/go-service/main.go ./

# Сборка
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app .

# ---
FROM alpine:3.21

RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app

EXPOSE 8080
ENTRYPOINT ["/app"]
