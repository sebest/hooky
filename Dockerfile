FROM golang:1.4.2

ENV GOAPP github.com/sebest/hooky

RUN go get github.com/tools/godep

ADD . /go/src/${GOAPP}
WORKDIR /go/src/${GOAPP}

RUN godep go install ${GOAPP}/cmd/hookyd

EXPOSE 8000

ENTRYPOINT /go/bin/hookyd
