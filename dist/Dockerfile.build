FROM golang:1.4

ENV CGO_ENABLED 0
ENV GOOS linux
ENV HOOKY_DIR /go/src/github.com/sebest/hooky

RUN mkdir -p $HOOKY_DIR
WORKDIR $HOOKY_DIR

ADD . $HOOKY_DIR

RUN go get github.com/tools/godep
RUN godep go build -a -installsuffix cgo -o hookyd cmd/hookyd/main.go
RUN godep go build -a -installsuffix cgo -o hooky cmd/hooky/main.go

CMD tar -czf - hooky hookyd
