FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN apk add --no-cache git
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /subscriptions ./cmd/subscriptions

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /subscriptions /subscriptions
EXPOSE 8080
CMD ["/subscriptions"]
