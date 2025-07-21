# Start from the latest golang base image
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o weather-api-redis main.go

# Start a new stage from scratch
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/weather-api-redis .
COPY config.yaml .
COPY .env .
EXPOSE 8080
CMD ["./weather-api-redis"] 