FROM golang:1.8.7

ADD . /go/src/github.com/adobe-platform/porter
WORKDIR /go/src/github.com/adobe-platform/porter

# Recompile everything and create a static binary
ENV CGO_ENABLED=0
CMD go build -v -a --installsuffix cgo -ldflags '-s' -o /host/porter
