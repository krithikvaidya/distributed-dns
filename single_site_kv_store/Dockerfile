FROM golang:buster

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /kv_store

COPY . .

RUN go build .

EXPOSE 8080

CMD ["./distributed-dns"]