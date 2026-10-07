FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o worker ./cmd/worker && go build -o api ./cmd/api && go build -o queue ./cmd/queue

FROM golang:1.26-alpine AS goose
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

FROM alpine:latest AS migrate
RUN apk add --no-cache postgresql17-client
WORKDIR /app
COPY --from=goose /go/bin/goose /usr/local/bin/goose
COPY scripts/migrations ./migrations
COPY scripts/seed ./seed
COPY scripts/migrate.sh ./migrate.sh
CMD ["./migrate.sh"]

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/worker /app/api /app/queue ./
