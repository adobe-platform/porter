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
package conf

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adobe-platform/porter/constants"
)

func (recv *Config) Validate() (err error) {

	err = recv.ValidateRegistryConfig()
	if err != nil {
		return
	}

	err = recv.ValidateTopLevelKeys()
	if err != nil {
		return
	}

	err = recv.ValidateHooks()
	if err != nil {
		return
	}

	err = recv.ValidateEnvironments()
	if err != nil {
		return
	}

	return
}

func (recv *Config) ValidateRegistryConfig() error {
	dockerRegistry := os.Getenv(constants.EnvDockerRegistry)
	dockerRepository := os.Getenv(constants.EnvDockerRepository)
	dockerPullUsername := os.Getenv(constants.EnvDockerPullUsername)
	dockerPullPassword := os.Getenv(constants.EnvDockerPullPassword)
	dockerPushUsername := os.Getenv(constants.EnvDockerPushUsername)
	dockerPushPassword := os.Getenv(constants.EnvDockerPushPassword)

	if strings.Contains(dockerRegistry, "/") {
		return errors.New("slashes disallowed in " + constants.EnvDockerRegistry)
	}

	if dockerRegistry != "" && dockerRepository == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerRegistry, constants.EnvDockerRepository)
	}

	if dockerRepository != "" && dockerRegistry == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerRepository, constants.EnvDockerRegistry)
	}

	if dockerPullUsername != "" && dockerPullPassword == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerPullUsername, constants.EnvDockerPullPassword)
	}

	if dockerPullPassword != "" && dockerPullUsername == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerPullPassword, constants.EnvDockerPullUsername)
	}

	if dockerPushUsername != "" && dockerPushPassword == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerPushUsername, constants.EnvDockerPushPassword)
	}

	if dockerPushPassword != "" && dockerPushUsername == "" {
		return fmt.Errorf("%s defined: missing %s",
			constants.EnvDockerPushPassword, constants.EnvDockerPushUsername)
	}

	return nil
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

func (recv *Config) ValidateHooks() (err error) {

	for name, hookList := range recv.Hooks {

		for _, hook := range hookList {

			switch name {
			case constants.HookPostPack,
				constants.HookPostProvision,
				constants.HookPostHotswap,
				constants.HookPostPromote,
				constants.HookPostPrune:
			default:
				if hook.RunCondition != constants.HRC_Pass {
					return fmt.Errorf("A run_condition option is not valid for a %s hook", name)
				}
			}

			switch hook.RunCondition {
			case constants.HRC_Pass,
				constants.HRC_Fail,
				constants.HRC_Always:
			default:
				return fmt.Errorf("Invalid run_condition [%s] on a %s hook",
					hook.RunCondition, name)
			}

			if hook.Repo == "" {

				if hook.Dockerfile == "" {
					return fmt.Errorf("A %s hook has neither a dockerfile nor a repo", name)
				}
			} else {

				if hook.Ref == "" {
					return fmt.Errorf("A %s hook has a configured repo but no ref", name)
				}
			}
		}
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

		if environment.HAProxy.UsingSSL() {
			if environment.HAProxy.SSL.Pem == nil || environment.HAProxy.SSL.Pem.SecretsExecName == "" {
				return errors.New("haproxy ssl pem defined but no pem secrets_exec_name was defined")
			}

			if environment.HAProxy.SSL.HTTPS_Only {

				for _, region := range environment.Regions {
					if !region.HasELB() {
						return errors.New("https_only is incompatible with elb: none")
					}
				}
			}
		}

		var err error
		if environment.HAProxy.Timeout.Client_, err = time.ParseDuration(*environment.HAProxy.Timeout.Client); err != nil {
			return errors.New("ParseDuration(timeout_client) " + err.Error())
		}

		if environment.HAProxy.Timeout.Server_, err = time.ParseDuration(*environment.HAProxy.Timeout.Server); err != nil {
			return errors.New("ParseDuration(timeout_server) " + err.Error())
		}

		if environment.HAProxy.Timeout.Tunnel_, err = time.ParseDuration(*environment.HAProxy.Timeout.Tunnel); err != nil {
			return errors.New("ParseDuration(timeout_tunnel) " + err.Error())
		}

		if environment.HAProxy.Timeout.HttpRequest_, err = time.ParseDuration(*environment.HAProxy.Timeout.HttpRequest); err != nil {
			return errors.New("ParseDuration(timeout_http_request) " + err.Error())
		}

		if environment.HAProxy.Timeout.HttpKeepAlive_, err = time.ParseDuration(*environment.HAProxy.Timeout.HttpKeepAlive); err != nil {
			return errors.New("ParseDuration(timeout_http_keep_alive) " + err.Error())
		}

		if environment.HAProxy.Timeout.Client_ != environment.HAProxy.Timeout.Server_ {
			return errors.New("timeout_client != timeout_server")
		}

		if environment.Hotswap {

			for _, region := range environment.Regions {
				if len(region.ELBs) > 1 {
					return errors.New("hot_swap is incompatible with multiple elbs")
				}
			}
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

	if region.PrimaryTopology() != Topology_Inet && region.HasELB() {
		return errors.New("primary container topology can not have an elb for region " + region.Name)
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

	containerCount := len(recv.Containers)
	if containerCount == 0 {

		return errors.New("No containers are defined. Was SetDefaults() run?")
	}

	var healthCheckMethod string
	var healthCheckPath string

	containerNames := make(map[string]interface{})
	for _, container := range recv.Containers {

		if container.SrcEnvFile != nil {

			if container.SrcEnvFile.S3Bucket != "" ||
				container.SrcEnvFile.S3Key != "" {

				if container.SrcEnvFile.S3Bucket == "" {
					return errors.New("src_env_file missing s3_bucket")
				}

				if container.SrcEnvFile.S3Key == "" {
					return errors.New("src_env_file missing s3_key")
				}

			} else if container.SrcEnvFile.ExecName == "" {

				return errors.New("src_env_file missing exec_name")
			}
		}

		if container.PidsLimit < 1 {
			return errors.New("pids_limit must be greater than or equal to 1")
		}

		if containerCount > 1 && !containerNameRegex.MatchString(container.Name) {
			return errors.New("Invalid container name")
		}

		if _, exists := containerNames[container.Name]; exists {
			return fmt.Errorf("Duplicate container %s", container.Name)
		}

		if container.Topology == Topology_Inet {

			if !healthMethodRegex.MatchString(container.HealthCheck.Method) {
				return fmt.Errorf("Invalid health check method %s on container %s", container.HealthCheck.Method, container.Name)
			}

			if healthCheckMethod == "" {
				healthCheckMethod = container.HealthCheck.Method
			} else if healthCheckMethod != container.HealthCheck.Method {
				return fmt.Errorf("All inet containers must have the same health check")
			}

			if healthCheckPath == "" {
				healthCheckPath = container.HealthCheck.Path
			} else if healthCheckPath != container.HealthCheck.Path {
				return fmt.Errorf("All inet containers must have the same health check")
			}
		}

		containerNames[container.Name] = nil

		switch container.Topology {
		case Topology_Inet, Topology_Worker:
			// valid
		default:
			return fmt.Errorf("Missing or invalid topology. Valid values are [%s, %s]",
				Topology_Inet, Topology_Worker)
		}

		// TODO check if Dockerfile EXPOSEs more than one port.
		// if so, the ServicePort is required
		/*if container.ServicePort < 80 || container.ServicePort > 65535 {
			return fmt.Errorf("invalid service_port %d", container.ServicePort)
		}*/
	}

	return nil
}
