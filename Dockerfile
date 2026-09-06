FROM golang:1.25-alpine AS builder

WORKDIR /src

# This image is the monorepo deployment path. Keep the two workspace modules
# present while dependencies are downloaded so the quickstart always builds
# against the go-modules source from this exact commit.
COPY go.mod go.sum go.work go.work.sum ./
COPY templates/quickstart/go.mod templates/quickstart/go.sum ./templates/quickstart/
RUN go mod download all

COPY foundation ./foundation
COPY modules ./modules
COPY templates/quickstart/cmd ./templates/quickstart/cmd
COPY templates/quickstart/internal ./templates/quickstart/internal
COPY templates/quickstart/deploy/config.yaml.example ./templates/quickstart/deploy/config.yaml.example

RUN cd templates/quickstart \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/quickstart ./cmd/quickstart

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata \
 && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/quickstart /app/quickstart
RUN mkdir -p /app/deploy
COPY --from=builder /src/templates/quickstart/deploy/config.yaml.example /app/deploy/config.yaml

ENV CONFIG=/app/deploy/config.yaml

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

RUN chown -R appuser:appuser /app

USER appuser

CMD ["/app/quickstart"]
