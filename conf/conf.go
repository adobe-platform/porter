/*
 * (c) 2016-2017 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
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
	"path/filepath"
	"regexp"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/stdin"
	"gopkg.in/inconshreveable/log15.v2"
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
	commonNameRegex      = regexp.MustCompile(`^(\w+\.)?[a-z]+$`)

	// https://github.com/docker/docker/blob/v1.11.2/utils/names.go#L6
	// minus '-' which is reserved
	containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.]+$`)
)

type (
	// Config supports multi-container deployments
	Config struct {
		ServiceName    string `yaml:"service_name"`
		ServiceVersion string
		PorterVersion  string            `yaml:"porter_version"`
		Environments   []*Environment    `yaml:"environments"`
		Slack          Slack             `yaml:"slack"`
		Hooks          map[string][]Hook `yaml:"hooks"`

		HAProxyStatsUsername string `yaml:"haproxy_stats_username"`
		HAProxyStatsPassword string `yaml:"haproxy_stats_password"`
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
		Hotswap             bool             `yaml:"hot_swap"`
		InstanceCount       uint             `yaml:"instance_count"`
		InstanceType        string           `yaml:"instance_type"`
		BlackoutWindows     []BlackoutWindow `yaml:"blackout_windows"`
		Regions             []*Region        `yaml:"regions"`

		HAProxy HAProxy `yaml:"haproxy"`

		// From the client's perspective this relates to SG creation and ELB
		// inspection that allows the 2 ELBs to communicate with EC2 instances.
		// From porter's perspective this is just a signal to create them so
		// further transformations can happen
		CreateSecurityGroups *bool `yaml:"autowire_security_groups"`
	}

	HAProxy struct {
		// Capture headers for logging
		ReqHeaderCaptures []HeaderCapture `yaml:"request_header_captures"`
		ResHeaderCaptures []HeaderCapture `yaml:"response_header_captures"`
		Log               *bool           `yaml:"log"`
		Compression       bool            `yaml:"compression"`
		CompressTypes     []string        `yaml:"compress_types"`
		SSL               SSL             `yaml:"ssl"`
	}

	SSL struct {
		CertDirectory  string `yaml:"cert_directory"`
		CertPath       string
		HTTPS_Only     bool `yaml:"https_only"`
		HTTPS_Redirect bool `yaml:"https_redirect"`

		Pem *Pem `yaml:"pem"`
	}

	Pem struct {
		SecretsExecName string   `yaml:"secrets_exec_name"`
		SecretsExecArgs []string `yaml:"secrets_exec_args"`
	}

	HeaderCapture struct {
		Header string `yaml:"header"`
		Length int    `yaml:"length"`
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
		AutoScalingGroup    AutoScalingGroup   `yaml:"auto_scaling_group"`
		SSLCertARN          string             `yaml:"ssl_cert_arn"`
		HostedZoneName      string             `yaml:"hosted_zone_name"`
		KeyPairName         string             `yaml:"key_pair_name"`
		S3Bucket            string             `yaml:"s3_bucket"`
		SSEKMSKeyId         *string            `yaml:"sse_kms_key_id"`
		Containers          []*Container       `yaml:"containers"`
		InstanceCount       uint               `yaml:"instance_count"`
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

	Hook struct {
		Repo         string            `yaml:"repo"`
		Ref          string            `yaml:"ref"`
		Dockerfile   string            `yaml:"dockerfile"`
		Environment  map[string]string `yaml:"environment"`
		Concurrent   bool              `yaml:"concurrent"`
		RunCondition string            `yaml:"run_condition"`
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

func (recv HAProxy) UsingSSL() bool {
	return recv.SSL.Pem != nil
}

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

	for _, hooks := range recv.Hooks {

		for i := 0; i < len(hooks); i++ {

			if hooks[i].RunCondition == "" {
				hooks[i].RunCondition = constants.HRC_Pass
			}
		}
	}

	for _, env := range recv.Environments {
		if env.InstanceCount == 0 {
			env.InstanceCount = 1
		}

		if env.InstanceType == "" {
			env.InstanceType = "m3.medium"
		}

		if len(env.HAProxy.CompressTypes) > 0 {
			env.HAProxy.Compression = true
		}

		if env.HAProxy.SSL.CertDirectory == "" {
			env.HAProxy.SSL.CertDirectory = "/etc/ssl/certs/"
		}

		env.HAProxy.SSL.CertPath = filepath.Join(env.HAProxy.SSL.CertDirectory, "porter.pem")

		// this is only for porterd which isn't currently informed of HTTPS_Only
		// porterd initially connects to http but will follow redirects
		if env.HAProxy.SSL.HTTPS_Only {
			env.HAProxy.SSL.HTTPS_Redirect = true
		}

		for _, region := range env.Regions {

			if len(region.AutoScalingGroup.SecurityGroupEgress) == 0 {
				region.AutoScalingGroup.SecurityGroupEgress = []SecurityGroupEgress{
					{ // DNS
						CidrIp:     "0.0.0.0/0",
						IpProtocol: "udp",
						FromPort:   53,
						ToPort:     53,
					},
					{ // NTP
						CidrIp:     "0.0.0.0/0",
						IpProtocol: "udp",
						FromPort:   123,
						ToPort:     123,
					},
					{ // HTTP
						CidrIp:     "0.0.0.0/0",
						IpProtocol: "tcp",
						FromPort:   80,
						ToPort:     80,
					},
					{ // HTTPS
						CidrIp:     "0.0.0.0/0",
						IpProtocol: "tcp",
						FromPort:   443,
						ToPort:     443,
					},
				}
			}

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
	for hookName, hookVal := range recv.Hooks {
		printHooks(hookName, hookVal)
	}

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
	fmt.Println("  " + name)
	for _, hook := range hooks {
		fmt.Println("  - .Repo", hook.Repo)
		fmt.Println("    .Ref", hook.Ref)
		fmt.Println("    .Dockerfile", hook.Dockerfile)
		fmt.Println("    .Environment")
		if hook.Environment != nil {
			for envKey, envValue := range hook.Environment {
				if envValue == "" {
					envValue = os.Getenv(envKey)
				}
				fmt.Println("      " + envKey + "=" + envValue)
			}
		}
	}
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

func GetConfig(log log15.Logger, validate bool) (*Config, bool) {

	return getConfigFile(log, constants.ConfigPath, true, validate)
}

func GetAlteredConfig(log log15.Logger) (*Config, bool) {

	return getConfigFile(log, constants.AlteredConfigPath, false, false)
}

func GetHostConfig(log log15.Logger) (*Config, bool) {

	return getConfigFile(log, os.Getenv(constants.EnvConfigPath), false, false)
}

func getConfigFile(log log15.Logger, filePath string, setDefaults, validate bool) (config *Config, success bool) {
	var parseConfigSuccess bool

	file, err := os.Open(filePath)
	if err != nil {
		log.Error("os.Open", "FilePath", filePath, "Error", err)
		return
	}
	defer file.Close()

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error("ioutil.ReadAll", "FilePath", filePath, "Error", err)
		return
	}

	config, parseConfigSuccess = parseConfig(log, configBytes)
	if !parseConfigSuccess {
		return
	}

	if setDefaults {

		config.SetDefaults()
	}

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
