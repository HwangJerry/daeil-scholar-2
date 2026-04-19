# backend/Dockerfile — Multi-stage build for the Go API server
# Stage 1: build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: run
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
# uploads 디렉토리 생성 (PVC mount 전 기본값)
RUN mkdir -p /var/www/uploads /var/www/legacy/files
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
