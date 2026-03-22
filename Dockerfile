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

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    imagemagick-libs

COPY --from=builder /server /usr/local/bin/server
COPY --from=builder /uploader /usr/local/bin/uploader
COPY wallpapers.db .

ENV PORT=8080
EXPOSE 8080

CMD ["server"]
