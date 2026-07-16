# Multi-stage build: compile with Go, run in a small image
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Copy go.mod/go.sum first to cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy project and build
COPY . ./
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

# Final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /usr/local/bin/server
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
