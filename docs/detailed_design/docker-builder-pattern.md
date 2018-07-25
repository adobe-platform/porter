Docker Builder Pattern
======================

The Docker context
------------------

The `docker build ...` command's context is, by default, the filesystem and
experienced Docker users quickly find out they need to add to a `.dockerignore`
file to prevent the context from bloating and slowing down build times.

`.dockerignore` is a blacklist.

`docker build ...` alternatively accepts as its context a tar stream on STDIN.
If a [dockerfile_build](config-reference.md#container-dockerfile-build) is found
porter will build this image and run it, piping the STDOUT of it into the STDIN
of `docker build -f Dockerfile`

For an example `service_name: foo`, container `dockerfile: Dockerfile`, and
container `dockerfile_build: Dockerfile.build`, porter would run these commands
during the pack phase:

```
docker build -t foo-builder -f Dockerfile.build .
docker run --rm foo-builder | docker build -t foo -f Dockerfile -
```

This enables separating a build environment from a runtime environment, as well
as simply turning the `.dockerignore` blacklist into a `tar` stream whitelist.

Build args
----------

Often a build environment needs credentials to call into supporting services
like privately hosted registries or repositories. These credentials can be passed
to the build container as [build args](https://docs.docker.com/engine/reference/commandline/build/#set-build-time-variables---build-arg).

From the [Dockerfile reference](https://docs.docker.com/engine/reference/builder/#arg)

> Warning: It is not recommended to use build-time variables for passing secrets
> like github keys, user credentials etc. Build-time variable values are visible
> to any user of the image with the docker history command.

While build args are not supposed to be used for secrets the build container
isn't the final container. Its contents are available only in `docker history`
of the machine that runs `porter build pack`, NOT in the final container (unless
the user passes them on, of course). A `docker rmi -f` is run after the build
image is built and run, even if intermediate steps fail.

> Also, these values don’t persist in the intermediate or final images like ENV values do.

If all of the above is acceptable then read on.

Porter will take the intersection of (1) build args declared in the
`Dockerfile.build` (or `dockerfile_build`) and (2) its own environment.

**Example `Dockerfile.build`**

```Dockerfile
FROM ubuntu:18.04

ARG username
ARG password

RUN download_dependencies -u $username -p $password

# ...
```

Porter parses each `ARG` value and performs a [`os.LookupEnv`](https://golang.org/pkg/os/#LookupEnv).
Environment variables may be empty but they must exist.

If the environment variable `username` is `Bob` and `password` is `easytocrack`
then the resulting commands would look like this:

```
docker build -t foo-builder -f Dockerfile.build --build-arg username=Bob --build-arg password=easytocrack .
docker run --rm foo-builder | docker build -t foo -f Dockerfile -
```

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
openjdk-7-jre` which is smaller.

For Go services we found this pattern particularly useful to ship ~4MB
(:exclamation:) images. Any language that can statically link its runtime can be
used to ship the smallest possible small Docker images by building in their
normal environment and then running the compiled binary inside a container
`FROM scratch`, or if the service needs TLS `FROM centurylink/ca-certs`.

For a full working example see our [sample project](../../sample_project)
