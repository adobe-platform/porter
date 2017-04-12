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
package auto_scaling

import "github.com/adobe-platform/porter/cfn"

type (
	AutoScalingGroup struct {
		cfn.Resource

		Properties struct {
			AvailabilityZones          interface{}                 `json:"AvailabilityZones,omitempty"`
			Cooldown                   string                      `json:"Cooldown,omitempty"`
			DesiredCapacity            string                      `json:"DesiredCapacity,omitempty"`
			HealthCheckGracePeriod     int                         `json:"HealthCheckGracePeriod,omitempty"`
			HealthCheckType            string                      `json:"HealthCheckType,omitempty"`
			InstanceId                 string                      `json:"InstanceId,omitempty"`
			LaunchConfigurationName    interface{}                 `json:"LaunchConfigurationName,omitempty"`
			LoadBalancerNames          []string                    `json:"LoadBalancerNames,omitempty"`
			MaxSize                    string                      `json:"MaxSize,omitempty"`
			MetricsCollection          []MetricsCollection         `json:"MetricsCollection,omitempty"`
			MinSize                    string                      `json:"MinSize,omitempty"`
			NotificationConfigurations []NotificationConfiguration `json:"NotificationConfigurations,omitempty"`
			PlacementGroup             string                      `json:"PlacementGroup,omitempty"`
			Tags                       []AutoScalingTag            `json:"Tags,omitempty"`
			TerminationPolicies        []string                    `json:"TerminationPolicies,omitempty"`
			VPCZoneIdentifier          []string                    `json:"VPCZoneIdentifier,omitempty"`
		} `json:"Properties,omitempty"`
	}

	MetricsCollection struct {
		Granularity string   `json:"Granularity,omitempty"`
		Metrics     []string `json:"Metrics,omitempty"`
	}

	NotificationConfiguration struct {
		NotificationTypes []string `json:"NotificationTypes,omitempty"`
		TopicARN          string   `json:"TopicARN,omitempty"`
	}

	AutoScalingTag struct {
		Key               string `json:"Key,omitempty"`
		Value             string `json:"Value,omitempty"`
		PropagateAtLaunch bool   `json:"PropagateAtLaunch,omitempty"`
	}
)

func NewAutoScalingGroup() AutoScalingGroup {
	return AutoScalingGroup{
		Resource: cfn.Resource{
			Type: "AWS::AutoScaling::AutoScalingGroup",
		},
	}
}
