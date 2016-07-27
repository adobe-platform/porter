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
package constants

import (
	"os"
	"time"
)

const (
	ProgramName = "porter"

	TempDir                    = ".porter-tmp"
	PorterDir                  = ".porter"
	ConfigPath                 = ".porter/config"
	HookDir                    = ".porter/hooks"
	PayloadWorkingDir          = TempDir + "/payload"
	PayloadPath                = TempDir + "/payload.tar.gz"
	PackOutputPath             = TempDir + "/pack_output.json"
	ProvisionOutputPath        = TempDir + "/provision_output.json"
	CreateStackOutputPath      = TempDir + "/create_stack_output.json"
	CloudFormationTemplatePath = TempDir + "/CloudFormationTemplate.json"
	EnvFile                    = "/dockerfile.env"

	EnvConfig                    = "DEBUG_CONFIG"
	EnvDebugAws                  = "DEBUG_AWS"
	EnvLogDebug                  = "LOG_DEBUG"
	EnvStackCreation             = "STACK_CREATION_TIMEOUT"
	EnvStackCreationPollInterval = "STACK_CREATION_POLL_INTERVAL"
	EnvNoDockerOverride          = "NO_DOCKER_OVERRIDE"
	EnvNoLogColor                = "NO_LOG_COLOR"
	EnvDevMode                   = "DEV_MODE"

	HookPrePack       = "pre-pack"
	HookPostPack      = "post-pack"
	HookPreProvision  = "pre-provision"
	HookPostProvision = "post-provision"
	HookPrePromote    = "pre-promote"
	HookPostPromote   = "post-promote"
	HookPrePrune      = "pre-prune"
	HookPostPrune     = "post-prune"
	HookEC2Bootstrap  = "ec2-bootstrap"

	// The relative path from the service payload to the serialized *conf.Config
	ServicePayloadConfigPath = "config.yaml"

	// The relative path from the repo root to the serialized *conf.Config
	AlteredConfigPath     = TempDir + "/" + ServicePayloadConfigPath
	PackPayloadConfigPath = PayloadWorkingDir + "/" + ServicePayloadConfigPath

	EC2MetadataURL  = "http://169.254.169.254/latest/meta-data"
	AmazonLinuxUser = "ec2-user"

	HAProxyConfigPath      = "/etc/haproxy/haproxy.cfg"
	HAProxyConfigPerms     = 0644
	HAProxyStatsUsername   = "da05bb59715c617c8cb48666975307ed"
	HAProxyStatsPassword   = "17ece40f81292b2dfd8afe1a6990a506"
	HAProxyStatsUri        = "/admin?stats"
	HAProxyStatsUrl        = "http://localhost" + HAProxyStatsUri
	HAProxyIpBlacklistPath = "/var/lib/haproxy/ip_blacklist.txt"

	PorterDaemonInitPath   = "/etc/init/porterd.conf"
	PorterDaemonInitPerms  = 0644
	PorterDaemonBindPort   = "3001"
	PorterDaemonHealthPath = "/health"

	RsyslogConfigPath       = "/etc/rsyslog.conf"
	RsyslogPorterConfigPath = "/etc/rsyslog.d/21-porter.conf"
	RsyslogConfigPerms      = 0644

	// Porter tags used to follow the AWS colon-delimited convention but this
	// doesn't work well in Datadog because everything is flattened under the
	// top-level key. Use hyphen-delimited keys for tags we care about so
	// they're properly parsed by Datadog
	AwsCfnLogicalIdTag                    = "aws:cloudformation:logical-id"
	AwsCfnStackNameTag                    = "aws:cloudformation:stack-name"
	AwsCfnStackIdTag                      = "aws:cloudformation:stack-id"
	PorterWaitConditionHandleLogicalIdTag = "porter:aws:cloudformation:waitconditionhandle:logical-id"
	PorterEnvironmentTag                  = "porter-config-environment"
	PorterServiceNameTag                  = "porter-service-name"
	PorterStackIdTag                      = "porter-aws-cloudformation-stack-id"

	// Replaced by the release_porter script.
	//
	// Don't change this.
	Version   = "%%VERSION%%"
	BinaryUrl = "%%BINARY_URL%%"

	ParameterServiceName = "PorterServiceName"
	ParameterEnvironment = "PorterEnvironment"
	ParameterStackName   = "PorterStackName"
	ParameterSecretsKey  = "PorterSecretsKey"
	MappingRegionToAMI   = "RegionToAMI"

	HC_HealthyThreshold   = 3
	HC_Interval           = 5
	HC_Timeout            = HC_Interval - 2
	HC_UnhealthyThreshold = 2

	// A key in resource metadata to tag security groups that should be
	// associated with a AWS::ElasticLoadBalancing::LoadBalancer
	MetadataElb = "elb-lb-sg"

	// A key in resource metadata to tag security groups that should be
	// associated with a AWS::AutoScaling::LaunchConfiguration
	MetadataAsLc = "as-lc-sg"

	MetadataAsEnvFiles = "env_files"

	ElbSgLogicalName = "InetToElb"

	// http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-hup.html#cfn-hup-config-file
	CfnHupPollIntervalMinutes = 1

	DstELBSecurityGroup = "DestinationELBToInstance"

	ContainerUserUid = "601"

	DockerBinaryDarwinURL = "https://get.docker.com/builds/Darwin/x86_64/docker-1.7.1"
)

var (
	InetBindPorts    []uint16
	AwsRegions       map[string]interface{}
	AwsInstanceTypes map[string]interface{}
)

func StackCreationTimeout() time.Duration {
	if dur, err := time.ParseDuration(os.Getenv(EnvStackCreation)); err == nil {
		// clamp duration to sts:AssumeRole session length bounds
		if dur < 900*time.Second {
			dur = 900 * time.Second
		}

		if dur > 1*time.Hour {
			dur = 1 * time.Hour
		}
		return dur
	}

	return 20 * time.Minute
}

func StackCreationPollInterval() time.Duration {
	if dur, err := time.ParseDuration(os.Getenv(EnvStackCreationPollInterval)); err == nil {
		return dur
	}
	return 10 * time.Second
}

func init() {
	InetBindPorts = []uint16{
		80,   // HTTP
		8080, // HTTP (SSL termination)
	}

	AwsRegions = map[string]interface{}{
		"ap-northeast-1": nil,
		"ap-northeast-2": nil,
		"ap-southeast-1": nil,
		"ap-southeast-2": nil,
		"eu-central-1":   nil,
		"eu-west-1":      nil,
		"sa-east-1":      nil,
		"us-east-1":      nil,
		"us-west-1":      nil,
		"us-west-2":      nil,
	}

	AwsInstanceTypes = map[string]interface{}{
		"t2.nano":   nil,
		"t2.micro":  nil,
		"t2.small":  nil,
		"t2.medium": nil,
		"t2.large":  nil,

		"m4.medium":   nil,
		"m4.large":    nil,
		"m4.xlarge":   nil,
		"m4.2xlarge":  nil,
		"m4.10xlarge": nil,

		"m3.2xlarge": nil,
		"m3.large":   nil,
		"m3.medium":  nil,
		"m3.xlarge":  nil,

		"c3.large":   nil,
		"c3.xlarge":  nil,
		"c3.2xlarge": nil,
		"c3.4xlarge": nil,
		"c3.8xlarge": nil,

		"r3.large":   nil,
		"r3.xlarge":  nil,
		"r3.2xlarge": nil,
		"r3.4xlarge": nil,
		"r3.8xlarge": nil,

		"g2.2xlarge": nil,
		"g2.8xlarge": nil,

		"i2.xlarge":  nil,
		"i2.2xlarge": nil,
		"i2.4xlarge": nil,
		"i2.8xlarge": nil,

		"d2.xlarge":  nil,
		"d2.2xlarge": nil,
		"d2.4xlarge": nil,
		"d2.8xlarge": nil,
	}
}
