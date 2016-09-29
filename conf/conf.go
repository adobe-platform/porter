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

//
// Changing configuration fields or semantics?
// Make sure to update commands/help/files/config.go as well
//
package conf

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/stdin"
	"github.com/inconshreveable/log15"
	yaml "gopkg.in/yaml.v2"
)

const (
	Topology_Inet   = "inet"
	Topology_Worker = "worker"
	Topology_Cron   = "cron"
)

// NOTE: It's important to keep a reserved character so that if any of these
// things need to be concatenated we can split on the reserved character to
// get the original values unambiguously. That character is - so be very careful
// where it's allowed. Currently service names allow - in them meaning they can
// only be used on the beginning or end of a string
var (
	serviceNameRegex     = regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9]*$`)
	roleARNRegex         = regexp.MustCompile(`^arn:aws:iam::\d+:role`)
	environmentNameRegex = regexp.MustCompile(`^[0-9a-zA-Z]+$`)
	healthMethodRegex    = regexp.MustCompile(`^GET$`)
	porterVersionRegex   = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	vpcIdRegex           = regexp.MustCompile(`^vpc-(\d|\w){8}$`)
	subnetIdRegex        = regexp.MustCompile(`^subnet-(\d|\w){8}$`)

	// https://github.com/docker/docker/blob/v1.11.2/utils/names.go#L6
	// minus '-' which is reserved
	containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.]+$`)
)

type (
	// Config supports multi-container deployments
	Config struct {
		ServiceName    string `yaml:"service_name"`
		ServiceVersion string
		PorterVersion  string         `yaml:"porter_version"`
		Environments   []*Environment `yaml:"environments"`
		Slack          Slack          `yaml:"slack"`
		Hooks          Hooks          `yaml:"hooks"`
	}

	Container struct {
		Name            string `yaml:"name"`
		OriginalName    string
		Topology        string       `yaml:"topology"`
		InetPort        int          `yaml:"inet_port"`
		Uid             *int         `yaml:"uid"`
		ReadOnly        *bool        `yaml:"read_only"`
		Dockerfile      string       `yaml:"dockerfile"`
		DockerfileBuild string       `yaml:"dockerfile_build"`
		HealthCheck     *HealthCheck `yaml:"health_check"`
		SrcEnvFile      *SrcEnvFile  `yaml:"src_env_file"`
	}

	SrcEnvFile struct {
		S3Key    string   `yaml:"s3_key"`
		S3Bucket string   `yaml:"s3_bucket"`
		S3Region string   `yaml:"s3_region"`
		ExecName string   `yaml:"exec_name"`
		ExecArgs []string `yaml:"exec_args"`
	}

	HealthCheck struct {
		Method string `yaml:"method"`
		Path   string `yaml:"path"`
	}

	Environment struct {
		Name                string           `yaml:"name"`
		StackDefinitionPath string           `yaml:"stack_definition_path"`
		RoleARN             string           `yaml:"role_arn"`
		InstanceCount       uint             `yaml:"instance_count"`
		InstanceType        string           `yaml:"instance_type"`
		BlackoutWindows     []BlackoutWindow `yaml:"blackout_windows"`
		Regions             []*Region        `yaml:"regions"`
	}

	BlackoutWindow struct {
		StartTime string `yaml:"start_time"`
		EndTime   string `yaml:"end_time"`
	}

	Region struct {
		Name                string             `yaml:"name"`
		StackDefinitionPath string             `yaml:"stack_definition_path"`
		VpcId               string             `yaml:"vpc_id"`
		AZs                 []AvailabilityZone `yaml:"azs"`
		ELBs                []*ELB             `yaml:"elbs"`
		ELB                 string             `yaml:"elb"`
		RoleARN             string             `yaml:"role_arn"`
		AutoScalingGroup    *AutoScalingGroup  `yaml:"auto_scaling_group"`
		SSLCertARN          string             `yaml:"ssl_cert_arn"`
		HostedZoneName      string             `yaml:"hosted_zone_name"`
		KeyPairName         string             `yaml:"key_pair_name"`
		S3Bucket            string             `yaml:"s3_bucket"`
		SSEKMSKeyId         *string            `yaml:"sse_kms_key_id"`
		Containers          []*Container       `yaml:"containers"`
	}

	AutoScalingGroup struct {
		SecurityGroupEgress []SecurityGroupEgress `yaml:"security_group_egress"`
		SecretsExecName     string                `yaml:"secrets_exec_name"`
		SecretsExecArgs     []string              `yaml:"secrets_exec_args"`
	}

	SecurityGroupEgress struct {
		CidrIp                     string `yaml:"cidr_ip" json:"CidrIp,omitempty"`
		FromPort                   int    `yaml:"from_port" json:"FromPort"`
		IpProtocol                 string `yaml:"ip_protocol" json:"IpProtocol,omitempty"`
		DestinationSecurityGroupId string `yaml:"destination_security_group_id" json:"DestinationSecurityGroupId,omitempty"`
		ToPort                     int    `yaml:"to_port" json:"ToPort"`
	}

	AvailabilityZone struct {
		Name     string `yaml:"name"`
		SubnetID string `yaml:"subnet_id"`
	}

	Hooks struct {
		PrePack       []Hook `yaml:"pre_pack"`
		PostPack      []Hook `yaml:"post_pack"`
		PreProvision  []Hook `yaml:"pre_provision"`
		PostProvision []Hook `yaml:"post_provision"`
		PrePromote    []Hook `yaml:"pre_promote"`
		PostPromote   []Hook `yaml:"post_promote"`
		PrePrune      []Hook `yaml:"pre_prune"`
		PostPrune     []Hook `yaml:"post_prune"`
		EC2Bootstrap  []Hook `yaml:"ec2_bootstrap"`
	}

	Hook struct {
		Repo        string            `yaml:"repo"`
		Ref         string            `yaml:"ref"`
		Dockerfile  string            `yaml:"dockerfile"`
		Environment map[string]string `yaml:"environment"`
		Concurrent  bool              `yaml:"concurrent"`
	}

	Slack struct {
		PackSuccessHook      string `yaml:"pack_success_webhook_url"`
		PackFailureHook      string `yaml:"pack_failure_webhook_url"`
		ProvisionSuccessHook string `yaml:"provision_success_webhook_url"`
		ProvisionFailureHook string `yaml:"provision_failure_webhook_url"`
		PromoteSuccessHook   string `yaml:"promote_success_webhook_url"`
		PromoteFailureHook   string `yaml:"promote_failure_webhook_url"`
	}

	ELB struct {
		ELBTag string `yaml:"tag"`
		Name   string `yaml:"name"`
	}
)

func (recv *Config) GetEnvironment(envName string) (*Environment, error) {
	for _, env := range recv.Environments {
		if env.Name == envName {
			return env, nil
		}
	}
	return nil, fmt.Errorf("Environment %s doesn't exist in the config", envName)
}

// Convention over configuration
func (recv *Config) SetDefaults() {

	for _, env := range recv.Environments {
		if env.InstanceCount == 0 {
			env.InstanceCount = 1
		}

		if env.InstanceType == "" {
			env.InstanceType = "m3.medium"
		}

		for _, region := range env.Regions {

			if region.ELB != "" {
				if region.ELBs == nil {
					region.ELBs = make([]*ELB, 0)
				}
				region.ELBs = append(region.ELBs, &ELB{
					Name: region.ELB,
				})
			}

			if len(region.Containers) == 0 {
				defaultContainer := &Container{}
				region.Containers = append(region.Containers, defaultContainer)
			}

			for _, container := range region.Containers {

				if container.Dockerfile == "" {
					container.Dockerfile = "Dockerfile"
				}
				if container.DockerfileBuild == "" {
					container.DockerfileBuild = "Dockerfile.build"
				}

				if container.Topology == Topology_Inet {

					if container.HealthCheck == nil {
						container.HealthCheck = &HealthCheck{}
					}
					if container.HealthCheck.Method == "" {
						container.HealthCheck.Method = "GET"
					}
					if container.HealthCheck.Path == "" {
						container.HealthCheck.Path = "/health"
					}
				}
			}

			if len(region.Containers) == 1 {

				if region.Containers[0].Topology == "" {
					region.Containers[0].Topology = Topology_Inet
				}

				if region.Containers[0].Name == "" {
					region.Containers[0].Name = "primary"
				}
			}
		}
	}
}

func (recv *Config) Print() {
	fmt.Println("service_name", recv.ServiceName)
	fmt.Println("porter_version", recv.PorterVersion)

	fmt.Println(".Hooks")
	printHooks(constants.HookPrePack, recv.Hooks.PrePack)
	printHooks(constants.HookPostPack, recv.Hooks.PostPack)
	printHooks(constants.HookPreProvision, recv.Hooks.PreProvision)
	printHooks(constants.HookPostProvision, recv.Hooks.PostProvision)
	printHooks(constants.HookPrePromote, recv.Hooks.PrePromote)
	printHooks(constants.HookPostPromote, recv.Hooks.PostPromote)
	printHooks(constants.HookPrePrune, recv.Hooks.PrePrune)
	printHooks(constants.HookPostPrune, recv.Hooks.PostPrune)
	printHooks(constants.HookEC2Bootstrap, recv.Hooks.EC2Bootstrap)

	fmt.Println(".Environments")
	for _, environment := range recv.Environments {
		fmt.Println("- .Name", environment.Name)
		fmt.Println("  .StackDefinitionPath", environment.StackDefinitionPath)
		fmt.Println("  .RoleARN", environment.RoleARN)
		fmt.Println("  .InstanceCount", environment.InstanceCount)
		fmt.Println("  .InstanceType", environment.InstanceType)

		fmt.Println("  .Regions")
		for _, region := range environment.Regions {
			fmt.Println("  - .Name", region.Name)
			fmt.Println("    .VpcId", region.VpcId)
			fmt.Println("    .RoleARN", region.RoleARN)
			fmt.Println("    .KeyPairName", region.KeyPairName)
			fmt.Println("    .S3Bucket", region.S3Bucket)

			fmt.Println("      .AZs")
			for _, az := range region.AZs {
				fmt.Println("      - .Name", az.Name)
				fmt.Println("        .SubnetID", az.SubnetID)
			}

			fmt.Println("      .ELB", region.ELB)

			fmt.Println("      .ELBs")
			for _, elb := range region.ELBs {
				fmt.Println("      - .Tag", elb.ELBTag)
				fmt.Println("        .Name", elb.Name)
			}

			fmt.Println("      .Containers")
			for _, container := range region.Containers {
				fmt.Println("      - .Name", container.Name)
				fmt.Println("        .InetPort", container.InetPort)

				if container.Topology == Topology_Inet {
					fmt.Println("        .HealthCheck.Method", container.HealthCheck.Method)
					fmt.Println("        .HealthCheck.Path", container.HealthCheck.Path)
				}

				if container.SrcEnvFile == nil {
					fmt.Println("        .SrcEnvFile nil")
				} else {
					fmt.Println("        .SrcEnvFile.S3Key", container.SrcEnvFile.S3Key)
					fmt.Println("        .SrcEnvFile.S3Bucket", container.SrcEnvFile.S3Bucket)
					fmt.Println("        .SrcEnvFile.S3Region", container.SrcEnvFile.S3Region)
					fmt.Println("        .SrcEnvFile.ExecName", container.SrcEnvFile.ExecName)
					fmt.Println("        .SrcEnvFile.ExecArgs", container.SrcEnvFile.ExecArgs)
				}
			}
		}
	}
}

func printHooks(name string, hooks []Hook) {
	fmt.Println(name)
	for _, hook := range hooks {
		fmt.Println("- .Repo", hook.Repo)
		fmt.Println("  .Ref", hook.Ref)
		fmt.Println("  .Dockerfile", hook.Dockerfile)
		fmt.Println("  .Environment")
		if hook.Environment != nil {
			for envKey, envValue := range hook.Environment {
				if envValue == "" {
					envValue = os.Getenv(envKey)
				}
				fmt.Println("    " + envKey + "=" + envValue)
			}
		}
	}
}

func GetConfig(log log15.Logger, validate bool) (config *Config, success bool) {
	var parseConfigSuccess bool

	file, err := os.Open(constants.ConfigPath)
	if err != nil {
		log.Error("Failed to open "+constants.ConfigPath, "Error", err)
		return
	}
	defer file.Close()

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error("Failed to read "+constants.ConfigPath, "Error", err)
		return
	}

	config, parseConfigSuccess = parseConfig(log, configBytes)
	if !parseConfigSuccess {
		return
	}

	config.SetDefaults()

	if validate {

		err = config.Validate()
		if err != nil {
			log.Error("Config validation", "Error", err)
			return
		}
	}

	success = true
	return
}

func GetStdinConfig(log log15.Logger) (config *Config, success bool) {

	configBytes, err := stdin.GetBytes()
	if err != nil {
		log.Error("Failed to read from stdin", "Error", err)
		return
	}

	config, success = parseConfig(log, configBytes)
	return
}

func GetAlteredConfig(log log15.Logger) (*Config, bool) {

	file, err := os.Open(constants.AlteredConfigPath)
	if err != nil {
		log.Error("Failed to open "+constants.AlteredConfigPath, "Error", err)
		return nil, false
	}
	defer file.Close()

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error("Failed to read "+constants.AlteredConfigPath, "Error", err)
		return nil, false
	}

	return parseConfig(log, configBytes)
}

func parseConfig(log log15.Logger, configBytes []byte) (config *Config, success bool) {
	config = &Config{}

	err := yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.Error("Failed to decode config", "Error", err)
		return
	}

	success = true
	return
}
