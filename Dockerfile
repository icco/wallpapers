From golang:1.22-alpine

ENV PORT 8080
EXPOSE 8080

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go .
COPY cmd cmd

RUN go build -v -o /usr/local/bin/server ./cmd/server
RUN go build -v -o /usr/local/bin/uploader ./cmd/uploader

CMD ["server"]
