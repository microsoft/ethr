FROM golang:1.11.3

ADD ./ $GOPATH/src/Ethr

RUN mkdir /out

WORKDIR $GOPATH/src/Ethr

RUN go get -u github.com/golang/dep/cmd/dep 
RUN dep ensure -v 
