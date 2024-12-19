FROM golang:1.20-bullseye as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build t go build -a -o /telegram_bot ./cmd/application

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /telegram_bot /telegram_bot

ENTRYPOINT ["/telegram_bot"]
