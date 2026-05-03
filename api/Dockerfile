FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /migrator ./cmd/migrator && \
    CGO_ENABLED=0 GOOS=linux go build -o /seed ./cmd/seed

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api .
COPY --from=builder /migrator .
COPY --from=builder /seed .

RUN touch prod.env dev.env

EXPOSE 4010

CMD ["./api", "-env", "dev"]
