## Developing `porter`

This is for porter developers.
Users of porter should checkout the [the documentation](docs)

### Engineering Goals

- Simplicity
- Resiliency
- Security
- Operational visibility
- Self-service

### Engineering Objectives

- Remember that everything will fail all the time - from AWS running out of
  `m3.xlarge` instances in `eu-west-1` to
  [sharks chewing the fibre connecting continents](http://www.theguardian.com/technology/2014/aug/14/google-undersea-fibre-optic-cables-shark-attacks)
- Know [the fallacies of distributed computing](https://en.wikipedia.org/wiki/Fallacies_of_distributed_computing)
- Don't make assumptions about deployment environments
- Don't create concepts that don't exist in AWS, Docker, or one of porter's
  other dependencies. If you don't create new concepts then you don't have to
  do the 2nd hardest thing in computer science: name them. Your concept is
  probably ill-defined anyway and will lead to confusion for everyone.
- Work backward from the problem
- Think about every log message you create as if you have no idea how porter
  works. Each message must be clear and actionable.

### Prerequisites

- Golang 1.6
- [godep](https://github.com/tools/godep#readme)
- Docker 1.7.1

The following are probably already installed which is part of why we use them -
they're ubiquitous. The versions likely won't matter but they're here for
reference incase of an issue.

- GNU Make 3.81
- Perl 5.18.2


#### Go

##### Download/install Go

```
curl -O https://storage.googleapis.com/golang/go1.6.darwin-amd64.pkg && open go1.6.darwin-amd64.pkg
```

Follow the instructions to install Go

##### Set your GOPATH

```
cd && mkdir go && export GOPATH=~/go
```

##### Install godep

`go get github.com/tools/godep`


#### Docker

##### Virtualbox

```
curl -O http://download.virtualbox.org/virtualbox/5.0.12/VirtualBox-5.0.12-104815-OSX.dmg && open VirtualBox-5.0.12-104815-OSX.dmg
```

Follow the instructions to install VirtualBox

##### docker-machine

Add `~/bin` to your `$PATH` and run

```
curl -L https://github.com/docker/machine/releases/download/v0.3.1/docker-machine_darwin-386 > ~/bin/docker-machine && chmod 0755 !#:4
```

##### Docker client

```
curl https://get.docker.com/builds/Darwin/x86_64/docker-1.7.1 > ~/bin/docker && chmod 0755 !#:3
```

##### Docker server

```
docker-machine create \
--driver virtualbox \
--virtualbox-memory 2048 \
--virtualbox-disk-size 20000 \
--virtualbox-boot2docker-url https://github.com/boot2docker/boot2docker/releases/download/v1.7.1/boot2docker.iso \
porter
```

Run `docker-machine env porter` and follow the instructions to set environment
variables.

##### Verify installation

Once complete you should be able to run `docker version` and see this output

```
[bcook:~]$ docker version
Client version: 1.7.1
Client API version: 1.19
Go version (client): go1.4.2
Git commit (client): 786b29d
OS/Arch (client): darwin/amd64
Server version: 1.7.1
Server API version: 1.19
Go version (server): go1.4.2
Git commit (server): 786b29d
OS/Arch (server): linux/amd64
```

#### Environment variables

Ensure you have an S3 bucket to upload the porter linux binary. Set
`PORTER_BIN_S3_BUCKET` to the bucket name and `PORTER_BIN_S3_BUCKET_REGION` to
the region for that bucket.

These environment variables are used by the `release_porter` script to

1. Bake a URL into the binary produced by the `release_porter` script
1. Upload a build to a S3 bucket you control
1. Form the URL porter places in EC2 UserData to download itself from (2)

Additionally `export DEV_MODE=1` to bypass version validation.

### Building porter

Type `make`

The default behavior is to build a static binary for Mac and place it in
`~/bin/` which should be in your `$PATH`. You know everything is setup correctly
when you do `make` followed by `which porter` and see that it lives at
`/Users/$(whoami)/bin/porter`

### Source layout

- `aws` - abstractions on the AWS SDK
- `cfn` - CloudFormation template struct definitions
- `cfn_template` - CloudFormation template creation
- `commands` - CLI commands. Since most of porter is driven from the CLI you
  should start here
- `daemon` - a stateful daemon that runs on EC2
- `files` - Non-Go files that are bundled with the porter binary
- `promote` - Instance promotion into ELB
- `provision` - High-level packaging and infrastructure provisioning APIs
- `prune` - GC

**DO NOT create files with `package main`**
