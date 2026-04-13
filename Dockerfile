FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -o crawler ./cmd/crawler/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/crawler .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/migrations ./migrations
ENV APP_ROOT=/app
EXPOSE 8091
CMD ["./server"]
