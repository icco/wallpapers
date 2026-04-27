FROM golang:1.26-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    imagemagick-dev \
    pkgconf

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ENV CGO_ENABLED=1

RUN go build -ldflags="-s -w" -o /server ./cmd/server
RUN go build -ldflags="-s -w" -o /uploader ./cmd/uploader

FROM alpine:3.23

LABEL org.opencontainers.image.source=https://github.com/icco/wallpapers
LABEL org.opencontainers.image.description="A site that lists the photos currently in my Wallpaper rotation."
LABEL org.opencontainers.image.licenses=MIT

RUN apk add --no-cache \
    ca-certificates \
    imagemagick-libs

# Create a non-root user.
RUN addgroup -S app && adduser -S -u 1001 -G app app

COPY --from=builder --chown=app:app /server /usr/local/bin/server
COPY --from=builder --chown=app:app /uploader /usr/local/bin/uploader

# Use a writable home directory so SQLite can create journal/WAL files alongside the DB.
WORKDIR /home/app
COPY --chown=app:app wallpapers.db .

USER app

ENV PORT=8080
EXPOSE 8080

CMD ["server"]
