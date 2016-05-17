pre-pack
========

Use Cases
---------

- Unit tests
- Linting
- Static analysis
- Config generation

Lifecycle
---------

This hook runs only once during `porter build pack`.

Environment
-----------

[Standard](../deployment-hooks.md#standard-environment-variables)
and [Custom](../deployment-hooks.md#custom-environment-variables)
environment variables

Examples
--------

```Dockerfile
FROM golang:1.6

# switch to the volume-mapped /repo_root
WORKDIR /repo_root

# run tests
CMD go test
```
