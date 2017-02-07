# Troubleshooting

The most common failure seen in porter deployments is a mis-configured container
that either (a) fails to start or (b) fails to pass a health check (which must
return 200)

## General advice

- Iterate using `porter create-stack`, not from a build box
- Enable debug options (`porter help debug`) like increasing the stack timeout
- **Login to the box** - otherwise you're flying blind

## My stack rolled back BEFORE the WaitCondition handle failed

Look at the CloudFormation console for what failed. There's usually an obvious
reason, often related to configuration or permissions.

## My stack rolled back AFTER the WaitCondition handle failed

That means EC2 initialization failed and porter never declared your container
healthy. This is a good thing. The alternative is porter allowing you to deploy
broken software.

### Was cloud-init able to install packages?

If you see log messages indicating a network timeout then your instance can't
connect to the internet. If your instance is in a public subnet make sure the
default route is to an internet gateway. If your instance is in a private subnet
make sure the default route is to a nat instance or gateway.

### Are you running ec2-bootstrap hooks?

These are often a source of failed ec2 initialization. Vanilla porter
deployments (i.e., those without ec2-bootstrap customizations) are rock-solid
and _fast_. You'll usually see errors in the cloud-init and cfn-init logs.

### Does everything look ok in the cloud-init log?

Then you're on to failed container startup or a failed container health check.
Look at porter's log to see how far you got into porter's initialization on the
host.

Start with `grep cmd=docker /var/log/porter.log` to look for docker errors. If
you don't see any logs with `lvl=eror` then repeat but look for `cmd=haproxy`
instead. Each warning or error contains a full stack trace so you can see
exactly where the log message was generated.

If all that looks good then all that's left is a failed health check. Confirm by
looking at porterd's logs `grep service=porterd /var/log/porter.log`
