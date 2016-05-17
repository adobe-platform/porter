post-promote
============

Use cases
---------

- Build notifications
- Running integration tests against the DNS of the ELB that instances were
  promoted into. This is a best practice to catch errors related to security
  groups and other possible mismatches between the provisioned ELB and the ELB
  into which instances were promoted.

Lifecycle
---------

This hook is called from `porter build promote` **for each** region
**after all** regions are promoted
(as opposed to after each region is promoted)

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
