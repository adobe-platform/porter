See the [CHANGELOG](CHANGELOG.md) for a complete list of changes.

`porter` is [semantically versioned](http://semver.org/spec/v2.0.0.html)

v3.0
====

[How to migrate](MIGRATING.md#v2-to-v3)

Code can now be hot swapped on existing infrastructure. This will mitigate a
whole class of errors related to provisioning.

One of the primary problems was signaling the build system that a hot swap
completed. This signal is now sent via a SQS queue that's provisioned with every
stack.

[Read more about how hot swap works](docs/detailed_design/hotswap.md)

v2.4
====

Hooks are now run concurrently across regions.

Hook have an optional `run_condition` enabling `post-*` hooks to run even if the
underlying command fails. Think of it as a `try...catch...finally` for
programming deployments.

v2.3
====

[Host-level secrets](docs/detailed_design/config-reference.md#secrets_exec_name)
can piggyback on the same secrets payload as container secrets which is
especially helpful for ec2-bootstrap hooks to pass sensitive info such as API
keys

[User-defined hooks](docs/detailed_design/deployment-hooks.md#user-defined-hooks)

v2.2
====

[Docker registries](docs/detailed_design/service-payload.md) are supported

[ASG egress rules can be defined](docs/detailed_design/config-reference.md#security_group_egress)

This changes the porter deployment policy to include
`ec2:AuthorizeSecurityGroupEgress` and `ec2:RevokeSecurityGroupEgress`

v2.1
====

It's now possible to deploy

1. Worker containers in a inet topology (i.e. with an ELB)
1. A worker-only CloudFormation stack (i.e. no ELB)
1. More than container per EC2 host

The the [topology config](docs/detailed_design/config-reference.md#topology) for
notes and limitations

v2
==

[How to migrate](MIGRATING.md#v1-to-v2)

## v2 - Version upgrades

- Docker has been updated to 1.11.2
- Amazon Linux AMI has been updated to 2016.03

## v2 - UID

v1.0.5 introduced what we later learned was a breaking change in how porter runs
docker containers. It was then fixed in v1.0.6 and v2 was created from v1.0.5 to
signal the break.

## v2 - Hooks

### v2 - Hook Location

Hooks are going through another pass at simplification.

There used to be a convention that hooks lives under `.porter/hooks`. Now the
path is configurable with the same key that's used to configure plugins:
`dockerfile`.

This gets rid of the constraint that "convention" hooks (i.e. those placed in
`.porter/hooks/`) are run before "configured" hooks. Now all hooks are
configured and execution order is completely in your control.

### v2 - Hook concurrency

Hooks can now be executed concurrently

### v2 - Custom Hook Environment

Custom hook environment used to be provided to all hooks by prefixing
environment variables with `PORTER_`. This is deprecated (but still supported)
in favor of whitelisting environment variables per hook in a similar style that
[Docker Compose](https://docs.docker.com/compose/compose-file/#/environment)
uses.

See [the docs](docs/detailed_design/deployment-hooks.md#custom-environment-variables)
for more

## v2 - S3 keys

Templates are now uploaded to

```
porter-template/{service name}/{environment}/{short sha}/
```

Deployment payloads and encrypted secrets are now uploaded to

```
porter-deployment/{service name}/{environment}/{short sha}/
```

This is a breaking change for tooling relying on the old layout.

## v2 - CloudFormation template location

**WARNING**: if you previously put secrets into your CloudFormation template be
aware that they are now uploaded to S3 in the same bucket defined in the config.

This **is not** a concern for infrastructure porter creates because it limits
permissions of EC2 IAM roles so that the template isn't accessible by default.

This **is** a concern for everything and everyone else that can access the S3
bucket.

Specifically instead of using `TemplateBody` with the [CreateStack](http://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_CreateStack.html)
API porter now uploads the template to S3 and calls CreateStack with
`TemplateURL` instead.

This was done to overcome the 51,200 byte limit on `TemplateBody`.

v1
==

Initial release
