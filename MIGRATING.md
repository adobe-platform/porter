Migration
=========

Read the [release notes](RELEASE_NOTES.md) for context on these changes.

v1 to v2
--------

### UID

For services already using `uid` **there is nothing to do**.

For services not using `uid` **your container may break** now that it's not
running as root.

### Hook Location

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
