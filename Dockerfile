FROM golang:1.24-alpine AS builder
ARG RELEASE_REVISION
ARG SOURCE_ARCHIVE_SHA256
RUN printf '%s' "$RELEASE_REVISION" | grep -Eq '^[0-9a-f]{40}$'
RUN printf '%s' "$SOURCE_ARCHIVE_SHA256" | grep -Eq '^[0-9a-f]{64}$'
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN grep -Fxq "$RELEASE_REVISION" release-source-revision
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-X main.releaseRevision=${RELEASE_REVISION}" -o server ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o crawler ./cmd/crawler/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o monitor-check ./cmd/monitor-check/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-X main.releaseRevision=${RELEASE_REVISION}" -o provider-cutover-preflight ./cmd/provider-cutover-preflight/

FROM alpine:3.19
ARG RELEASE_REVISION
ARG SOURCE_ARCHIVE_SHA256
LABEL org.opencontainers.image.revision=$RELEASE_REVISION \
      org.opencontainers.image.source_archive_sha256=$SOURCE_ARCHIVE_SHA256
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/crawler .
COPY --from=builder /app/monitor-check .
COPY --from=builder /app/provider-cutover-preflight .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/migrations ./migrations
ENV APP_ROOT=/app
EXPOSE 8091
CMD ["./server"]
