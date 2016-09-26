Service payload
===============

The service payload by default is a compressed tarball containing a modified
`.porter/config` and the output of `docker save -o` for each container defined.

This is multipart uploaded to S3 with the following key

```
porter-deployment/{service name}/{environment}/{git rev-parse --short HEAD}/{md5 of tarball}.tar
```

Docker registry
---------------

Alternatively a docker registry can be used for docker images. With this
configuration the S3 payload only contains the modified `.porter/config`.

Registry-based deployments are configured outside of `.porter/config` with
environment variables.

`DOCKER_REGISTRY` is the hostname of the registry excluding the `http://` or
`https://` prefixes. When using public Docker Hub set this to `index.docker.io`.

`DOCKER_REPOSITORY` is the repository name.
For public Docker Hub use `<username>/<repository>`

`DOCKER_PUSH_USERNAME` and `DOCKER_PUSH_PASSWORD`, if defined, will be used to
perform `docker login` before doing a `docker push`. Otherwise `docker login`
will be skipped.

`DOCKER_PULL_USERNAME` and `DOCKER_PULL_PASSWORD`, if defined, will be used to
perform `docker login` before running containers on a EC2 host. Otherwise
`docker login` will be skipped. These credentials are placed in the same secrets
payload as [container config](container-config.md).

Set `DOCKER_INSECURE_REGISTRY=1` if you're using a registry with self-signed
certs and porter will add `--insecure-registry=$DOCKER_REGISTRY` to the EC2
host's docker daemon config.
