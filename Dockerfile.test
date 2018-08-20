FROM golang:1.8.7

RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get github.com/onsi/gomega

ADD . /go/src/github.com/adobe-platform/porter
WORKDIR /go/src/github.com/adobe-platform/porter

CMD ginkgo -r -p
