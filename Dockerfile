FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o epg-server ./cmd/epg/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/epg-server .
RUN mkdir -p /data

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["./epg-server"]
CMD ["-addr", ":8080", "-data-dir", "/data"]
