# Используем официальный образ Go
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы проекта
COPY . .

# Скачиваем зависимости
RUN go mod download

# Собираем приложение
RUN go build -o main ./cmd

# Используем минимальный образ для финального контейнера
FROM alpine:latest

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем бинарный файл из builder
COPY --from=builder /app/main .

# Копируем статические файлы (HTML, CSS, JS)
COPY ./web/views ./web/views

# Копируем сертификаты (если используются)
COPY cert.pem key.pem ./

# Открываем порт 8080
EXPOSE 8080

# Команда для запуска приложения
CMD ["./main"]