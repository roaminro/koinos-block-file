FROM golang:1.18-alpine as builder

ADD . /koinos-block-file
WORKDIR /koinos-block-file

RUN apk update && \
    apk add \
        gcc \
        musl-dev \
        linux-headers

RUN go get ./... && \
    go build -o koinos_block_file cmd/koinos-block-file/main.go

FROM alpine:latest
COPY --from=builder /koinos-block-file/koinos_block_file /usr/local/bin
ENTRYPOINT [ "/usr/local/bin/koinos_block_file" ]
