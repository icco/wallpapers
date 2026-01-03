FROM golang:1.25 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go .
COPY cmd cmd
COPY db db
COPY analysis analysis

ENV CGO_ENABLED=1

RUN go build -ldflags="-s -w" -o /server ./cmd/server
RUN go build -ldflags="-s -w" -o /uploader ./cmd/uploader

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /server /usr/local/bin/server
COPY --from=builder /uploader /usr/local/bin/uploader
COPY wallpapers.db .

ENV PORT=8080
EXPOSE 8080

CMD ["server"]
