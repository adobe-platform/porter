Config reference
================

For each field the following notation is used

- (==1?) means the field is OPTIONAL and ONLY ONE can exist
- (>=1?) means the field is OPTIONAL and MORE THAN ONE can exist
- (==1!) means the field is REQUIRED and ONLY ONE can exist
- (>=1!) means the field is REQUIRED and MORE THAN ONE can exist

`.porter/config`

- [service_name](#service_name) (==1!)
- [porter_version](#porter_version) (==1!)
- [environments](#environments) (>=1!)
  - [name](#environment-name) (>=1!)
  - [stack_definition_path](#stack_definition_path) (==1?)
  - [role_arn](#role_arn) (==1!)
  - [instance_count](#instance_count) (==1?)
  - [instance_type](#instance_type) (==1?)
  - [blackout_windows](#blackout_windows) (>=1?)
  - [regions](#regions) (>=1!)
    - [name](#region-name) (==1!)
    - [stack_definition_path](#stack_definition_path) (==1?)
    - [vpc_id](#vpc_id) (==1?)
    - [role_arn](#role_arn) (==1!)
    - [ssl_cert_arn](#ssl_cert_arn) (==1?)
    - [hosted_zone_name](#hosted_zone_name) (==1?)
    - [key_pair_name](#key_pair_name) (==1?)
    - [s3_bucket](#s3_bucket) (==1!)
    - [elb](#elb) (==1?)
    - [azs](#azs) (>=1!)
      - name
      - [subnet_id](#subnet_id) (==1?)
    - [containers](#containers) (>=1?)
      - name
      - [topology](#topology) (==1?)
      - [inet_port](#inet_port) (==1?)
      - [uid](#uid) (==1?)
      - [health_check](#health_check) (==1?)
      - [src_env_file](#src_env_file) (==1?)
      - [dst_env_file](#dst_env_file) (==1?)
- [hooks](#hooks) (==1?)
  - pre_pack (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - post_pack (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - pre_provision (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - post_provision (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - pre_promote (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - post_promote (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - pre_prune (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - post_prune (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)
  - ec2_bootstrap (==1?)
    - [repo](#repo) (==1!)
    - [ref](#ref) (==1!)
    - [dockerfile](#dockerfile) (==1?)

### service_name

service_name is used to form CloudFormation stack names as well as "folders" in
the bucket that a service is uploaded into.

Must match `/^[a-zA-Z][-a-zA-Z0-9]*$/`

### porter_version

porter_version is used by deployment servers to download the correct version of
porter before running porter build commands

Must match `/^v\d+\.\d+\.\d+$/`

### environments

environments is a complex object and namespace for configuration

### environment name

Must match `/^[0-9a-zA-Z]+$/`

### stack_definition_path

stack_definition_path is a relative path from the `.porter/config` to a
CloudFormation stack definition file.

The most specific definition is used meaning if it's defined on an environment
and an environment's region, the region value will be used.

### role_arn

role_arn is the IAM Role that porter will call AssumeRole on in order to perform
the AWS API actions needed to create, update, rollback, and delete
CloudFormation stacks and the resources they define.

It is marked optional but is required either on the environment or region. If
both are specified the region value will be used.

Example ARN

```
arn:aws:iam::123456789012:role/porter-deployment
```

### instance_count

instance_count is the desired number of instance per environment-region.

The default is 1.

### instance_type

instance_type the EC2 instance type to be used.

The default is m3.medium

Acceptable values are

```
c1.medium
c1.xlarge
c3.2xlarge
c3.4xlarge
c3.8xlarge
c3.large
c3.xlarge
cc2.8xlarge
hs1.8xlarge
i1.4xlarge
i2.2xlarge
i2.4xlarge
i2.8xlarge
i2.xlarge
m1.large
m1.medium
m1.small
m1.xlarge
m2.2xlarge
m2.4xlarge
m2.xlarge
m3.2xlarge
m3.large
m3.medium
m3.xlarge
```

### blackout_windows

blackout_window contains a start_time and end_time between which
'porter build ...' commands will exit with status 1 immediately.

The times are parsed by https://golang.org/pkg/time/#Parse with layout RFC3339
from https://golang.org/pkg/time/#pkg-constants

The blackout window is considered active if

1. The current system time is greater than start_time and less than end_time
1. time.Parse returns an error when parsing start_time or end_time
1. start_time > end_time

A sample blackout_window

```
blackout_windows:
- start_time: 2015-01-02T15:04:05Z07:00
  end_time: 2015-01-03T15:04:05Z07:00
```

### regions

region is a complex object defining region-specific things

A sample region

```
regions:
- name: us-west-2
```

### region name

Must be one of

```
ap-northeast-1
ap-northeast-2
ap-southeast-1
ap-southeast-2
eu-central-1
eu-west-1
sa-east-1
us-east-1
us-west-1
us-west-2
```

### ssl_cert_arn

ssl_cert_arn is ARN of a SSL cert. If defined a HTTPS listener is added to the
*provisioned* ELB (not the elb defined to promote instances into which must be
configured manually)

This is typically used with `hosted_zone_name` to create a developer stack that
works with SSL.

### hosted_zone_name

hosted_zone_name is DNS zone in Route53 that will be aliased with the
provisioned ELB's A record if provided

This is typically used with `ssl_cert_arn` to create a developer stack that
works with SSL.

### key_pair_name

key_pair_name is name of the SSH key pair that will be used to login to EC2
instances.

### s3_bucket

The bucket used by porter to upload builds into.

### vpc_id

The VPC id needed to create security groups

Must match `/^vpc-(\d|\w){8}$/`

### azs

Availability zones are heterogeneous and differ between AWS accounts so they
must be explicity defined

A sample availability zone:

```
regions:
- name: us-west-2
  azs:
  - {name: us-west-2a, subnet_id: subnet-abcd1234}
```

### subnet_id

If a VPC is defined then subnets must also be defined within each AZ for
Autoscaling groups and ELBs to work correctly.

Must match `/^subnet-(\d|\w){8}$/`

### elb

The name of an elb. This is found in the AWS console and can be created with
`porter bootstrap elb`.

The value is used during `porter build promote` to determine where instances
should be promoted into.

The value is also used during `porter build prune` to determine which
Cloudformation stacks are eligible for deletion.

### containers

container is a container definition and complex object.

If undefined a single default container definition is provided:

```
containers:
- name: primary
  topology: inet
  health_check:
    method: GET
    path: /health
```

### topology

topology describes the basic topology of the service and allow porter to do
certain validation around the CloudFormation template to ensure things like
a load balancer are defined.

The only topology currently supported is `inet`

Future work will support `worker` and `cron`

### inet_port

This specifies which EXPOSEd docker porter is to receive internet traffic.

Services only need to define this if the Dockerfile EXPOSEs more than one port
otherwise the single EXPOSEd port is used.

If a service EXPOSEs more than one port this field is required.

This enables services to open up ports for things like profiling tools.

### uid

This specifies the uid the container is run with (i.e. `docker run -u`).

The default if left unset is to use root.

It's strongly recommended to use the provisioned user uid of `601`.

### health_check

Health check is a complex object defining a container's HTTP-based health check

The method and path are needed if a health check object is provided.

The default health check for every container is

```
health_check:
  method: GET
  path: /health
```

### src_env_file

See the docs on [container config](container-config.md) for more info on this
field

### dst_env_file

See the docs on [container config](container-config.md) for more info on this
field

### hooks

Read more about [deployment hooks](#deployment-hooks.md)

### repo

The repo to pull from. Porter simply does a git clone on whatever value is given
meaning URLs and relative paths to a repo are supported.

### ref

The branch, sha, or tag to use

### dockerfile

The relative path from the repo root to a dockerfile
