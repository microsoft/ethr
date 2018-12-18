FROM golang:1.11.3

ADD ./ $GOPATH/src/ethr

RUN mkdir /out

WORKDIR $GOPATH/src/ethr

RUN go get -u github.com/golang/dep/cmd/dep 
RUN dep ensure -v 
