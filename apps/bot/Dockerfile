# Estágio de compilação (Builder)
FROM golang:alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Cache de dependências do Go
COPY go.mod go.sum ./
RUN go mod download

# Copia código fonte
COPY . .

# Compila o binário estático do Bot
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/bot ./cmd/bot

# Estágio final de execução (Runner)
FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/bot /usr/local/bin/bot

USER appuser

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/bot"]
