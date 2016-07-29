Migration
=========

v1 to v2
--------

v1.0.5 introduced what we later learned was a breaking change in how porter runs
docker containers. It was then fixed in v1.0.6 and v2 was created from v1.0.5 to
signal the break.

The only _known_ breaking change between v1 and v2 is the addition of `-u` to
`docker run` which defaults to a non-root user. This is configurable with
[`uid`](docs/detailed_design/config-reference.md#uid).

Other possible breaking changes include upgrades to the default Amazon Linux AMI
and Docker versions which could affect users using a custom AMI.
