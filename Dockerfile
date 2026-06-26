FROM golang:1.26.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o hyperliquid-mcp ./cmd

FROM scratch
COPY --from=builder /app/hyperliquid-mcp /hyperliquid-mcp
ENTRYPOINT ["/hyperliquid-mcp"]
CMD ["--transport=http", "--addr=:8080"]
EXPOSE 8080
