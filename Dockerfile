# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Dependencias del sistema (git puede ser necesario para algunos módulos)
RUN apk add --no-cache git ca-certificates

# Cache de módulos
COPY go.mod go.sum ./
RUN go mod download

# Copiar código
COPY . .

# Compilar binario (estático)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app/bin/api ./cmd/api

# ---- Run stage ----
FROM alpine:3.20

WORKDIR /app

# Certificados (por HTTPS, drivers, etc.)
RUN apk add --no-cache ca-certificates

# Copiar binario
COPY --from=builder /app/bin/api /app/api

# Railway inyecta PORT; tu app debe usarlo (ya lo haces en cfg.Port).
ENV PORT=8080

EXPOSE 8080

# Healthcheck local opcional (Railway hace su propio healthcheck con PORT)
# HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1

CMD ["/app/api"]