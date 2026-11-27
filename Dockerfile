FROM golang:1.25.6-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/api


FROM alpine:3.21

WORKDIR /app

RUN apk --no-cache add tzdata ca-certificates \
	&& addgroup -S appgroup \
	&& adduser -S appuser -G appgroup

COPY --from=builder /app/server .
COPY --from=builder /app/assets/ ./assets/
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 3000

CMD ["./server"]