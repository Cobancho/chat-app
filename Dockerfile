FROM --platform=linux/arm64/v8 golang:1.22 AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o /go/bin/chat

FROM --platform=linux/arm64/v8 scratch

EXPOSE 8080

ENV PORT 8080

COPY --from=builder /go/bin/chat /chat
COPY --from=builder /app/.env /
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/chat"]
