# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go .
COPY cmd cmd

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /uploader ./cmd/uploader

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache ca-certificates

COPY --from=builder /server /usr/local/bin/server
COPY --from=builder /uploader /usr/local/bin/uploader

ENV PORT=8080
EXPOSE 8080

CMD ["server"]
