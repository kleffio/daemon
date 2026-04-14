# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy module manifests first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy full source and build. modernc/sqlite is pure Go — CGO_ENABLED=0 is safe.
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /kleffd \
    ./cmd/kleffd/

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S -G app app

# Create the data directory the daemon writes its SQLite DB into,
# owned by the app user before we drop privileges.
RUN mkdir -p /var/lib/kleffd/data && chown -R app:app /var/lib/kleffd

COPY --from=builder /kleffd /kleffd

USER app
WORKDIR /var/lib/kleffd

ENTRYPOINT ["/kleffd"]
