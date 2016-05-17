.PHONY: default builder build run test clean \
	start_cluster login godeps_reset \

DOCKERIP := $$(docker-machine ip dev 2> /dev/null)

default: run

builder:
	go fmt ./...
	docker build -t porterd-builder -f Dockerfile.build .

build: builder
	docker run -e LOCAL=true --rm porterd-builder > builder.docker
	docker build -t porterd - < builder.docker

run: build start_cluster
	docker run \
	-it --rm \
	-p 9000:9000 \
	-e LOCAL_DEV=true \
	--link etcd-proxy:etcd \
	--add-host dockerhost:$(DOCKERIP) \
	porterd -p 9000

test:
	go fmt ./...
	docker build -t porterd-test -f Dockerfile.test .
	docker run -it --rm \
	-e LOCAL_DEV=true \
	-e TEST=true \
	porterd-test

clean:
	- rm -fr porterd
	- rm -fr .provisioning_directives.json
	- rm -fr .MANIFEST.json
	- rm -fr *.gz
	- rm -fr *.docker

start_cluster:
	./scripts/start_cluster

# Login to the builder container.
# This is useful to run go and godep commands (e.g. to add a 3rd-party lib)
login: builder
	docker run -it --rm \
	-v $$(pwd):/go/src/github.com/adobe-platform/porter/daemon \
	porterd-builder \
	bash

# blow out any changes to the Godeps folder
godeps_reset:
	git reset -- Godeps vendor
	git checkout -- Godeps vendor
	git clean -df Godeps vendor
