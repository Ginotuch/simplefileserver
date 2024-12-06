FROM golang:1.20 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o simplefileserver ./cmd/simplefileserver

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/simplefileserver /app/

# Copy in your certificate/key if you have them (for local testing, you can mount them in docker-compose)
# For demonstration, assume they come from docker-compose volumes.

ENTRYPOINT ["/app/simplefileserver"]
