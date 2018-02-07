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
package auto_scaling

import "github.com/adobe-platform/porter/cfn"

type (
	LaunchConfiguration struct {
		cfn.Resource

		Properties struct {
			AssociatePublicIpAddress bool `json:"AssociatePublicIpAddress,omitempty"`
			BlockDeviceMappings      []struct {
				DeviceName string `json:"DeviceName,omitempty"`
				Ebs        struct {
					DeleteOnTermination bool   `json:"DeleteOnTermination,omitempty"`
					Iops                int    `json:"Iops,omitempty"`
					SnapshotId          string `json:"SnapshotId,omitempty"`
					VolumeSize          int    `json:"VolumeSize,omitempty"`
					VolumeType          string `json:"VolumeType,omitempty"`
				} `json:"Ebs,omitempty"`
				NoDevice    bool   `json:"NoDevice,omitempty"`
				VirtualName string `json:"VirtualName,omitempty"`
			} `json:"BlockDeviceMappings,omitempty"`
			ClassicLinkVPCId             string        `json:"ClassicLinkVPCId,omitempty"`
			ClassicLinkVPCSecurityGroups []string      `json:"ClassicLinkVPCSecurityGroups,omitempty"`
			EbsOptimized                 bool          `json:"EbsOptimized,omitempty"`
			IamInstanceProfile           string        `json:"IamInstanceProfile,omitempty"`
			ImageId                      string        `json:"ImageId,omitempty"`
			InstanceId                   string        `json:"InstanceId,omitempty"`
			InstanceMonitoring           bool          `json:"InstanceMonitoring,omitempty"`
			InstanceType                 string        `json:"InstanceType,omitempty"`
			KernelId                     string        `json:"KernelId,omitempty"`
			KeyName                      string        `json:"KeyName,omitempty"`
			PlacementTenancy             string        `json:"PlacementTenancy,omitempty"`
			RamDiskId                    string        `json:"RamDiskId,omitempty"`
			SecurityGroups               []interface{} `json:"SecurityGroups,omitempty"`
			SpotPrice                    string        `json:"SpotPrice,omitempty"`
			UserData                     string        `json:"UserData,omitempty"`
		} `json:"Properties,omitempty"`
	}
)

func NewLaunchConfiguration() LaunchConfiguration {
	return LaunchConfiguration{
		Resource: cfn.Resource{
			Type: "AWS::AutoScaling::LaunchConfiguration",
		},
	}
}
