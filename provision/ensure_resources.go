/*
 * (c) 2016-2018 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package provision

import (
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/cfn_template"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
)

func (recv *stackCreator) ensureResources(template *cfn.Template) (success bool) {
	if !recv.ensureParameters(template) {
		return
	}

	if !recv.ensureMappings(template) {
		return
	}

	if !recv.ensureSignalQueue(template) {
		return
	}

	if !recv.ensureIAMRole(template) {
		return
	}

	if !recv.ensureIAMInstanceProfile(template) {
		return
	}

	if !recv.ensureAutoScalingLaunchConfig(template) {
		return
	}

	if !recv.ensureAutoScalingGroup(template) {
		return
	}

	// overloaded to include metadata which is why it applies
	// to all topologies
	if !recv.ensureWaitConditionHandle(template) {
		return
	}

	if !recv.ensureWaitCondition(template) {
		return
	}

	switch recv.region.PrimaryTopology() {
	case conf.Topology_Inet:
		if recv.region.HasELB() {

			if !recv.ensureELB(template) {
				return
			}

			if recv.environment.CreateSecurityGroups == nil || *recv.environment.CreateSecurityGroups == true {

				if !recv.ensureDestinationELBSecurityGroup(template) {
					return
				}

				if !recv.ensureInetSG(template) {
					return
				}

				if !recv.ensureProvisionedELBToInstanceSG(template) {
					return
				}
			}

			if !recv.ensureDNSResources(template) {
				return
			}
		} else {
			if recv.environment.CreateSecurityGroups == nil || *recv.environment.CreateSecurityGroups == true {

				if !recv.ensureInetSG(template) {
					return
				}
			}
		}

	}

	success = true
	return
}

func (recv *stackCreator) ensureParameters(template *cfn.Template) bool {
	if template.Parameters == nil {
		template.Parameters = make(map[string]cfn.ParameterInput)
	}

	template.Parameters[constants.ParameterEnvironment] = cfn.ParameterInput{
		Description:   "Porter environment",
		Type:          "String",
		AllowedValues: []string{recv.environment.Name},
		Default:       recv.environment.Name,
	}

	template.Parameters[constants.ParameterServiceName] = cfn.ParameterInput{
		Description:   "Porter service name",
		Type:          "String",
		AllowedValues: []string{recv.config.ServiceName},
		Default:       recv.config.ServiceName,
	}

	template.Parameters[constants.ParameterStackName] = cfn.ParameterInput{
		Description: "Porter stack name",
		Type:        "String",
	}

	template.Parameters[constants.ParameterSecretsKey] = cfn.ParameterInput{
		Description: "Symmetric key for secrets",
		Type:        "String",
	}

	template.Parameters[constants.ParameterSecretsLoc] = cfn.ParameterInput{
		Description: "Secrets payload location",
		Type:        "String",
	}

	return true
}

func (recv *stackCreator) ensureMappings(template *cfn.Template) bool {
	if template.Mappings == nil {
		template.Mappings = make(map[string]interface{})
	}

	template.Mappings[constants.MappingRegionToAMI] = cfn_template.RegionToAmazonLinuxAMI()

	return true
}

func (recv *stackCreator) ensureSignalQueue(template *cfn.Template) bool {
	resource := map[string]interface{}{
		"Type": cfn.SQS_Queue,
		"Properties": map[string]interface{}{
			"MaximumMessageSize":     1024,
			"MessageRetentionPeriod": 120,
		},
	}

	template.SetResource(constants.SignalQueue, resource)

	return true
}

func (recv *stackCreator) ensureIAMRole(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.IAM_Role); exists {
		return true
	}

	resource := map[string]interface{}{
		"Type": cfn.IAM_Role,
	}
	template.SetResource("IAMRole", resource)

	return true
}

func (recv *stackCreator) ensureIAMInstanceProfile(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.IAM_InstanceProfile); exists {
		return true
	}

	resource := map[string]interface{}{
		"Type": cfn.IAM_InstanceProfile,
	}
	template.SetResource("IAMInstanceProfile", resource)

	return true
}

func (recv *stackCreator) ensureELB(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.ElasticLoadBalancing_LoadBalancer); exists {
		return true
	}

	resource := map[string]interface{}{
		"Type": cfn.ElasticLoadBalancing_LoadBalancer,
	}
	template.SetResource("ApplicationLoadBalancer", resource)

	return true
}

func (recv *stackCreator) ensureAutoScalingLaunchConfig(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.AutoScaling_LaunchConfiguration); exists {
		return true
	}

	resource := map[string]interface{}{
		"Type": cfn.AutoScaling_LaunchConfiguration,
	}
	template.SetResource("AutoScalingLaunchConfiguration", resource)

	return true
}

func (recv *stackCreator) ensureAutoScalingGroup(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.AutoScaling_AutoScalingGroup); exists {
		return true
	}

	resource := map[string]interface{}{
		"Type": cfn.AutoScaling_AutoScalingGroup,
	}
	template.SetResource("AutoScalingGroup", resource)

	return true
}

func (recv *stackCreator) ensureWaitCondition(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.CloudFormation_WaitCondition); exists {
		return true
	}

	waitCondition := map[string]interface{}{
		"Type": cfn.CloudFormation_WaitCondition,
	}
	template.SetResource("WaitCondition", waitCondition)

	return true
}

func (recv *stackCreator) ensureWaitConditionHandle(template *cfn.Template) bool {
	if exists := template.ResourceExists(cfn.CloudFormation_WaitConditionHandle); exists {
		return true
	}

	waitConditionHandle := map[string]interface{}{
		"Type": cfn.CloudFormation_WaitConditionHandle,
	}
	template.SetResource("WaitConditionHandle", waitConditionHandle)

	return true
}

func (recv *stackCreator) ensureDNSResources(template *cfn.Template) bool {
	if recv.region.HostedZoneName == "" {
		return true
	}

	elbLogicalId, err := template.GetResourceName(cfn.ElasticLoadBalancing_LoadBalancer)
	if err != nil {
		recv.log.Warn("GetResourceName. DNS Alias won't be created", "Error", err)
		return true
	}

	recordSet := map[string]interface{}{
		"Type": cfn.Route53_RecordSet,
		"Properties": map[string]interface{}{
			"Type":           "A",
			"HostedZoneName": recv.region.HostedZoneName,
			"Name": map[string]interface{}{
				"Fn::Join": []interface{}{
					"",
					[]interface{}{
						map[string]interface{}{"Ref": constants.ParameterStackName},
						".", recv.region.HostedZoneName,
					},
				},
			},
			"AliasTarget": map[string]interface{}{
				"DNSName": map[string]interface{}{
					"Fn::GetAtt": []string{elbLogicalId, "DNSName"},
				},
				"HostedZoneId": map[string]interface{}{
					"Fn::GetAtt": []string{elbLogicalId, "CanonicalHostedZoneNameID"},
				},
			},
		},
	}

	template.SetResource("ELBARecordAlias", recordSet)

	return true
}

func (recv *stackCreator) ensureInetSG(template *cfn.Template) bool {

	vpc := recv.region.VpcId != ""
	https := recv.region.SSLCertARN != ""
	httpsOnly := recv.environment.HAProxy.SSL.HTTPS_Only
	var metadataKey, resourceName string
	if recv.region.HasELB() {
		metadataKey = constants.MetadataElb
		resourceName = constants.ElbSgLogicalName
	} else {
		metadataKey = constants.MetadataAsLc
		resourceName = constants.AsgSgLogicalName
	}
	resource := cfn_template.InetSg(vpc, https, httpsOnly, metadataKey)

	template.SetResource(resourceName, resource)

	return true
}

func (recv *stackCreator) ensureProvisionedELBToInstanceSG(template *cfn.Template) bool {

	elbLogicalId, err := template.GetResourceName(cfn.ElasticLoadBalancing_LoadBalancer)
	if err != nil {
		recv.log.Warn("GetResourceName. Couldn't find ELB logical name", "Error", err)
		return false
	}

	httpsPort := 0
	if recv.region.SSLCertARN != "" {
		if recv.environment.HAProxy.UsingSSL() {

			httpsPort = constants.HTTPS_Port
		} else {

			httpsPort = constants.HTTPS_TermPort
		}
	}
	vpc := recv.region.VpcId != ""
	httpsOnly := recv.environment.HAProxy.SSL.HTTPS_Only
	resource := cfn_template.ELBToInstance(vpc, httpsOnly, httpsPort,
		elbLogicalId, constants.ElbSgLogicalName)

	template.SetResource("ProvisionedELBToInstance", resource)

	return true
}

func (recv *stackCreator) ensureDestinationELBSecurityGroup(template *cfn.Template) (success bool) {

	sgELBProperties := make(map[string]interface{})
	sgELBMetadata := make(map[string]interface{})
	sgELBMetadata[constants.MetadataAsLc] = true
	sgELB := map[string]interface{}{
		"Type":       cfn.EC2_SecurityGroup,
		"Properties": sgELBProperties,
		"Metadata":   sgELBMetadata,
	}
	sgELBProperties["GroupDescription"] = "Ingress for destination ELBs"

	client := elb.New(recv.roleSession)

	ingressRules := make([]interface{}, 0)

	for _, elbConfig := range recv.region.ELBs {

		log := recv.log.New("ELBName", elbConfig.Name)

		input := &elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(elbConfig.Name)},
		}
		output, err := client.DescribeLoadBalancers(input)
		if err != nil {
			log.Error("DescribeLoadBalancers", "Error", err)
			return
		}

		if len(output.LoadBalancerDescriptions) != 1 {
			log.Error("DescribeLoadBalancers len")
			return
		}

		elbDescription := output.LoadBalancerDescriptions[0]

		if elbDescription == nil {
			log.Error("elbDescription == nil")
			return
		}

		if elbDescription.SourceSecurityGroup == nil {
			log.Error("elbDescription.SourceSecurityGroup == nil")
			return
		}

		if recv.region.VpcId == "" {

			if elbDescription.SourceSecurityGroup.GroupName == nil ||
				elbDescription.SourceSecurityGroup.OwnerAlias == nil {
				log.Error("A SourceSecurityGroup field is nil")
				return
			}

			if *elbDescription.SourceSecurityGroup.GroupName == "" ||
				*elbDescription.SourceSecurityGroup.OwnerAlias == "" {
				log.Error("A SourceSecurityGroup field is empty")
				return
			}
		} else {

			if len(elbDescription.SecurityGroups) == 0 {
				log.Error("SecurityGroups field is nil")
				return
			}
		}

		ports := make([]int, 0)
		if recv.environment.HAProxy.UsingSSL() {

			ports = append(ports, constants.HTTPS_Port)
		} else {

			ports = append(ports, constants.HTTPS_TermPort)
		}

		if !recv.environment.HAProxy.SSL.HTTPS_Only {
			ports = append(ports, constants.HTTP_Port)
		}

		for _, port := range ports {

			ingressRule := map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   int(port),
				"ToPort":     int(port),
			}

			if recv.region.VpcId == "" {

				ingressRule["SourceSecurityGroupOwnerId"] = *elbDescription.SourceSecurityGroup.OwnerAlias
				ingressRule["SourceSecurityGroupName"] = *elbDescription.SourceSecurityGroup.GroupName
			} else {

				// We only need one security group from the elb to enable
				// instances to receive traffic from it. The rules of the
				// security group itself are meaningless
				ingressRule["SourceSecurityGroupId"] = *elbDescription.SecurityGroups[0]
			}

			ingressRules = append(ingressRules, ingressRule)
		}
	}

	sgELBProperties["SecurityGroupIngress"] = ingressRules

	template.SetResource(constants.DstELBSecurityGroup, sgELB)

	success = true
	return
}
