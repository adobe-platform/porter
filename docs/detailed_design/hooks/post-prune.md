post-prune
==========

Use cases
---------

- Additional cleanup that may or may not be associated with the pruned stack

Lifecycle
---------

This hook is called from `porter build prune` **for each** region
**after all** regions are pruned
(as opposed to after each region is pruned)

Environment
-----------

[Standard](../deployment-hooks.md#standard-environment-variables)
and [Custom](../deployment-hooks.md#custom-environment-variables)
environment variables

```
PORTER_ENVIRONMENT
AWS_REGION
```

`AWS_DEFAULT_REGION` `AWS_ACCESS_KEY_ID` `AWS_SECRET_ACCESS_KEY`
`AWS_SESSION_TOKEN` `AWS_SECURITY_TOKEN` are available to enable AWS SDKs to
make API calls and are the credentials of the assumed role.

`AWS_CLOUDFORMATION_STACKID` is known after provisioning and is set by porter.

`AWS_ELASTICLOADBALANCING_LOADBALANCER_DNS` is the DNS of the provisioned ELB
(which may be an empty string if the ELB is internal). This is not the DNS of
the ELB that instances were promoted into.

Additional environment variables can be injected by prefixing them with
`PORTER_` as noted in [deployment hooks](../deployment-hooks.md#hook-environment)
