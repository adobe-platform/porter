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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/adobe-platform/porter/constants"
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
	containerNameRegex   = regexp.MustCompile(`^[0-9a-zA-Z]+$`)
	healthMethodRegex    = regexp.MustCompile(`^GET$`)
	porterVersionRegex   = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	vpcIdRegex           = regexp.MustCompile(`^vpc-(\d|\w){8}$`)
	subnetIdRegex        = regexp.MustCompile(`^subnet-(\d|\w){8}$`)
)

// config can come from os.Stdin which we can't read more than one so we must
// store the config we read so multiple calls to GetStdinConfig succeed
var stdinConfig *Config

type (
	// Config supports multi-container deployments
	Config struct {
		ServiceName   string         `yaml:"service_name"`
		PorterVersion string         `yaml:"porter_version"`
		Environments  []*Environment `yaml:"environments"`
		Slack         Slack          `yaml:"slack"`
		Hooks         Hooks          `yaml:"hooks"`
	}

	Container struct {
		Name        string       `yaml:"name"`
		Topology    string       `yaml:"topology"`
		InetPort    int          `yaml:"inet_port"`
		Primary     bool         `yaml:"primary"`
		Uid         *int         `yaml:"uid"`
		HealthCheck *HealthCheck `yaml:"health_check"`
		SrcEnvFile  *SrcEnvFile  `yaml:"src_env_file"`
		DstEnvFile  *DstEnvFile  `yaml:"dst_env_file"`
	}

	SrcEnvFile struct {
		S3Key    string `yaml:"s3_key"`
		S3Bucket string `yaml:"s3_bucket"`
		S3Region string `yaml:"s3_region"`
	}

	DstEnvFile struct {
		S3Bucket string `yaml:"s3_bucket"`
		KMSARN   string `yaml:"kms_arn"`
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
		SSLCertARN          string             `yaml:"ssl_cert_arn"`
		HostedZoneName      string             `yaml:"hosted_zone_name"`
		KeyPairName         string             `yaml:"key_pair_name"`
		S3Bucket            string             `yaml:"s3_bucket"`
		Containers          []*Container       `yaml:"containers"`
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
		Repo       string `yaml:"repo"`
		Ref        string `yaml:"ref"`
		Dockerfile string `yaml:"dockerfile"`
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

// inet is a superset of worker which are almost identical to cron
func (recv *Region) PrimaryTopology() (dominant string) {
	for _, container := range recv.Containers {
		switch container.Topology {
		case Topology_Inet:
			dominant = container.Topology
		case Topology_Worker:
			if dominant != Topology_Inet {
				dominant = container.Topology
			}
		}
	}
	return
}

func (recv *Environment) GetELBForRegion(reg string, elb string) (string, error) {
	region, err := recv.GetRegion(reg)
	if err != nil {
		return "", err
	}

	// always return this if defined. it supersedes the old scheme
	if region.ELB != "" {
		return region.ELB, nil
	}

	// backward compatibility with old scheme
	for _, loadBalancer := range region.ELBs {
		if loadBalancer.ELBTag == elb {
			return loadBalancer.Name, nil
		}
	}

	return "", fmt.Errorf("ELB tagged %s doesn't exist in the config for region %s", elb, reg)
}

func (recv *Environment) GetRegion(regionName string) (*Region, error) {
	for _, region := range recv.Regions {
		if region.Name == regionName {
			return region, nil
		}
	}
	return nil, fmt.Errorf("Region %s missing in environment %s", regionName, recv.Name)
}

func (recv *Environment) GetRoleARN(regionName string) (string, error) {
	region, err := recv.GetRegion(regionName)
	if err != nil {
		return "", err
	}

	if region.RoleARN != "" {
		return region.RoleARN, nil
	}

	if recv.RoleARN == "" {
		return "", errors.New("previous validation failure led to missing RoleARN")
	}

	return recv.RoleARN, nil
}

func (recv Environment) GetStackDefinitionPath(regionName string) (string, error) {
	region, err := recv.GetRegion(regionName)
	if err != nil {
		return "", err
	}

	if region.StackDefinitionPath != "" {
		return region.StackDefinitionPath, nil
	}

	return recv.StackDefinitionPath, nil
}

func (recv *Environment) IsWithinBlackoutWindow() error {
	now := time.Now()

	for _, window := range recv.BlackoutWindows {
		startTime, err := time.Parse(time.RFC3339, window.StartTime)
		if err != nil {
			return err
		}

		endTime, err := time.Parse(time.RFC3339, window.EndTime)
		if err != nil {
			return err
		}

		if startTime.After(endTime) {
			return errors.New("start_time is after end_time")
		}

		if now.After(startTime) && now.Before(endTime) {
			return errors.New(now.Format(time.RFC3339) + " is currently within a blackout window")
		}
	}

	return nil
}

func (recv *Config) Validate() (err error) {
	err = recv.ValidateTopLevelKeys()
	if err != nil {
		return
	}

	err = recv.ValidateEnvironments()
	if err != nil {
		return
	}

	return
}

func (recv *Config) ValidateTopLevelKeys() error {

	// TODO validate this doesn't have spaces and can be used as a key in S3
	// and wherever else we use it
	if !serviceNameRegex.MatchString(recv.ServiceName) {
		return errors.New("Invalid service_name")
	}

	if os.Getenv(constants.EnvDevMode) == "" &&
		!porterVersionRegex.MatchString(recv.PorterVersion) {
		return errors.New("Invalid porter_version")
	}

	return nil
}

func (recv *Config) ValidateEnvironments() error {
	if len(recv.Environments) == 0 {
		return errors.New("No environments defined")
	}

	for _, environment := range recv.Environments {

		if len(environment.Regions) == 0 {
			return errors.New("Environment [" + environment.Name + "] doesn't define any regions")
		}

		validateRegionRoleArn := true

		if environment.RoleARN != "" {

			validateRegionRoleArn = false
			if !roleARNRegex.MatchString(environment.RoleARN) {
				return errors.New("Invalid role_arn for environment " + environment.Name)
			}
		}

		for _, region := range environment.Regions {
			err := ValidateRegion(region, validateRegionRoleArn)
			if err != nil {
				return errors.New("Error in environment [" + environment.Name + "] " + err.Error())
			}
		}

		if _, exists := constants.AwsInstanceTypes[environment.InstanceType]; !exists {
			return errors.New("Invalid instance_type for environment [" + environment.Name + "]")
		}

		if !environmentNameRegex.MatchString(environment.Name) {
			return errors.New("Invalid name for environment [" + environment.Name + "]. Valid characters are [0-9a-zA-Z]")
		}
	}

	return nil
}

func ValidateRegion(region *Region, validateRoleArn bool) error {

	err := region.ValidateContainers()
	if err != nil {
		return err
	}

	if _, exists := constants.AwsRegions[region.Name]; !exists {
		return errors.New("Invalid region name " + region.Name)
	}

	if validateRoleArn && !roleARNRegex.MatchString(region.RoleARN) {
		return errors.New("Invalid role_arn for region " + region.Name)
	}

	// TODO validate characters
	if region.HostedZoneName != "" {
		// normalize with ending period
		region.HostedZoneName = strings.TrimRight(region.HostedZoneName, ".") + "."
	}

	// TODO validate the bucket prefix is one that S3 allows
	if region.S3Bucket == "" {
		return errors.New("Empty or missing s3_bucket")
	}

	if len(region.AZs) == 0 {
		return errors.New("Missing availability zone for region " + region.Name)
	}

	definedVPC := false
	if region.VpcId != "" {
		definedVPC = true
		if !vpcIdRegex.MatchString(region.VpcId) {
			return errors.New("Invalid vpc_id for region " + region.Name)
		}
	}

	for _, az := range region.AZs {
		if az.Name == "" {
			return errors.New("Empty AZ name for region " + region.Name)
		}

		if definedVPC {
			if !subnetIdRegex.MatchString(az.SubnetID) {
				return errors.New("Invalid subnet_id for region " + region.Name)
			}
		} else {
			if az.SubnetID != "" {
				return errors.New("Defined subnet_id but no vpc_id for region " + region.Name)
			}
		}
	}

	return nil
}

func (recv *Region) ValidateContainers() error {

	// TODO remove once we support multi-container deployment
	if len(recv.Containers) > 1 {
		return fmt.Errorf("Only supporting single-container deployments. Found %d container definitions", len(recv.Containers))
	}

	if len(recv.Containers) == 0 {

		return errors.New("No containers are defined. Was SetDefaults() run?")
	} else if len(recv.Containers) == 1 {
		// nothing to do. SetDefaults will mark a primary container
	} else {

		foundPrimary := false
		for _, container := range recv.Containers {

			if container.Primary {
				if foundPrimary {

					return errors.New("Only one primary container is allowed")
				} else {

					foundPrimary = true
				}
			}
		}

		if !foundPrimary {
			return errors.New("One container must be marked as primary")
		}
	}

	containerNames := make(map[string]interface{})
	for _, container := range recv.Containers {

		if container.SrcEnvFile == nil && container.DstEnvFile != nil {

			return errors.New("dst_env_file defined but src_env_file undefined")
		} else if container.SrcEnvFile != nil && container.DstEnvFile == nil {

			return errors.New("src_env_file defined but dst_env_file undefined")
		} else if container.SrcEnvFile != nil && container.DstEnvFile != nil {

			if container.SrcEnvFile.S3Bucket == "" {
				return errors.New("src_env_file missing s3_bucket")
			}

			if container.SrcEnvFile.S3Key == "" {
				return errors.New("src_env_file missing s3_key")
			}

			if container.DstEnvFile.S3Bucket == "" {
				return errors.New("dst_env_file missing s3_bucket")
			}

			if container.DstEnvFile.KMSARN == "" {
				return errors.New("dst_env_file missing kms_arn")
			}
		}

		if !containerNameRegex.MatchString(container.Name) {
			return errors.New("Invalid container name")
		}

		if _, exists := containerNames[container.Name]; exists {
			return fmt.Errorf("Duplicate container %s", container.Name)
		}

		if !healthMethodRegex.MatchString(container.HealthCheck.Method) {
			return fmt.Errorf("Invalid health check method %s on container %s", container.HealthCheck.Method, container.Name)
		}

		containerNames[container.Name] = nil

		switch container.Topology {
		case Topology_Inet:
			// valid
		default:
			return fmt.Errorf("Missing or invalid topology. Valid values are [%s]",
				Topology_Inet)
		}

		// TODO check if Dockerfile EXPOSEs more than one port.
		// if so, the ServicePort is required
		/*if container.ServicePort < 80 || container.ServicePort > 65535 {
			return fmt.Errorf("invalid service_port %d", container.ServicePort)
		}*/
	}

	return nil
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

			if len(region.Containers) == 1 {
				region.Containers[0].Primary = true

				if region.Containers[0].Name == "" {
					region.Containers[0].Name = "primary"
				}

				if region.Containers[0].Topology == "" {
					region.Containers[0].Topology = Topology_Inet
				}
			}
		}
	}
}

func (recv *Config) Print() {
	fmt.Println("service_name", recv.ServiceName)
	fmt.Println("porter_version", recv.PorterVersion)

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
				fmt.Println("        .Primary", container.Primary)

				fmt.Println("        .HealthCheck.Method", container.HealthCheck.Method)
				fmt.Println("        .HealthCheck.Path", container.HealthCheck.Path)
			}
		}
	}

}

func GetConfig(log log15.Logger) (*Config, bool) {

	file, err := os.Open(constants.ConfigPath)
	if err != nil {
		log.Error("Failed to open "+constants.ConfigPath, "Error", err)
		return nil, false
	}
	defer file.Close()

	config, success := readAndParse(log, file, constants.ConfigPath)
	if !success {
		return nil, false
	}

	config.SetDefaults()
	err = config.Validate()
	if err != nil {
		log.Error("Config validation", "Error", err)
		return nil, false
	}

	return config, true
}

func GetStdinConfig(log log15.Logger) (*Config, bool) {
	if stdinConfig != nil {
		stdinConfigCopy := *stdinConfig
		return &stdinConfigCopy, true
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Error("stat STDIN", "Error", err)
		return nil, false
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Error("Nothing to read from STDIN")
		return nil, false
	}

	config, success := readAndParse(log, os.Stdin, "STDIN")
	if success {
		configCopy := *config
		stdinConfig = &configCopy
		return config, true
	} else {
		return nil, false
	}
}

func GetAlteredConfig(log log15.Logger) (*Config, bool) {

	file, err := os.Open(constants.AlteredConfigPath)
	if err != nil {
		log.Error("Failed to open "+constants.AlteredConfigPath, "Error", err)
		return nil, false
	}
	defer file.Close()

	return readAndParse(log, file, constants.AlteredConfigPath)
}

func readAndParse(log log15.Logger, file *os.File, filePath string) (config *Config, success bool) {
	config = &Config{}

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error("Failed to read "+filePath, "Error", err)
		return
	}

	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.Error("Failed to decode "+filePath, "Error", err)
		return
	}

	success = true
	return
}
