# 1. Сборка Go приложения
FROM golang:1.25 AS builder

WORKDIR /app

# Скопировать go.mod и go.sum и скачать зависимости
COPY go.mod go.sum ./
RUN go mod download

# Скопировать код
COPY . .

# Собрать бинарник
RUN go build -o server main.go

# 2. Минимальный образ для запуска
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/server .

# Указываем порт (совпадает с docker-compose)
ENV SERVERPORT=8080
EXPOSE 8080

CMD ["./server"]
