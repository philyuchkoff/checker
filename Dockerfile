# Этап 1: Сборка приложения
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Сначала копируем только файлы модулей
COPY go.mod .
COPY go.sum .

# Проверяем наличие go.sum (опционально)
RUN test -f go.sum || (echo "Error: go.sum not found!" && exit 1)

RUN go mod download

# Копируем остальные файлы
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o speedmon .

# Этап 2: Финальный образ
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/speedmon .
EXPOSE 8080
CMD ["./speedmon"]
