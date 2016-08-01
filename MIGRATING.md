Migration
=========

v1 to v2
--------

### Summary

For services already using `uid` **there is nothing to do**.

For services not using `uid` **your container may break** now that it's not running as root.

### Details

v1.0.5 introduced what we later learned was a breaking change in how porter runs
docker containers. It was then fixed in v1.0.6 and v2 was created from v1.0.5 to
signal the break.

The only _known_ breaking change between v1 and v2 is the addition of `-u` to
`docker run` which defaults to a non-root user. This is configurable with
[`uid`](docs/detailed_design/config-reference.md#uid).

Other possible breaking changes include upgrades to the default Amazon Linux AMI
and Docker versions which could affect users using a custom AMI.
