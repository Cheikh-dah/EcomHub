# Build stage
FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ecomhub ./cmd/server

# Run stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/ecomhub .
# Static assets are embedded in the binary in this project, 
# but if you have external ones, copy them here:
# COPY --from=builder /app/internal/web/static ./internal/web/static

EXPOSE 8080
ENTRYPOINT ["./ecomhub"]
