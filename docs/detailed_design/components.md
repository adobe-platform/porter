porter components
=================

`porter` consists of services, and components that run in various contexts.

## Contexts

### Local developer box

`porter` was designed to be developer-friendly and help is just a `porter help`
command away.

Commonly used commands are the fewest keystrokes, with commands used in
host-level configuration and build boxes nested under other commands.

### CI/CD box

The `porter build ...` commands are designed to be run in an automated
environment with minimal dependencies.

We like GoCD because
[it models parallel and serial steps explicitly](http://www.go.cd/documentation/user/current/introduction/concepts_in_go.html)
and allows pipelines to be chained together. GoCD also gives us scheduling, job
templates, an API for regulating deployment windows, and a UI for navigating it
all.

`porter` works with any CI/CD software.
[Learn how to use your own](detailed_design/ci-cd-integration.md)

### EC2

There is a stateless and stateful aspect to `porter` on an EC2 host.

#### Stateless

Both of the first two contexts will lead here, to an EC2 host. Each of the
previous contexts built a `cloud-init` script as part of an
AWS::AutoScaling::LaunchConfiguration's UserData property to be run when an EC2
instance is created. This script runs `porter host ...` commands to do the
configuration of Amazon Linux AMI to install things like HAProxy and Splunk.

Since `porter` built the scripts that run on a host this is essentially `porter`
talking to itself.

More on `cloud-init`
- [User Data and cloud-init Directives](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-cloud-init)
- [Amazon Linux AMI Basics - cloud-init](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AmazonLinuxAMIBasics.html#CloudInit)

#### Stateful

The `cloud-init` script also installs `porterd` as a host-level daemon.
`porterd` has two primary functions.

1. Provides useful features to services over HTTP
2. Plays a role in coordinating service lifecycle and monitoring

One useful bit of coordination `porterd` does is to monitor a service's health
and call the AWS::CloudFormation::WaitConditionHandle on behalf of a service,
which allows the CloudFormation stack to complete.

This means a "Hello world HTTP" service doesn't need an AWS SDK to get off the
ground.
