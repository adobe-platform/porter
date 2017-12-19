FROM golang:1.9.2

ADD . /go/src/github.com/adobe-platform/porter_test
WORKDIR /go/src/github.com/adobe-platform/porter_test

# Recompile everything and create a static binary
ENV CGO_ENABLED=0
RUN go build -v -a --installsuffix cgo -ldflags '-s' -o main

# Produce the docker context
CMD ["tar", "-c", "main", "Dockerfile.inet"]
