# ============================================================
# Dockerfile — game_lounge_be (Go + Gin)
# Multi-stage build: compile di stage builder, copy binary
# ke image final alpine yang ringan.
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Download dependencies terlebih dahulu (cache layer)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/main.go

# ── Stage 2: Runtime ─────────────────────────────────────────
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/main .

ENV TZ=Asia/Jakarta

EXPOSE 8080

CMD ["./main"]
