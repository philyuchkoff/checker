# Этап сборки
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Статическая сборка (без зависимостей)
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Финальный образ
FROM alpine:3.18
WORKDIR /app

# Копируем бинарник
COPY --from=builder /app/app .

# Опционально: добавляем сертификаты для HTTPS-запросов
#RUN apk add --no-cache ca-certificates

# Запуск приложения
CMD ["./app"]
