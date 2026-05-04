FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/quickstart ./cmd/quickstart

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata \
 && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/quickstart /app/quickstart
RUN mkdir -p /app/deploy
COPY deploy/config.yaml.example /app/deploy/config.yaml

ENV CONFIG=/app/deploy/config.yaml

EXPOSE 8080

RUN chown -R appuser:appuser /app

USER appuser

CMD ["/app/quickstart"]
