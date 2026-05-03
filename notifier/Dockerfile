FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /notifier ./cmd/notifier && \
    CGO_ENABLED=0 GOOS=linux go build -o /migrator ./cmd/migrator

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /notifier .
COPY --from=builder /migrator .

RUN touch prod.env dev.env

EXPOSE 50051

CMD ["./notifier", "-env", "dev"]
