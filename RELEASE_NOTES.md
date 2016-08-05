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
