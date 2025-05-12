# Используем многоэтапную сборку
# Этап 1: Сборка приложения
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем файлы модулей и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o speedmon .

# Этап 2: Создаем минимальный образ
FROM alpine:3.18

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /app/speedmon .

# Устанавливаем tzdata для корректного времени
RUN apk add --no-cache tzdata

# Открываем порт для метрик
EXPOSE 8080

# Запускаем приложение с флагами по умолчанию
CMD ["./speedmon"]
