FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o simulator .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/simulator .
# The web UI is served from disk, so the static assets must ship with the binary
COPY --from=builder /app/web ./web
# OCPI partner profiles and the event log live here; mount a volume to keep them
RUN mkdir -p /app/data
VOLUME ["/app/data"]
EXPOSE 8080
CMD ["./simulator"]
