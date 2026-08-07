FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o message-service ./services/message

FROM alpine:3.20
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser/

COPY --from=builder /app/message-service .

USER appuser

EXPOSE 50054

CMD ["./message-service"]
