Migration
=========

Read the [release notes](RELEASE_NOTES.md) for context on these changes.

v4 to v5
--------

No migration steps are needed unless you hit
[`pids_limit`](docs/detailed_design/config-reference.md#pids_limit)

v3 to v4
--------

### Egress rules

If you HAVE defined `security_group_egress` this is not a breaking change.

If you HAVE NOT defined `security_group_egress` the default list is now

```yaml
security_group_egress:

# allow all DNS
- cidr_ip: 0.0.0.0/0
  ip_protocol: udp
  from_port: 53
  to_port: 53

# allow all NTP
- cidr_ip: 0.0.0.0/0
  ip_protocol: udp
  from_port: 123
  to_port: 123

# allow all HTTP
- cidr_ip: 0.0.0.0/0
  ip_protocol: tcp
  from_port: 80
  to_port: 80

# allow all HTTPS
- cidr_ip: 0.0.0.0/0
  ip_protocol: tcp
  from_port: 443
  to_port: 443
```

See the [`security_group_egress`](docs/detailed_design/config-reference.md#security_group_egress)
config for more info.

Since egress rules are now added to every deployment the addition of
`ec2:AuthorizeSecurityGroupEgress` and `ec2:RevokeSecurityGroupEgress` to the
porter deployment policy are now required. They used to be optional.

### HAProxy request header captures

This is a breaking change for logging field extractions that expect to parse
`X-Request-Id` or `X-Forwarded-For` in haproxy logs.

This is a snippet of the old config

```
  #
  # This capture list is ordered. APPEND ONLY or risk messing up log parsing
  #

  # Log request ids
  capture request header X-Request-Id len 40
  capture response header X-Request-Id len 40

  # Log original request IP
  # Currently only IPv4 is seen from EC2 instances but 45 handles IPv6 with
  # possible IPv4 tunneling. This is one of those things that's good to plan for
  # http://docs.aws.amazon.com/ElasticLoadBalancing/latest/DeveloperGuide/x-forwarded-headers.html
  capture request header X-Forwarded-For len 45
```

To match the old behavior add the `haproxy` object to your `.porter/config`

```yaml
environments:
- name: dev
  haproxy:
    request_header_captures:
    - header: X-Request-Id
      length: 40
    - header: X-Forwarded-For
      length: 45

    response_header_captures:
    - header: X-Request-Id
      length: 40
```

**You must re-provision for these changes to take effect**

See the [HAProxy docs](https://cbonte.github.io/haproxy-dconv/1.5/configuration.html#8.8)
for exactly where these end up in your logs

v2 to v3
--------

### 1. Additional permissions

Add the following permissions to your `porter-deployment` role's policy document.

If you didn't use `porter bootstrap iam` to create the deployment policy it may
have a different name. You can look at your `.porter/config` for `role_arn` to
find the role that needs updated.

```
cloudformation:DescribeStackResources
elasticloadbalancing:DescribeTags
sqs:CreateQueue
sqs:DeleteQueue
sqs:GetQueueAttributes
sqs:GetQueueUrl
sqs:ReceiveMessage
```

### 2. (optional) Enable hot swap

#### 2.1 Upgrade porter and do a normal deployment

You must do 1 normal deployment so the stack has the resources porter needs
at build time to perform a hot swap.

#### 2.2 Change config to enable hot swap

```yaml
environments:
- name: stage
  hot_swap: true
```

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
