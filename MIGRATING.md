Migration
=========

Read the [release notes](RELEASE_NOTES.md) for context on these changes.

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
