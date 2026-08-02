# syntax=docker/dockerfile:1.7
# backend/Dockerfile — Multi-stage build for the Go API server
# Required build secret: docker build --secret id=github_token,src=/secure/path/to/token .
# Stage 1: build
FROM golang:1.25.12-alpine AS builder
WORKDIR /app
RUN apk --no-cache add git ca-certificates
ENV GOPRIVATE=github.com/cherryberryyogurt/* \
    GIT_TERMINAL_PROMPT=0
COPY go.mod go.sum ./
RUN --mount=type=secret,id=github_token,required=true \
    set -eu; \
    token="$(cat /run/secrets/github_token)"; \
    printf 'machine github.com login x-access-token password %s\n' "$token" > /root/.netrc; \
    chmod 600 /root/.netrc; \
    go mod download; \
    rm -f /root/.netrc
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
