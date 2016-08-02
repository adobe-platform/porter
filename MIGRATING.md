Migration
=========

v1 to v2
--------

### Version upgrades

- Docker has been updated to 1.11.2
- Amazon Linux AMI has been updated to 2016.03

### UID

#### Summary

For services already using `uid` **there is nothing to do**.

For services not using `uid` **your container may break** now that it's not running as root.

#### Details

v1.0.5 introduced what we later learned was a breaking change in how porter runs
docker containers. It was then fixed in v1.0.6 and v2 was created from v1.0.5 to
signal the break.

### Hooks

#### Hook Location

Hooks are going through another pass at simplification.

There used to be a convention that hooks lives under `.porter/hooks`. Now the
path is configurable with the same key that's used to configure plugins:
`dockerfile`.

A v1 pre-pack hook would be placed at `.porter/hooks/pre-pack` and not need any
additional configuration for porter to find it.

A v2 pre-pack hook can be placed anywhere porter can resolve the path and you
specify the location in `.porter/config`. The only change needed to migrate to
v2 is to specify the path:

```yaml
hooks:
  pre_pack:
    - dockerfile: .porter/hooks/pre-pack
```

#### Custom Hook Environment

Custom hook environment used to be provided to all hooks by prefixing
environment variables with `PORTER_`. This is deprecated (but still supported)
in favor of whitelisting environment variables per hook in a similar style that
[Docker Compose](https://docs.docker.com/compose/compose-file/#/environment)
uses.

See [the docs](docs/detailed_design/deployment-hooks.md#custom-environment-variables)
for more
