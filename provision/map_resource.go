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
package provision

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/cfn_template"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/provision_output"
)

// MapResource is a function that operates on the input resource
type MapResource func(*stackCreator, *cfn.Template, map[string]interface{}) bool

func (recv *stackCreator) mapResources(template *cfn.Template) (success bool) {

	ops := make(map[string][]MapResource)

	switch recv.region.PrimaryTopology() {
	case conf.Topology_Inet:
		ops[cfn.AutoScaling_LaunchConfiguration] = []MapResource{
			addASGSecurityGroups,
			setInstanceType,
			setKeyName,
			setIamInstanceProfile,
			setImageId,
			setAutoScalingLaunchConfigurationMetadata,
			setUserData,
		}
		ops[cfn.AutoScaling_AutoScalingGroup] = []MapResource{
			addAutoScaleGroupTags,
			setPoolSize,
			setAutoScalingGroupMultiAZ,
			setLaunchConfigurationName,
			setLoadBalancerNames,
		}
		ops[cfn.ElasticLoadBalancing_LoadBalancer] = []MapResource{
			addELBSecurityGroups,
			setELBMultiAZ,
			addHTTPSListener,
			addHTTPListener,
			setCrossZone,
			setConnectionDrainingPolicy,
			setHealthCheck,
		}
		ops[cfn.EC2_SecurityGroup] = []MapResource{
			setVpcId,
		}
		ops[cfn.CloudFormation_WaitCondition] = []MapResource{
			setTimeout,
			setCount,
			setDependsOnAutoScalingGroup,
			setHandle,
		}
		ops[cfn.IAM_InstanceProfile] = []MapResource{
			setRoleAndPath,
		}
		ops[cfn.IAM_Role] = []MapResource{
			addInlinePolicies,
		}

	case conf.Topology_Worker, conf.Topology_Cron:
		ops[cfn.AutoScaling_LaunchConfiguration] = []MapResource{
			addASGSecurityGroups,
			setInstanceType,
			setKeyName,
			setIamInstanceProfile,
			setImageId,
			setAutoScalingLaunchConfigurationMetadata,
			setUserData,
		}
		ops[cfn.AutoScaling_AutoScalingGroup] = []MapResource{
			addAutoScaleGroupTags,
			setPoolSize,
			setAutoScalingGroupMultiAZ,
			setLaunchConfigurationName,
		}
		ops[cfn.EC2_SecurityGroup] = []MapResource{
			setVpcId,
		}
		ops[cfn.CloudFormation_WaitCondition] = []MapResource{
			setTimeout,
			setCount,
			setDependsOnAutoScalingGroup,
			setHandle,
		}
		ops[cfn.IAM_InstanceProfile] = []MapResource{
			setRoleAndPath,
		}
		ops[cfn.IAM_Role] = []MapResource{
			addInlinePolicies,
		}
	}

	for key, value := range recv.templateTransforms {
		ops[key] = append(ops[key], value...)
	}

	for _, resourceRaw := range template.Resources {

		if resource, ok := resourceRaw.(map[string]interface{}); ok {

			if resourceType, ok := resource["Type"].(string); ok {

				if fns, ok := ops[resourceType]; ok {

					for _, fn := range fns {
						fnSuccess := fn(recv, template, resource)
						if !fnSuccess {
							return
						}
					}
				}
			}
		}
	}

	success = true
	return
}

// In EC2-Classic an ELB has a stable security group like this
//
//  "SourceSecurityGroup": {
//    "OwnerAlias": "amazon-elb",
//    "GroupName": "amazon-elb-sg"
//  }
//
// In a default VPC the security group varies and looks something like
//
//  "SourceSecurityGroup": {
//    "OwnerAlias": "123456789012", <- account number
//    "GroupName": "WhateverINamedMySecurityGroup"
//  }
//
// We can handle both by adding to the AWS::AutoScaling::LaunchConfiguration
// the SourceSecurityGroup of the ELB in which instances _will be_ promoted
//
// This won't work for a non-default VPC because it requires
// SourceSecurityGroupId
// http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-security-group-rule.html#cfn-ec2-security-group-rule-sourcesecuritygroupid
func addELBSecurityGroups(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	return addTaggedSecurityGroups(recv, template, resource, constants.MetadataElb)
}

func addASGSecurityGroups(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	return addTaggedSecurityGroups(recv, template, resource, constants.MetadataAsLc)
}

func addTaggedSecurityGroups(recv *stackCreator, template *cfn.Template, resource map[string]interface{}, metadataTag string) bool {
	var (
		props map[string]interface{}
		ok    bool

		securityGroups []interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if securityGroups, ok = props["SecurityGroups"].([]interface{}); !ok {
		securityGroups = make([]interface{}, 0)
	}

	logicalNameToSecurityGroup := template.GetResourcesByType(cfn.EC2_SecurityGroup)
	for logicalName, securityGroupRaw := range logicalNameToSecurityGroup {

		if securityGroup, ok := securityGroupRaw.(map[string]interface{}); ok {

			if metadata, ok := securityGroup["Metadata"].(map[string]interface{}); ok {

				if _, ok := metadata[metadataTag].(bool); ok {
					securityGroupRef := map[string]interface{}{
						"Ref": logicalName,
					}
					securityGroups = append(securityGroups, securityGroupRef)
				}
			}
		}
	}

	props["SecurityGroups"] = securityGroups
	return true
}

func setELBMultiAZ(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	azList := make([]string, 0)
	if recv.region.VpcId == "" {
		if _, exists := props["AvailabilityZones"]; !exists {
			delete(props, "Subnets")
			for _, az := range recv.region.AZs {
				azList = append(azList, az.Name)
			}
			props["AvailabilityZones"] = azList
		}
	} else {
		if _, exists := props["Subnets"]; !exists {
			delete(props, "AvailabilityZones")
			for _, az := range recv.region.AZs {
				azList = append(azList, az.SubnetID)
			}
			props["Subnets"] = azList
		}
	}
	return true
}

func setAutoScalingGroupMultiAZ(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	azList := make([]string, 0)
	if recv.region.VpcId == "" {
		if _, exists := props["AvailabilityZones"]; !exists {
			delete(props, "VPCZoneIdentifier")
			for _, az := range recv.region.AZs {
				azList = append(azList, az.Name)
			}
			props["AvailabilityZones"] = azList
		}
	} else {
		if _, exists := props["VPCZoneIdentifier"]; !exists {
			delete(props, "AvailabilityZones")
			for _, az := range recv.region.AZs {
				azList = append(azList, az.SubnetID)
			}
			props["VPCZoneIdentifier"] = azList
		}
	}
	return true
}

func setLaunchConfigurationName(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["LaunchConfigurationName"]; !exists {

		launchConfigurationName, err := template.GetResourceName(cfn.AutoScaling_LaunchConfiguration)
		if err != nil {
			recv.log.Error("template.GetResourceName", "Error", err)
			return
		}

		props["LaunchConfigurationName"] = map[string]interface{}{
			"Ref": launchConfigurationName,
		}
	}

	success = true
	return
}

func setLoadBalancerNames(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	var (
		props map[string]interface{}
		ok    bool

		loadBalancerNames []interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if loadBalancerNames, ok = props["LoadBalancerNames"].([]interface{}); !ok {
		loadBalancerNames = make([]interface{}, 0)
	}

	loadBalancerLogicalName, err := template.GetResourceName(cfn.ElasticLoadBalancing_LoadBalancer)
	if err != nil {
		recv.log.Error("template.GetResourceName", "Error", err)
		return
	}

	loadBalancerName := map[string]interface{}{
		"Ref": loadBalancerLogicalName,
	}
	loadBalancerNames = append(loadBalancerNames, loadBalancerName)

	props["LoadBalancerNames"] = loadBalancerNames

	success = true
	return
}

func setKeyName(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["KeyName"]; !exists && recv.region.KeyPairName != "" {
		props["KeyName"] = recv.region.KeyPairName
	}
	return true
}

func setUserData(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	autoScalingLaunchConfiguration, err := template.GetResourceName(cfn.AutoScaling_LaunchConfiguration)
	if err != nil {
		recv.log.Error("template.GetResourceName", "Error", err)
		return
	}

	userData, err := cfn_template.UserData(autoScalingLaunchConfiguration)
	if err != nil {
		recv.log.Error("cfn_template.UserData", "Error", err)
		return
	}

	props["UserData"] = userData

	success = true
	return
}

func setAutoScalingLaunchConfigurationMetadata(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {

	elbNames := make([]string, 0)
	for _, elb := range recv.region.ELBs {
		elbNames = append(elbNames, elb.Name)
	}
	elbCSV := strings.Join(elbNames, ",")

	var eC2BootstrapScript bytes.Buffer

	hookSuccess := hook.Execute(recv.log,
		constants.HookEC2Bootstrap,
		recv.environment.Name,
		[]provision_output.Region{
			{
				AWSRegion: recv.region.Name,
			},
		},
		func(opts *hook.Opts) {
			// opts.BuildStdout = nil
			// opts.BuildStderr = nil
			opts.RunStdout = &eC2BootstrapScript
			// opts.RunStderr = nil
		})
	if !hookSuccess {
		return
	}

	cfnInitContext := cfn_template.AWSCloudFormationInitCtx{
		PorterVersion: constants.Version,
		Environment:   recv.args.Environment,
		Region:        recv.region.Name,
		EnvFile:       constants.EnvFile,

		ServiceName:    recv.config.ServiceName,
		ServiceVersion: recv.serviceVersion,

		EC2BootstrapScript: eC2BootstrapScript.String(),

		Elbs: strconv.Quote(elbCSV),

		ServicePayloadBucket:     recv.region.S3Bucket,
		ServicePayloadKey:        recv.servicePayloadKey,
		ServicePayloadConfigPath: constants.ServicePayloadConfigPath,

		InetHealthCheckMethod: strconv.Quote(recv.region.HealthCheckMethod()),
		InetHealthCheckPath:   strconv.Quote(recv.region.HealthCheckPath()),

		PorterBinaryUrl: constants.BinaryUrl,

		DevMode:  os.Getenv(constants.EnvDevMode) != "",
		LogDebug: os.Getenv(constants.EnvLogDebug) != "",

		ContainerUserUid: constants.ContainerUserUid,
	}

	for _, container := range recv.region.Containers {
		cfnInitContext.ImageNames = append(cfnInitContext.ImageNames, container.Name)
	}

	autoScalingLaunchConfiguration, err := template.GetResourceName(cfn.AutoScaling_LaunchConfiguration)
	if err != nil {
		recv.log.Error("template.GetResourceName", "Error", err)
		return
	}

	cfnInitMetadata, err := cfn_template.AWSCloudFormationInit(autoScalingLaunchConfiguration, cfnInitContext)
	if err != nil {
		recv.log.Error("cfn_template.AWSCloudFormationInit", "Error", err)
		return
	}

	var metadata map[string]interface{}
	var ok bool
	if metadata, ok = resource["Metadata"].(map[string]interface{}); !ok {
		metadata = make(map[string]interface{})
		resource["Metadata"] = metadata
	}

	metadata["AWS::CloudFormation::Init"] = cfnInitMetadata

	success = true
	return
}

func setIamInstanceProfile(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["IamInstanceProfile"]; !exists {

		iamInstanceProfile, err := template.GetResourceName(cfn.IAM_InstanceProfile)
		if err != nil {
			recv.log.Error("template.GetResourceName", "Error", err)
			return
		}

		props["IamInstanceProfile"] = map[string]interface{}{
			"Ref": iamInstanceProfile,
		}
	}

	success = true
	return
}

func setImageId(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["ImageId"]; !exists {

		props["ImageId"] = cfn_template.ImageIdInMap(constants.MappingRegionToAMI)
	}
	return true
}

func setPoolSize(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["MinSize"]; !exists {
		props["MinSize"] = recv.environment.InstanceCount
	}

	if _, exists := props["MaxSize"]; !exists {
		props["MaxSize"] = recv.environment.InstanceCount
	}
	return true
}

func setCount(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["Count"]; !exists {
		props["Count"] = recv.environment.InstanceCount
	}
	return true
}

// The WaitCondition DependsOn the ASG because its timeout starts as soon as its
// created an the ASG is the last thing to be created so we want the timeout
// countdown to start as soon as all the other resources in the stack have been
// created
func setDependsOnAutoScalingGroup(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	if _, exists := resource["DependsOn"]; !exists {

		autoScalingGroup, err := template.GetResourceName(cfn.AutoScaling_AutoScalingGroup)
		if err != nil {
			recv.log.Error("template.GetResourceName", "Error", err)
			return
		}

		resource["DependsOn"] = autoScalingGroup
	}

	success = true
	return
}

func setInstanceType(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["InstanceType"]; !exists {
		props["InstanceType"] = recv.environment.InstanceType
	}
	return true
}

// TODO sha of service
func addAutoScaleGroupTags(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {

	var waitConditionHandle string

	additionalTags := []interface{}{
		map[string]interface{}{
			"Key":               constants.PorterEnvironmentTag,
			"Value":             recv.args.Environment,
			"PropagateAtLaunch": true,
		},
		map[string]interface{}{
			"Key":               constants.PorterServiceNameTag,
			"Value":             recv.config.ServiceName,
			"PropagateAtLaunch": true,
		},
	}

	waitConditionHandle, err := template.GetResourceName(cfn.CloudFormation_WaitConditionHandle)
	if err != nil || waitConditionHandle == "" {
		recv.log.Warn("Missing "+cfn.CloudFormation_WaitConditionHandle+" in the stack definition", "Error", err)
	} else {
		tag := map[string]interface{}{
			"Key":               constants.PorterWaitConditionHandleLogicalIdTag,
			"Value":             waitConditionHandle,
			"PropagateAtLaunch": true,
		}
		additionalTags = append(additionalTags, tag)
	}

	var (
		tags  []interface{}
		props map[string]interface{}
		ok    bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if tags, ok = props["Tags"].([]interface{}); !ok {
		tags = make([]interface{}, 0)
	}

	recv.log.Info("Adding tags to AWS::AutoScaling::AutoScalingGroup")

	nameTagValue, err := GetStackName(recv.config.ServiceName, recv.args.Environment, false)
	if err != nil {
		recv.log.Error("Failed to get stack name", "Error", err)
	} else {
		var nameTag map[string]interface{}

		for _, tag := range tags {
			if msi, ok := tag.(map[string]interface{}); ok {
				if msi["Key"] == "Name" {
					nameTag = tag.(map[string]interface{})
					break
				}
			}
		}

		if nameTag == nil {
			tag := map[string]interface{}{
				"Key":               "Name",
				"Value":             nameTagValue,
				"PropagateAtLaunch": true,
			}
			tags = append(tags, tag)
		} else {
			nameTag["Value"] = nameTagValue
			nameTag["PropagateAtLaunch"] = true
		}
	}

	for _, tag := range additionalTags {
		tags = append(tags, tag)
	}

	props["Tags"] = tags
	return true
}

func addHTTPSListener(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {

	if recv.region.SSLCertARN == "" {
		return true
	}

	recv.log.Info("ssl_cert_arn defined. Adding HTTPS listener to AWS::ElasticLoadBalancing::LoadBalancer")

	var (
		props map[string]interface{}
		ok    bool

		listeners []interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if listeners, ok = props["Listeners"].([]interface{}); !ok {
		listeners = make([]interface{}, 0)
	}

	for _, listener := range listeners {
		if msi, ok := listener.(map[string]interface{}); ok {
			if msi["Protocol"] == "HTTPS" ||
				msi["LoadBalancerPort"] == "443" ||
				msi["LoadBalancerPort"] == 443 {
				recv.log.Warn("HTTPS listener collision. Nothing to do")
				return true
			}
		}
	}

	httpsListener := map[string]interface{}{
		"LoadBalancerPort": "443",
		"InstancePort":     strconv.Itoa(int(constants.InetBindPorts[1])),
		"Protocol":         "HTTPS",
		"SSLCertificateId": recv.region.SSLCertARN,
	}
	listeners = append(listeners, httpsListener)

	props["Listeners"] = listeners
	return true
}

func addHTTPListener(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {

	var (
		listeners []interface{}
		props     map[string]interface{}
		ok        bool
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if listeners, ok = props["Listeners"].([]interface{}); !ok {
		listeners = make([]interface{}, 0)
	}

	for _, listener := range listeners {
		if msi, ok := listener.(map[string]interface{}); ok {
			if msi["Protocol"] == "HTTP" ||
				msi["LoadBalancerPort"] == "80" ||
				msi["LoadBalancerPort"] == 80 {
				recv.log.Warn("HTTP listener collision. Nothing to do")
				return true
			}
		}
	}

	httpListener := map[string]interface{}{
		"LoadBalancerPort": "80",
		"InstancePort":     constants.InetBindPorts[0],
		"Protocol":         "HTTP",
	}
	listeners = append(listeners, httpListener)

	props["Listeners"] = listeners
	return true
}

func setTimeout(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["Timeout"]; !exists {
		props["Timeout"] = constants.StackCreationTimeout().Seconds()
	}
	return true
}

func setCrossZone(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["CrossZone"]; !exists {
		props["CrossZone"] = true
	}
	return true
}

func setConnectionDrainingPolicy(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["ConnectionDrainingPolicy"]; !exists {
		props["ConnectionDrainingPolicy"] = map[string]interface{}{
			"Enabled": true,
			"Timeout": 300,
		}
	}
	return true
}

func setHealthCheck(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["HealthCheck"]; !exists {

		healthCheckTarget := "TCP:80"

		if recv.region.HealthCheckMethod() == "GET" {
			healthCheckTarget = "HTTP:80/" + strings.TrimPrefix(recv.region.HealthCheckPath(), "/")
		}

		props["HealthCheck"] = map[string]interface{}{
			"HealthyThreshold":   strconv.Itoa(constants.HC_HealthyThreshold),
			"UnhealthyThreshold": strconv.Itoa(constants.HC_UnhealthyThreshold),
			"Interval":           strconv.Itoa(constants.HC_Interval),
			"Timeout":            strconv.Itoa(constants.HC_Timeout),
			"Target":             healthCheckTarget,
		}
	}
	return true
}

func setHandle(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) (success bool) {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["Handle"]; !exists {

		waitConditionHandle, err := template.GetResourceName(cfn.CloudFormation_WaitConditionHandle)
		if err != nil {
			recv.log.Error("template.GetResourceName", "Error", err)
			return
		}

		props["Handle"] = map[string]interface{}{
			"Ref": waitConditionHandle,
		}
	}

	success = true
	return
}

func setVpcId(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if recv.region.VpcId != "" {
		if _, exists := props["VpcId"]; !exists {
			props["VpcId"] = recv.region.VpcId
		}
	}
	return true
}

func setRoleAndPath(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok    bool
		props map[string]interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["Path"]; !exists {
		props["Path"] = "/"
	}

	if _, exists := props["Roles"]; !exists {

		iamRole, err := template.GetResourceName(cfn.IAM_Role)
		if err != nil {
			recv.log.Error("template.GetResourceName", "Error", err)
			return false
		}

		props["Roles"] = []interface{}{
			map[string]interface{}{"Ref": iamRole},
		}
	}

	return true
}

func addInlinePolicies(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
	var (
		ok       bool
		props    map[string]interface{}
		policies []interface{}
	)

	if props, ok = resource["Properties"].(map[string]interface{}); !ok {
		props = make(map[string]interface{})
		resource["Properties"] = props
	}

	if _, exists := props["Path"]; !exists {
		props["Path"] = "/"
	}

	if _, exists := props["AssumeRolePolicyDocument"]; !exists {
		// This might seem like we have a policy attached to an EC2 instance like
		// you would attach a policy to a user. That's not the case. Instead this
		// defines a role with an inline policy that is implicitly assumed by an EC2
		// instance which is why the role must trust ec2.amazonaws.com
		props["AssumeRolePolicyDocument"] = map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": []string{"ec2.amazonaws.com"},
					},
					"Action": []string{
						"sts:AssumeRole",
					},
				},
			},
		}
	}

	if policies, ok = props["Policies"].([]interface{}); !ok {
		policies = make([]interface{}, 0)
	}

	i := 1
	for _, policyRaw := range policies {
		if policy, ok := policyRaw.(map[string]interface{}); ok {
			if name, exists := policy["PolicyName"]; exists {
				if name == "porter" {
					recv.log.Error("The name porter is reserved for inline policies")
					return false
				}
			} else {
				policy["PolicyName"] = fmt.Sprintf("service-defined-%d", i)
				i++
			}
		}
	}

	porterPolicy := map[string]interface{}{
		"PolicyName": "porter",
		"PolicyDocument": map[string]interface{}{
			"Statement": []interface{}{
				map[string]interface{}{
					"Sid":    "1",
					"Effect": "Allow",
					"Action": []string{
						// porterd
						"ec2:DescribeTags",
						"elasticloadbalancing:DescribeTags",
						"elasticloadbalancing:RegisterInstancesWithLoadBalancer",

						// decrypt .env-file
						"kms:Decrypt",
					},
					"Resource": "*",
				},
				map[string]interface{}{
					"Sid":    "2",
					"Effect": "Allow",
					"Action": []string{
						// porterd
						"cloudformation:DescribeStackResource",

						// secrets
						"cloudformation:DescribeStacks",
					},
					"Resource": map[string]string{"Ref": "AWS::StackId"},
				},
				map[string]interface{}{
					"Sid":    "3",
					"Effect": "Allow",
					"Action": []string{
						// pull down the service payload
						"s3:GetObject",
					},
					"Resource": fmt.Sprintf("arn:aws:s3:::%s/%s/*",
						recv.region.S3Bucket, recv.s3KeyRoot(s3KeyOptDeployment)),
				},
			},
		},
	}

	policies = append(policies, porterPolicy)
	props["Policies"] = policies

	return true
}
