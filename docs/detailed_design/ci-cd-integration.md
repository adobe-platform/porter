CI/CD integration
=================

In porter an environment is an opaque string that names a complex object with
some region and environment-specific configuration.

It's up to CI software to work with a service's `.porter/config` to build the
correct environments.

There are three major things to understand to integrate porter.

1. Build phases
1. Artifacts
1. Roles

tl;dr
-----

From a EC2 box with the right IAM role permissions, and an executable porter
binary in the working directory

```bash
./porter build pack && \
./porter build provision -e fill_this_in && \
./porter build promote && \
./porter build prune
```

Build phases
------------

There are four major phases in a service's lifecycle

1. Pack
2. Provision
3. Promote
4. Prune

See the
[high level flow](https://www.lucidchart.com/documents/view/95a3fdca-ff76-40c5-98fd-6b3071ba86bc)
for some visuals.

### Pack

The pack phase packages up an application. It doesn't do much besides a `docker
build` of the configured containers. Its job is to create the service payload
that will be shipped to the configured `service_distribution` type. It has no
concept of an environment.

It's the job of a CI box to support submodules, download the version of porter
for a particular version of code, etc.

```bash

# support git submodules
git submodule update --init --recursive

# download the configured version of porter
/usr/bin/download_porter .porter/config

# package the service
porter build pack
```

The `download_porter` script should be installed on the machine or be put inline
in a job definition.

```bash
#!/bin/bash -e
#
# This script will be used by CI/CD servers to download the version of porter
# defined in config file .porter/config
#
VERSION=$(perl -wne 'print "v$1" if /porter_version: \"?v(([A-Z]|[0-9]|\.)+)/' $1)
echo "Downloading porter $VERSION"
curl -Lo porter -s https://github.com/adobe-platform/porter/releases/download/$VERSION/porter_linux386 && chmod +x !#:2
```

### Provision

Provision operates on a particular environment in the `.porter/config`, and
concurrently on each region in the environment.

At a high level provision

1. Takes the service payload produced by the pack phase and ships it to the
   configured `service_distribution` type.
1. Creates a CloudFormation template per region and calls [CreateStack](http://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_CreateStack.html)
1. Polls stack events waiting for `CREATE_COMPLETE` per region

```bash
# provision the service
porter build provision -e some_environment
```

### Promote

Promote operates on a particular environment in the `.porter/config` (the same
environment given to provision), and concurrently on each region in the
environment.

At a high level promote

1. [Registers instances](http://docs.aws.amazon.com/ElasticLoadBalancing/latest/APIReference/API_RegisterInstancesWithLoadBalancer.html) with the configured ELB
1. Waits for all instances to be `InService`
1. Deregisters instances not part of the provisioned stack
1. Tags the ELB with the CloudFormation stack id

The ELB is tagged for resiliency. Everytime any EC2 instance is initialized it
queries all ELBs for the environment-region that it could possibly be promoted
into. If its CloudFormation stack id matches the ELB tag value then it registers
it with the ELB so it can receive traffic.

Promotion is idempotent and can be used on previous builds. It doesn't take any
arguments because it relies on build artifacts in `.porter-tmp` that were
produced by `porter build provision`

```bash
porter build promote
```

### Prune

Prune operates on a particular environment in the `.porter/config` (the same
environment given to provision), and concurrently on each region in the
environment.

Its only job is to call DeleteStack on CloudFormation stacks with EC2 instances
not current registered to any static ELB. The number of stacks (eligible for
deletion) to keep is an optional parameter and defaults to 0.

```bash
porter build prune
```

Artifacts
---------

The Pack phase produces temporary files in `.porter-tmp/` that must be available
for subsequent phases to work.

Roles
-----

In a typical CI setup where the CI boxes are on EC2, there are always two IAM
roles needed for porter to operate: the invoke role and the assumed role.

### Invoke role

The invoke role is the role associated with the EC2 instance calling porter. The
simplest setup to describe is CI software installed on an EC2 host which has
access to EC2 metadata.

The invoke role needs the following policy doc:

```json
{
  "Statement": [
    {
      "Action": [
        "sts:AssumeRole"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

Porter always calls STS AssumeRole before calling AWS APIs.

This approach yields the most flexibility and enables

1. Users with long-term credentials to call porter
1. Federated users with temporary credentials to call porter
1. Porter to assume roles in other AWS accounts

### Assumed role

The assumed role is assumed by the invoke role (or user, federated user, etc.).

It has attached to it a policy permitting the role to operate on various AWS
resources in the account. Additionally it has a trust policy specifying the ARN
of the invoke role.

An example trust policy allowing the invoke role `build_box` and the user
`someone` in account `123456789012` to assume it:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::123456789012:user/someone",
          "arn:aws:iam::123456789012:role/build_box"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```
