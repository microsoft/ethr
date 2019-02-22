FROM golang:1.11.3

WORKDIR /app

ADD ./ /app

RUN mkdir /out && \
    go get ./...
