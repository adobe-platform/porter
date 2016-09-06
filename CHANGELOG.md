**v2.1.1**

- fix ec2-bootstrap hook clone for multi-region deployment
- configurable `-x` in `/var/log/cloud-init-output.log`
- service payload path is relative to support non-root volume

**v2.1.0**

- `topology: worker` now supported
- configurable `read_only: false` to disable `docker run --read-only`

**v2.0.0**

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

**v1.0.5**

- run the container as a non-root user by default (configurable with uid)

**v1.0.4**

- add retries to one more DescribeStackResource
- add an adjustable stack status polling frequency, see `porter debug help`

**v1.0.3**

- add retries to DescribeStackResource

**v1.0.2**

- update aws sdk to v1.1.36

**v1.0.1**

- Fixed security group on ELB for SSL in VPC

**v1 - Initial release**
