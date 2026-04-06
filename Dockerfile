FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o aggregator .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /build/aggregator .

ENTRYPOINT ["./aggregator"]
