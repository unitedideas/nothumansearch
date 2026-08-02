FROM golang:1.24-alpine AS builder
ARG RELEASE_REVISION
RUN printf '%s' "$RELEASE_REVISION" | grep -Eq '^[0-9a-f]{40}$'
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN grep -Fxq "$RELEASE_REVISION" release-source-revision
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-X main.releaseRevision=${RELEASE_REVISION}" -o server ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o crawler ./cmd/crawler/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o monitor-check ./cmd/monitor-check/

FROM alpine:3.19
ARG RELEASE_REVISION
LABEL org.opencontainers.image.revision=$RELEASE_REVISION
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/crawler .
COPY --from=builder /app/monitor-check .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/migrations ./migrations
ENV APP_ROOT=/app
EXPOSE 8091
CMD ["./server"]
