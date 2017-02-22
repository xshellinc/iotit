FROM golang:latest

RUN mkdir -p /go/src/github.com/xshellinc/iotit
WORKDIR /go/src/github.com/xshellinc/iotit
ADD . /go/src/github.com/xshellinc/iotit
RUN go get github.com/laher/goxc
RUN go get ./...
