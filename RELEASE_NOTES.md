See the [CHANGELOG](CHANGELOG.md) for a complete list of changes.

`porter` is [semantically versioned](http://semver.org/spec/v2.0.0.html)

v4.9
====

HAProxy's [`maxconn`](docs/detailed_design/config-reference.md#maxconn)
is now configurable.

v4.8
====

Some [HAProxy timeouts](docs/detailed_design/config-reference.md#timeout)
are configurable.

v4.7
====

ELB is now [optional](docs/detailed_design/config-reference.md#no-elb) for
`inet` containers.

The use of `elb: none` requires the additional permission
`autoscaling:DescribeTags`

v4.6
====

**Opting into host-level SSL support is a breaking change**

Host-level SSL support has been added using existing secrets management
mechanism for placing the cert on the host.

In order to have SSL all the way through it must be enabled on the host and on
the [ELB](http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/elb-create-https-ssl-load-balancer.html)

### Enable HTTPS on the provisioned ELB

```yaml
environments:
- name: dev

  regions:
  - name: us-west-2
    ssl_cert_arn: arn:aws:iam::123456789012:server-certificate/yourdomain.com
```

### Enable HTTPS on the host

```yaml
environments:
- name: dev

  regions:
  - name: us-west-2
    ssl_cert_arn: arn:aws:iam::123456789012:server-certificate/yourdomain.com

  haproxy:
    ssl:
      pem:
        secrets_exec_name: /bin/cat
        secrets_exec_args:
        - path/to/yourdomain_com.pem
```

For more on these fields see the [docs](docs/detailed_design/config-reference.md#ssl)

### v4.6 breaking changes

**These are only breaking changes if you opt into host-level SSL**

#### v4.6 ELB listeners

porter stack used to listen on ports 80 and 8080 but, when opted in, listens on
80 and 443. 8080 represented the SSL termination port. When opted into host-
level SSL the ELB into which porter promotes instances must be changed to send
HTTPS traffic to instance port 443.

For example if we have config like this

```yaml
environments:
- name: dev
  regions:
  - name: us-west-2
    elb: some-elb
```

If `some-elb` was previously setup to do SSL termination it would likely have
these listeners

| Load balancer protocol | Load Balancer Port | Instance protocol | Instance port | Cipher | SSL Certificate |
|------------------------|--------------------|-------------------|---------------|--------|-----------------|
| HTTP                   | 80                 | HTTP              | 80            |        |                 |
| HTTPS                  | 443                | HTTP              | 8080          |        | yourdomain.com  |

It would need to be changed to the following (changes in *italics*)

| Load balancer protocol | Load Balancer Port | Instance protocol | Instance port | Cipher | SSL Certificate |
|------------------------|--------------------|-------------------|---------------|--------|-----------------|
| HTTP                   | 80                 | HTTP              | 80            |        |                 |
| HTTPS                  | 443                | *HTTPS*           | *443*         |        | yourdomain.com  |

#### v4.6 ELB health check

If using `https_only` or `https_redirect` the health check on `some-elb` must be
changed as well. A health check of `HTTP:80/health` would need to be changed to
`HTTPS:443/health`

When migrating to host-level SSL it's recommended to complete the transition to
SSL before enabling either `https_only` or `https_redirect`

v4.5
====

Exposed HAProxy configuration to adjust logging and compression.

v4.4
====

Tuned network and CPU performance

v4.3
====

Revised instance type list

v4.2
====

All uploads now use S3's Standard - Infrequent Access to reduce S3 costs.

Keep-alive from HAProxy to the containers used to be disabled because of how hot
swap was implemented. Another round of load testing was done to ensure hot swap
continued to work as expected now that keep-alive is enabled.

v4.1
====

There is now a configurable instance count per region.

Extended infrastructure TTL (the time allowed before re-provisioning must occur)
from 24 hours to a week.


v4.0
====

[How to migrate](MIGRATING.md#v3-to-v4)

v4 - Egress rules
-----------------

For a number of security reasons porter now locks down ASG egress traffic to
allow by default NTP (udp 123), DNS (udp 53), HTTP (tcp 80), and
HTTPS (tcp 443). You can still define your own egress rules using [`security_group_egress`](docs/detailed_design/config-reference.md#security_group_egress)
or turn off porter's security group management entirely with
[`autowire_security_groups: false`](docs/detailed_design/config-reference.md#autowire_security_groups).

v4 - HAProxy header capture
---------------------------

Enabled configurable HAProxy header captures which make HAProxy logs more useful
to those that weren't using UUIDv4 for `X-Request-Id`

v4 - Failed stacks delete
-------------------------

The new default to the `OnFailure` parameter of `cloudformation:CreateStack` is
`DELETE`.

If the previous behavior is desired it can be set using
`STACK_CREATION_ON_FAILURE`. Run `porter help debug` for more.

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

v2 - Version upgrades
---------------------

- Docker has been updated to 1.11.2
- Amazon Linux AMI has been updated to 2016.03

v2 - UID
--------

v1.0.5 introduced what we later learned was a breaking change in how porter runs
docker containers. It was then fixed in v1.0.6 and v2 was created from v1.0.5 to
signal the break.

v2 - Hooks
----------

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

v2 - S3 keys
------------

Templates are now uploaded to

```
porter-template/{service name}/{environment}/{short sha}/
```

Deployment payloads and encrypted secrets are now uploaded to

```
porter-deployment/{service name}/{environment}/{short sha}/
```

This is a breaking change for tooling relying on the old layout.

v2 - CloudFormation template location
-------------------------------------

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
