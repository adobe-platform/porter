.PHONY: default builder build run test clean \
	login godep_reset godep_save \

default: release_darwin_local

PACKAGE_LIST := ./aws ./cfn ./cfn_template ./commands ./conf ./constants ./daemon ./files ./logger ./promote ./provision ./sample_project ./util

test: prebuild
	docker build -t porter-test -f Dockerfile.test .
	docker run -it --rm \
	-e TEST=true \
	porter-test

build_darwin: prebuild
	CGO_ENABLED=0 godep go build -a --installsuffix cgo -ldflags '-s' -o porter

build_linux:
	docker build -t porter -f Dockerfile.linux .
	docker run --rm -v $$PWD:/host porter

# Create a darwin build for production release
stage_darwin: build_darwin
	mkdir -p bin
	mv porter bin/porter_darwin386

# Create a linux build for production release
stage_linux: build_linux
	mkdir -p bin
	mv porter bin/porter_linux386

stage: stage_darwin stage_linux

# Create a darwin build and place it in ~/bin which should be in your $PATH
release_darwin_local:
	./release_porter upload dev
	mv bin/porter_darwin386 ~/bin/porter

# Push a release into production
release_PRODUCTION: clean
	./release_porter upload


prebuild: generate fmt vet

fmt:
	gofmt -s -w .

vet:
	go vet $$PACKAGE_LIST

generate:
	# TODO fix this to not include vendor
	- go generate ./...

clean:
	- rm -fr bin
	- rm -fr porter
	- rm -fr *.gz
	- rm -fr *.docker
	- find . -path "*_generated.go" -exec rm {} \;

# blow out any changes to the Godeps folder
godep_reset:
	git reset -- Godeps vendor
	git checkout -- Godeps vendor
	git clean -df Godeps vendor

godep_save:
	docker run -it --rm \
	-v $$PWD:/go/src/github.com/adobe-platform/porter \
	-w /go/src/github.com/adobe-platform/porter \
	golang:1.7.3 \
	go get github.com/tools/godep && \
	godep restore && \
	godep save ./...
