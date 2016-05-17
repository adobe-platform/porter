post-pack
=========

Use Cases
---------

- Clean up test or build artifacts
- `rm -fr .git` if it's not longer needed to reduce artifact copying

Lifecycle
---------

This hook runs only once during `porter build pack` after packaging succeeds.

Environment
-----------

[Standard](../deployment-hooks.md#standard-environment-variables)
and [Custom](../deployment-hooks.md#custom-environment-variables)
environment variables
