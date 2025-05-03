FROM golang:1.20-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gospeak ./cmd/main.go

FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

RUN adduser -D -g '' gospeak

WORKDIR /home/gospeak

COPY --from=builder /app/gospeak .

COPY --from=builder /app/web ./web
COPY --from=builder /app/config ./config

ENV PORT=8080
ENV GIN_MODE=release

RUN chown -R gospeak:gospeak /home/gospeak

USER gospeak

EXPOSE 8080

# Запуск приложения
CMD ["./gospeak"]