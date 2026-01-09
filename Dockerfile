# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum сначала для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Собираем бинарник
RUN go build -o api cmd/api/main.go

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Копируем бинарник
COPY --from=builder /app/api .

# Папки для данных
RUN mkdir -p uploads

EXPOSE 3001

CMD ["./api"]