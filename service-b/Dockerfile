FROM golang:1.21-alpine AS builder
WORKDIR /app

# Copia os arquivos go.mod e go.sum antes do restante do código
COPY go.mod go.sum ./

# Baixa as dependências necessárias
RUN go mod download

# Copia o restante dos arquivos da aplicação
COPY . .

# Constrói o executável
RUN go build -o service-b .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/service-b .
ENTRYPOINT ["./service-b"]
