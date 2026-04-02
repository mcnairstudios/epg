FROM golang:1.25.1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o epg-server ./cmd/epg/main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/epg-server .
RUN mkdir -p /data

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["./epg-server"]
CMD ["-addr", ":8080", "-data-dir", "/data", "-max-connections", "3", "-rolling", "-rolling-parallel", "3"]
