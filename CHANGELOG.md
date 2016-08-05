**v2.0**

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
