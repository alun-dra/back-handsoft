# ---- Build stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Cache de módulos
COPY go.mod go.sum ./
RUN go mod download

# Copiar código
COPY . .

# Instalar swag (solo en build)
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Intentar generar docs SIN romper build si falla
# Importante: generar en /tmp y solo reemplazar internal/docs si fue OK
RUN /go/bin/swag init -g cmd/api/main.go -o /tmp/docs \
  && rm -rf internal/docs \
  && mv /tmp/docs internal/docs \
  || echo "⚠️ Swagger docs generation failed. Using existing internal/docs (placeholder)."

# Compilar binario
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /app/bin/api ./cmd/api

# ---- Run stage ----
FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/bin/api /app/api

ENV PORT=8080
EXPOSE 8080

CMD ["/app/api"]