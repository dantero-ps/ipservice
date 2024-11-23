FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ipservice ./cmd/ipservice

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/ipservice .

EXPOSE 8080

CMD ["./ipservice"]