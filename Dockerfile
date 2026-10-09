FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/chat-go ./cmd/api

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/chat-go ./chat-go

EXPOSE 8080

CMD ["./chat-go"]
