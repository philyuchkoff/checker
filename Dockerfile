FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:3.18
COPY --from=builder /app/app .
CMD ["./app"]
