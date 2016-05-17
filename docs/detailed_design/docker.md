Docker
======

porter ships Docker containers.

A simple service only needs a `Dockerfile` but porter enables some interesting
optimizations and use cases available in Docker.

The Docker context
------------------

The `docker build ...` command's context is, by default, the filesystem and
experienced Docker users quickly find out they need to add to a `.dockerignore`
file to prevent the context from bloating and slowing down build times.
`.dockerignore` is a blacklist.

`docker build ...` alternatively accepts as its context a tar stream on STDIN.
porter looks for a `Dockerfile.build`. If found it will build this image and run
it, piping the STDOUT of it into the STDIN of `docker build -f Dockerfile`

The exact sequence goes something like this for a service `foo`

```
docker build -t foo-builder -f Dockerfile.build .
docker run -it --rm foo-builder | docker build -t foo -
```

This enables some interesting use cases, described below.

Use cases
---------

### #1 A whitelist

The simplest use case would be to prevent the need for constantly maintaining a
`.dockerignore`. Sure, add some obvious entries to `.dockerignore` but it's not
longer going to impact container size, just build times.

### #2 Build environment vs Runtime environment

If a service runs a compiled language like Java or Go, the build environment can
be separated from the runtime environment. Perhaps a Java service's
`Dockerfile.build` is `FROM openjdk-7-jdk` and its `Dockerfile` is `FROM
openjdk-7-jre` which is hopefully smaller.

For Go services we found this pattern particularly useful to ship ~4MB
(:exclamation:) images. Any language that can statically link its runtime can be
used to ship the smallest possible small Docker images by building in their
normal environment and then running the compiled binary inside a container
`FROM scratch`, or if the service needs SSL `FROM centurylink/ca-certs`.

For a working example see our [test project](https://github.com/adobe-platform/porter-test)
