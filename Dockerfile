FROM golang

ADD . /go/src/github.com/sebest/hooky

RUN go get github.com/sebest/hooky/cmd/hookyd

RUN go install github.com/sebest/hooky/cmd/hookyd

ENTRYPOINT /go/bin/hookyd

EXPOSE 8000
