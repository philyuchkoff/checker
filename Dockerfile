FROM golang:1.21-alpine AS builder
WORKDIR /app

# Копируем только файлы модулей сначала
COPY go.mod go.sum ./

# Загружаем зависимости с проверкой
RUN if [ -f go.sum ]; then \
    GOPROXY=https://proxy.golang.org,direct go mod download; \
    else \
    echo "No go.sum - skipping download"; \
    fi

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:3.18
COPY --from=builder /app/app .
CMD ["./app"]
