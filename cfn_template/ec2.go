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
package cfn_template

import (
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/constants"
)

// Allow EC2 instances to receive traffic from the provisioned ELB
func ELBToInstance(vpc, httpsOnly bool, httpsPort int, elbName, elbSecurityGroup string) map[string]interface{} {
	properties := map[string]interface{}{
		"GroupDescription": "Enable communication from the provisioned ELB",
	}
	metadata := make(map[string]interface{})
	metadata[constants.MetadataAsLc] = true

	securityGroup := map[string]interface{}{
		"Type":       cfn.EC2_SecurityGroup,
		"Properties": properties,
		"Metadata":   metadata,
	}

	sgIngress := make([]interface{}, 0)

	if vpc {

		if !httpsOnly {
			ingress := map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   constants.HTTP_Port,
				"ToPort":     constants.HTTP_Port,
				"SourceSecurityGroupId": map[string]string{
					"Ref": elbSecurityGroup,
				},
			}
			sgIngress = append(sgIngress, ingress)
		}

		if httpsPort > 0 {
			ingress := map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   httpsPort,
				"ToPort":     httpsPort,
				"SourceSecurityGroupId": map[string]string{
					"Ref": elbSecurityGroup,
				},
			}
			sgIngress = append(sgIngress, ingress)
		}
	} else {

		if !httpsOnly {
			ingress := map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   constants.HTTP_Port,
				"ToPort":     constants.HTTP_Port,
				"SourceSecurityGroupOwnerId": map[string]interface{}{
					"Fn::GetAtt": []string{
						elbName,
						"SourceSecurityGroup.OwnerAlias",
					},
				},
				"SourceSecurityGroupName": map[string]interface{}{
					"Fn::GetAtt": []string{
						elbName,
						"SourceSecurityGroup.GroupName",
					},
				},
			}
			sgIngress = append(sgIngress, ingress)
		}

		if httpsPort > 0 {
			httpsIngress := map[string]interface{}{
				"IpProtocol": "tcp",
				"FromPort":   httpsPort,
				"ToPort":     httpsPort,
				"SourceSecurityGroupOwnerId": map[string]interface{}{
					"Fn::GetAtt": []string{
						elbName,
						"SourceSecurityGroup.OwnerAlias",
					},
				},
				"SourceSecurityGroupName": map[string]interface{}{
					"Fn::GetAtt": []string{
						elbName,
						"SourceSecurityGroup.GroupName",
					},
				},
			}
			sgIngress = append(sgIngress, httpsIngress)
		}
	}

	properties["SecurityGroupIngress"] = sgIngress

	return securityGroup
}

// Allow internet traffic to the ELB. This is only use in a custom VPC.
func InetToELB(vpc, https, httpsOnly bool) map[string]interface{} {

	properties := map[string]interface{}{
		"GroupDescription": "Allow internet traffic",
	}
	metadata := make(map[string]interface{})

	securityGroup := map[string]interface{}{
		"Type":       cfn.EC2_SecurityGroup,
		"Properties": properties,
		"Metadata":   metadata,
	}

	// Only associate this sg and create ingress rules in a custom VPC.
	// EC2-Classic and Default VPC don't need this
	if vpc {
		metadata[constants.MetadataElb] = true

		sgIngress := make([]interface{}, 0)

		if !httpsOnly {
			httpIngress := map[string]interface{}{
				"IpProtocol": "tcp",
				"CidrIp":     "0.0.0.0/0",
				"FromPort":   constants.HTTP_Port,
				"ToPort":     constants.HTTP_Port,
			}

			sgIngress = append(sgIngress, httpIngress)
		}

		if https {

			httpsIngress := map[string]interface{}{
				"IpProtocol": "tcp",
				"CidrIp":     "0.0.0.0/0",
				"FromPort":   constants.HTTPS_Port,
				"ToPort":     constants.HTTPS_Port,
			}

			sgIngress = append(sgIngress, httpsIngress)
		}

		properties["SecurityGroupIngress"] = sgIngress
	}

	return securityGroup
}
