FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rate-limiter ./cmd/app

FROM scratch
COPY --from=builder /app/rate-limiter /app/rate-limiter
COPY --from=builder /app/cmd/app/.env /app/.env
WORKDIR /app
ENTRYPOINT ["/app/rate-limiter"]