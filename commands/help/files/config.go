/*
 *  Copyright 2016 Adobe Systems Incorporated. All rights reserved.
 *  This file is licensed to you under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License. You may obtain a copy
 *  of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 *  OF ANY KIND, either express or implied. See the License for the specific language
 *  governing permissions and limitations under the License.
 */
package files

import (
	"github.com/phylake/go-cli"
	"github.com/phylake/go-cli/cmd"
)

const (
	configFileLongHelp = `.porter/config is the first file loaded by most porter commands.

For each field the following notation is used
 - (==1?) means the field is OPTIONAL and ONLY ONE can exist
 - (>=1?) means the field is OPTIONAL and MORE THAN ONE can exist
 - (==1!) means the field is REQUIRED and ONLY ONE can exist
 - (>=1!) means the field is REQUIRED and MORE THAN ONE can exist`

	service_name = `service_name (==1!)

service_name is used to form CloudFormation stack names as well as "folders" in
the bucket that a service is uploaded into.`

	topology = `topology (==1!)

topology describes the basic topology of the service and allow porter to do
certain validation around the CloudFormation template to ensure things like
a load balancer are defined.`

	porter_version = `porter_version (==1!)

porter_version is used by deployment servers to download the correct version of
porter before running porter build commands`

	containers = `containers (>=1?)

container is a container definition and complex object.

If undefined a default container will be provided and be named "primary"`

	inet_port = `inet_port (==1?)

This specifies which EXPOSEd docker porter is to receive internet traffic.

Services only need to define this if the Dockerfile EXPOSEs more than one port
otherwise the single EXPOSEd port is used.

If a service EXPOSEs more than one port this field is required.

This enables services to open up ports for things like profiling tools.`

	health_check = `health_check (==1?)

Health check is a complex object defining a container's HTTP-based health check

The method and path are needed if a health check object is provided.

The default health check for every container is

    health_check:
      method: GET
      path: /health`

	environments = `environments (>=1!)

environments is a complex object and namespace for configuration`

	stack_definition_path = `stack_definition_path (==1?)

stack_definition_path is a relative path to a CloudFormation stack definition
file`

	role_arn = `role_arn (==1?)

role_arn is the IAM Role that porter will call AssumeRole on in order to perform
the AWS API actions needed to create, update, rollback, and delete
CloudFormation stacks and the resources they define.

It is marked optional but is required either on the environment or region. If
both are specified the region value will be used.

If the Stormcloud team is deploying your service simply use the role arn

    arn:aws:iam::523591739732:role/porter-deployment`

	instance_count = `instance_count (==1?)

instance_count is the desired number of instance per environment-region.

The default is 1.`

	instance_type = `instance_type (==1?)

instance_type the EC2 instance type to be used.

The default is m3.medium

Acceptable values are

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
    m3.xlarge`

	blackout_windows = `blackout_windows (>=1?)

blackout_window contains a start_time and end_time between which
'porter build ...' commands **that take an environment parameter as input** will
exit with status 1 immediately.

The times are parsed by https://golang.org/pkg/time/#Parse with layout RFC3339
from https://golang.org/pkg/time/#pkg-constants

The blackout window is considered active if

    1. The current system time is greater than start_time and less than end_time
    2. time.Parse returns an error when parsing start_time or end_time
    3. start_time > end_time

A sample blackout_window

    blackout_windows:
    - start_time: 2015-01-02T15:04:05Z07:00
      end_time: 2015-01-03T15:04:05Z07:00`

	regions = `regions (>=1!)

region is a complex object defining region-specific things

A sample region

    regions:
    - name: us-west-2`

	ssl_cert_arn = `ssl_cert_arn (==1?)

ssl_cert_arn is ARN of a SSL cert. If defined a HTTPS listener is added to the
*provisioned* ELB (not the elb defined to promote instances into)`

	hosted_zone_name = `hosted_zone_name (==1?)

hosted_zone_name is DNS zone in Route53 that will be aliased with the
provisioned ELB's A record if provided.`

	key_pair_name = `key_pair_name (==1?)

key_pair_name is name of the SSH key pair that will be used to login to EC2
instances. It is required if stack_definition_path is empty.`

	s3_bucket = `s3_bucket (==1!)

The bucket used by porter to upload builds into.`

	vpc_id = `vpc_id (==1?)

The VPC id needed to create security groups`

	azs = `azs (>=1!)

Availability zones are heterogeneous and differ between AWS accounts so they
must be explicity defined

A sample availability zone

    regions:
    - name: us-west-2
      azs:
      - {name: us-west-2a, subnet_id: subnet-abcd1234}`

	subnet_id = `subnet_id (>=1!)

If a VPC is defined then subnets must also be defined within each AZ for
Autoscaling groups and ELBs to work correctly.`

	elbs = `elbs (==1?)

For blue-green deployment porter needs to know into which elb to promote
instances.

This value is REQUIRED for ELB-based topologies and IGNORED for other topologies`

	elb_tag = `tag (==1!)

tag is used by porter build promote command to select which elb to promote
instances into.`

	elb_name = `name (==1!)

The name of the elb found in the AWS console.`
)

var Config cli.Command

func init() {
	Config = &cmd.Default{
		NameStr:      "config",
		ShortHelpStr: "./porter/config help",
		LongHelpStr:  configFileLongHelp,
		SubCommandList: []cli.Command{
			&cmd.Default{
				NameStr:      "service_name",
				ShortHelpStr: "(==1!)",
				LongHelpStr:  service_name,
			},
			&cmd.Default{
				NameStr:      "porter_version",
				ShortHelpStr: "(==1!)",
				LongHelpStr:  porter_version,
			},
			&cmd.Default{
				NameStr:      "containers",
				ShortHelpStr: "(>=1?)",
				LongHelpStr:  containers,
				SubCommandList: []cli.Command{
					&cmd.Default{
						NameStr:      "topology",
						ShortHelpStr: "(==1!)",
						LongHelpStr:  topology,
					},
					&cmd.Default{
						NameStr:      "inet_port",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  inet_port,
					},
					&cmd.Default{
						NameStr:      "health_check",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  health_check,
					},
				},
			},
			&cmd.Default{
				NameStr:      "environments",
				ShortHelpStr: "(>=1!)",
				LongHelpStr:  environments,
				SubCommandList: []cli.Command{
					&cmd.Default{
						NameStr:      "role_arn",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  role_arn,
					},
					&cmd.Default{
						NameStr:      "stack_definition_path",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  stack_definition_path,
					},
					&cmd.Default{
						NameStr:      "instance_count",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  instance_count,
					},
					&cmd.Default{
						NameStr:      "instance_type",
						ShortHelpStr: "(==1?)",
						LongHelpStr:  instance_type,
					},
					&cmd.Default{
						NameStr:      "blackout_windows",
						ShortHelpStr: "(>=1?)",
						LongHelpStr:  blackout_windows,
					},
					&cmd.Default{
						NameStr:      "regions",
						ShortHelpStr: "(>=1!)",
						LongHelpStr:  regions,
						SubCommandList: []cli.Command{
							&cmd.Default{
								NameStr:      "role_arn",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  role_arn,
							},
							&cmd.Default{
								NameStr:      "ssl_cert_arn",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  ssl_cert_arn,
							},
							&cmd.Default{
								NameStr:      "hosted_zone_name",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  hosted_zone_name,
							},
							&cmd.Default{
								NameStr:      "s3_bucket",
								ShortHelpStr: "(==1!)",
								LongHelpStr:  s3_bucket,
							},
							&cmd.Default{
								NameStr:      "key_pair_name",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  key_pair_name,
							},
							&cmd.Default{
								NameStr:      "vpc_id",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  vpc_id,
							},
							&cmd.Default{
								NameStr:      "azs",
								ShortHelpStr: "(>=1!)",
								LongHelpStr:  azs,
								SubCommandList: []cli.Command{
									&cmd.Default{
										NameStr:      "subnet_id",
										ShortHelpStr: "(==1?)",
										LongHelpStr:  subnet_id,
									},
								},
							},
							&cmd.Default{
								NameStr:      "elbs",
								ShortHelpStr: "(==1?)",
								LongHelpStr:  elbs,
								SubCommandList: []cli.Command{
									&cmd.Default{
										NameStr:      "tag",
										ShortHelpStr: "(==1?)",
										LongHelpStr:  elb_tag,
									},
									&cmd.Default{
										NameStr:      "name",
										ShortHelpStr: "(==1?)",
										LongHelpStr:  elb_name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
