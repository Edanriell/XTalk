FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ../message .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chat-service ./chat-service

FROM alpine:3.20
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser/

COPY --from=builder /app/chat-service .

USER appuser

EXPOSE 50053

CMD ["./chat-service"]
