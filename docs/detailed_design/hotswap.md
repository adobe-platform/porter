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
24 hours before re-provision must occur. We think this strikes a nice balance
between a fast feedback loop, a good security posture, and avoiding
infrastructure drift.

When not to use it
------------------

Often integration tests are run in the `pre_promote` hook as a quality gate. Since
promotion is skipped there's no chance to run tests against your running code
before it starts serving traffic.

We suggest a stage and production environment (in general but especially for hot
swap). Both can be configured to hot swap and in addition to a `pre_promote`
hook you would configure stage to run a `post_hotswap` hook that performs the
same testing as a gate to allow code through to production.

How it works
------------

During `porter build provision` on an environment with hot swap enabled the
following occurs:

**On the build box**

1. Check the creation time of the stack that was most recently promoted to the
   configured ELB and determine if it's within the 24 hour timeframe
1. If so perform the normal steps of uploading the service payload, building
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
1. `/usr/bin/porter_hotswap` runs the configured docker containers
1. `/usr/bin/porter_hotswap` reloads nginx config to send traffic to them
1. `/usr/bin/porter_hotswap` drains connections on the old containers
1. `/usr/bin/porter_hotswap` sends a success message to the same SQS queue that
    porter is currently receiving messages on
