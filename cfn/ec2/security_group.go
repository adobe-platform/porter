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
package ec2

import "github.com/adobe-platform/porter/cfn"

type (
	SecurityGroup struct {
		cfn.Resource

		Properties struct {
			GroupDescription     string                 `json:"GroupDescription,omitempty"`
			SecurityGroupEgress  []SecurityGroupEgress  `json:"SecurityGroupEgress,omitempty"`
			SecurityGroupIngress []SecurityGroupIngress `json:"SecurityGroupIngress,omitempty"`
			Tags                 []cfn.Tag              `json:"Tags,omitempty"`
			VpcId                string                 `json:"VpcId,omitempty"`
		} `json:"Properties,omitempty"`
	}

	SecurityGroupIngress struct {
		CidrIp                     string      `json:"CidrIp,omitempty"`
		FromPort                   int         `json:"FromPort,omitempty"`
		IpProtocol                 string      `json:"IpProtocol,omitempty"`
		SourceSecurityGroupId      interface{} `json:"SourceSecurityGroupId,omitempty"`
		SourceSecurityGroupName    interface{} `json:"SourceSecurityGroupName,omitempty"`
		SourceSecurityGroupOwnerId interface{} `json:"SourceSecurityGroupOwnerId,omitempty"`
		ToPort                     int         `json:"ToPort,omitempty"`
	}

	SecurityGroupEgress struct {
		CidrIp                     string `json:"CidrIp,omitempty"`
		FromPort                   int    `json:"FromPort,omitempty"`
		IpProtocol                 string `json:"IpProtocol,omitempty"`
		DestinationSecurityGroupId string `json:"DestinationSecurityGroupId,omitempty"`
		ToPort                     int    `json:"ToPort,omitempty"`
	}
)

func NewSecurityGroup() SecurityGroup {
	return SecurityGroup{
		Resource: cfn.Resource{
			Type: "AWS::EC2::SecurityGroup",
		},
	}
}
