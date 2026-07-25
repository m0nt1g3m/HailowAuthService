FROM golang:1.26-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

RUN go install github.com/go-task/task/v3/cmd/task@v3.31.0
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

COPY . .

RUN task build_proto
RUN CGO_ENABLED=0 GOOS=linux go build -o /HailowAuthService ./cmd/server/main.go

FROM debian:12-slim

WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates netcat-openbsd && rm -rf /var/lib/apt/lists/*

COPY --from=builder /HailowAuthService /HailowAuthService
COPY --from=builder /go/bin/task /usr/local/bin/task
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY Taskfile.yml /app/Taskfile.yml
COPY migrations /app/migrations

RUN printf '#!/bin/sh\nset -e\ncd /app\nif [ -z "$DATABASE_URL" ]; then echo "DATABASE_URL is required" >&2; exit 1; fi\nurl="${DATABASE_URL#*://}"\nhostport="${url#*@}"\nDB_HOST="${hostport%%:*}"\nDB_PORT="${hostport#*:}"\nDB_PORT="${DB_PORT%%/*}"\necho "Waiting for database at $DB_HOST:$DB_PORT"\nwhile ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; do echo "Database not ready, retrying..."; sleep 1; done\ntask migrate_up\nexec /HailowAuthService\n' > /app/entrypoint.sh \
    && chmod +x /app/entrypoint.sh

EXPOSE 8080

ENV ADDR=0.0.0.0
ENV PORT=8080

ENTRYPOINT ["/app/entrypoint.sh"]
