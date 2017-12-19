`porter` is [semantically versioned](http://semver.org/spec/v2.0.0.html)

### v5.0.0

- upgrade to Go 1.9.2

### v4.9.0

- HAProxy `maxconn` is configurable
- fixed issue where `maxconn` wasn't set on the frontend

### v4.8.3

- enabling fix for volume mounts on SELinux by setting the environment variable `VOLUME_FLAG`

### v4.8.2

- allow selinux hosts to share the mounted volumes with the containers.
- allow ap-south-1 region

### v4.8.1

- replace deprecated sysctl setting

### v4.8.0

- HAProxy `timeout client` is configurable
- HAProxy `timeout server` is configurable
- HAProxy `timeout tunnel` is configurable
- HAProxy `timeout http-request` is configurable
- HAProxy `timeout http-keep-alive` is configurable

### v4.7.0

- build porter with Go 1.8.1
- optional ELB

### v4.6.0

- host-level SSL support

### v4.5.0

- added opt-in HAProxy compression
- added configurable list of MIME types to compress
- HAProxy logs can be turned off

### v4.4.0

- disabled userland proxy
- tuned network buffers

### v4.3.0

- added `c4.*`, `r4.*`, and `x1.*` instance types
- updated `m4.*` and `t2.*` instance types
- removed `g2.*`, `i2.*`, and `d2.*` instance types

### v4.2.0

- HAProxy stats endpoint auth is now randomized
- re-enabled keep-alive between HAProxy and containers
- building on go 1.8
- added STANDARD_IA to secrets and CFN template uploads

### v4.1.1

- ASG size matching only occurs when `hot_swap: true`

### v4.1.0

- configurable instance count per region
- extended infrastructure ttl to a week

### v4.0.1

- fix type assertion for sg-ids that are statically defined

### v4.0.0

- failed stacks now delete instead of rollback
- lock down ASG egress traffic to allow by default NTP, DNS, HTTP, and HTTPS
- configurable haproxy header capture for logging

### v3.1.1

- service payloads were not encrypted as the docs said they were

### v3.1.0

- increased devicemapper base size to 50GB

### v3.0.5

- add `autowire_security_groups` so security group management can be turned off

### v3.0.4

- fixed issue with region-concurrent cleanup of service payload
- fixed possible issues with false-positive command success

### v3.0.3

- add `CREATE_IN_PROGRESS` to list of statuses that ignore ASG size
- add `DELETE_IN_PROGRESS` to list of statuses that ignore ASG size
- add `ROLLBACK_IN_PROGRESS` to list of statuses that ignore ASG size

### v3.0.2

- additional UPDATE steady states allow ASG matching
- any UPDATE in progress state causes hot swap to fail
- mac binaries are now built with Go 1.7.3
- match currently promoted stack's ASG size for provisioning and hot swap

### v3.0.1

- allow 10 mins for service payload download+install during hot swap
- check for egress rules before writing `SecurityGroupEgress`

### v3.0.0

- updated to Amazon Linux 2016.09
- use Standard - Infrequent Access for service payload
- hot swap code on existing infrastructure
- kernel tuning allowing more concurrent connections
- added pre and post hotswap hooks
- fixed v2.4.3 issue that could create false-positives in `porter build` steps
- `net.ipv4.netfilter.ip_conntrack_tcp_timeout_time_wait = 1`
- added `cloudformation:DescribeStackResources` to deployment policy
- added `elasticloadbalancing:DescribeTags` to deployment policy
- added `sqs:CreateQueue` to deployment policy
- added `sqs:DeleteQueue` to deployment policy
- added `sqs:GetQueueAttributes` to deployment policy
- added `sqs:GetQueueUrl` to deployment policy
- added `sqs:ReceiveMessage` to deployment policy
- added `sqs:SendMessage` to ASG inline policy

### v2.4.3

- reject config files with `run_condition` set in a pre hook
- run post hooks with `run_condition` set to `fail` when a pre hook fails

### v2.4.2

- fix missing or incomplete hook logs

### v2.4.1

- gather hook log output by hook since they run concurrently

### v2.4.0

- added retries to instance autoregistration
- gather hook log output by region since they run concurrently
- log colorization is off by default
- run hooks concurrently across regions
- hook `run_condition`

### v2.3.0

- more resiliency for service payload downloads
- switch to sha-256 and validate service payload integrity
- extend container secret management to the host with `porter_get_secrets`
- fix support for running arbitrary user defined hooks

### v2.2.0

- run docker with `--security-opt=no-new-privileges`
- support docker registries as an alternative to S3
- support auto scaling group egress whitelist
- deprecated `dst_env_file`
- added `sse_kms_key_id` for optional SSE-KMS on all porter uploads

### v2.1.2

- increase logrotate size from 10M to 100M

### v2.1.1

- fix ec2-bootstrap hook clone for multi-region deployment
- configurable `-x` in `/var/log/cloud-init-output.log`
- service payload path is relative to support non-root volume

### v2.1.0

- `topology: worker` now supported
- configurable `read_only: false` to disable `docker run --read-only`

### v2.0.0

- improved secrets handling in transit
- enabled pluggable secrets provider
- locked down CloudFormation and S3 API call scopes to the resources needed
- service payload for S3 is now `{service name}/{environment}/{short sha}`
- add LOG_DEBUG environment variable for debug logging
- updated Amazon Linux AMI to 2016.03
- updated Docker to 1.11.2
- fixed config validation failure producing a false positive of success
- improved hook environment variable injection to match Docker Compose
- got rid of hardcoded `.porter/hooks/` and made path to hooks configurable
- tweaked config validation so config can be created dynamically in pre_pack
- enabled deployment hooks to run concurrently
- CIS Docker benchmark 1.11.0 remediations (2.13, 5.12, 5.14)
- CIS Linux 2014.09 benchmark remediation 9.2.13
- CloudFormation templates are now uploaded to S3 to avoid the 51,200 byte limit
- S3 keys are scoped under `porter-deployment` and `porter-template`

### v1.0.6

- run the container as root (configurable with uid) to fix breaking change

### v1.0.5

- run the container as a non-root user by default (configurable with uid)

### v1.0.4

- add retries to one more DescribeStackResource
- add an adjustable stack status polling frequency, see `porter debug help`

### v1.0.3

- add retries to DescribeStackResource

### v1.0.2

- update aws sdk to v1.1.36

### v1.0.1

- Fixed security group on ELB for SSL in VPC

### v1 - Initial release
