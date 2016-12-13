Hot swap
========

Hot swap is a configurable "fork" in porter's normal deployment flow. It allows
new code to be deployed on existing infrastructure.

Infrastructure provisioning is a common point of failure and has a slow feedback
loop. That downside has a big upside in that [security patches](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AmazonLinuxAMIBasics.html#security-updates)
are applied to EC2 hosts on their first launch. There were 169 CVEs in 2015
(that's 1 every ~2 days) that [AWS tracked and patched](https://alas.aws.amazon.com)

Another reason to use short-lived infrastructure is to prevent "drift". Examples
of drift include any host-level state like logs and package lists. Long-lived
infrastructure exposes a whole class of errors that are easily avoidable.

For these reasons porter enforces that hot swap can only occur for a maximum of
one week hour before re-provision must occur. We think this strikes a nice
balance between a fast feedback loop, a good default security posture, and
avoiding infrastructure drift.

Assumptions
-----------

The assumptions exist in normal provisioning as well but they are slightly
different in the hot swap flow and worth calling out.

### Short-lived connections

ELB keep-alive to HAProxy is 60 seconds. Per-host, porter waits an additional
120 seconds before declaring hot swap a failure. The primary assumption of hot
swap is short-lived connections. If you have long-lived connections you'll want
to avoid hot swap. That said you can't have longer-lived connections than what
you set your ELB connection draining to anyway (max is 3600 seconds).

### Version compatibility

Partial failures can occur and porter doesn't try to repair them. In the worst
case a multi-region hot swap could see a whole region fail to hot swap and other
regions partially fail to hot swap.

If version A of your software is deployed and can not coexist with version B
then you should not use hot swap.

When not to use it
------------------

An extension of assumptions relates to best practices around delivering SaaS -
namely testing.

Often integration tests are run in the `pre_promote` hook as a quality gate.
Since promotion is skipped there's no chance to run tests against your running
code before it starts serving traffic.

We suggest - in general, but especially for hot swap - a stage and production
environment. Both can be configured to hot swap and in addition to a
`pre_promote` hook you would configure stage to run a `post_hotswap` hook that
performs the same testing as a gate to allow code through to production.

Interaction with AutoScaling
----------------------------

It's important that AutoScaling events are preserved. Porter inspects the
currently promoted stack's AutoScalingGroup's `MinSize`, `MaxSize`, and
`DesiredCapacity`. Porter ensures that the following is true before performing a
hot swap:

```
MaxSizeᵀ = your CloudFormation template's AutoScalingGroup's MaxSize
MinSizeᵀ = your CloudFormation template's AutoScalingGroup's MinSize
DesiredCapacityᵀ = your CloudFormation template's AutoScalingGroup's DesiredCapacity
DesiredCapacityᴾ = the currently promoted stacks's AutoScalingGroup's DesiredCapacity

# must be true for hot swap to occur
MinSizeᵀ <= DesiredCapacityᴾ <= MaxSizeᵀ
```

During hot swap, when porter assembles your CloudFormation template it will
overwrite your template-defined `DesiredCapacityᵀ` with `DesiredCapacityᴾ` to
match the stack being updated so that any scaling that was done is preserved.
The same is done during normal provisioning.

This means if you opt into hot swap that `DesiredCapacityᵀ` is only used once
and from then on porter uses `DesiredCapacityᴾ` as long as hot swap is enabled.

To have `DesiredCapacityᵀ` be used again you must turn hot swap off.

How it works
------------

During `porter build provision` on an environment with hot swap enabled the
following occurs:

**On the build box**

1. Check the creation time of the stack that was most recently promoted to the
   configured ELB and determine if it's within the one week timeframe
1. If so, perform the normal steps of uploading the service payload, building
   out a CloudFormation template, and uploading it to S3
1. Call `pre_hotswap` hook
1. Instead of `cloudformation:CreateStack`, call `cloudformation:UpdateStack`
   with the uploaded CloudFormation template
1. Determine the number of instances in the ASG (`instanceCount`)
1. Retrieve the stack's SQS queue dedicated to hot swap and poll for
   `instanceCount` success messages
1. Call `post_hotswap` hook

**On each EC2 host**

1. [cfn-hup](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-hup.html)
   sits in a polling loop and triggers `/usr/bin/porter_hotswap` once a stack
   update is detected
1. Run the configured docker containers
1. Health check the containers on their configured `inet_port` and `health_check`
1. Once healthy reload haproxy config to send traffic to them
1. Drain connections on the old containers
1. `docker stop` on the old container ids
1. `docker rm` on the old container ids
1. `docker rmi` on the old image
1. Send a success message to the same SQS queue that porter is currently
   receiving messages on
