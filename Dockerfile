FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kleffd ./cmd/kleffd

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/kleffd /kleffd

ENTRYPOINT ["/kleffd"]
