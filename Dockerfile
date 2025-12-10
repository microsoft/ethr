FROM golang:1.25

WORKDIR /app

ADD ./ /app

RUN mkdir /out && \
    go build .
