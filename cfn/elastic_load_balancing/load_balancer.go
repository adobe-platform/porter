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
package elastic_load_balancing

import "github.com/adobe-platform/porter/cfn"

type (
	LoadBalancer struct {
		cfn.Resource

		Properties struct {
			AccessLoggingPolicy       *AccessLoggingPolicy         `json:"AccessLoggingPolicy,omitempty"`
			AppCookieStickinessPolicy []AppCookieStickinessPolicy  `json:"AppCookieStickinessPolicy,omitempty"`
			AvailabilityZones         interface{}                  `json:"AvailabilityZones,omitempty"`
			ConnectionDrainingPolicy  *ConnectionDrainingPolicy    `json:"ConnectionDrainingPolicy,omitempty"`
			ConnectionSettings        *ConnectionSettings          `json:"ConnectionSettings,omitempty"`
			CrossZone                 bool                         `json:"CrossZone,omitempty"`
			HealthCheck               *HealthCheck                 `json:"HealthCheck,omitempty"`
			Instances                 []string                     `json:"Instances,omitempty"`
			LBCookieStickinessPolicy  []LBCookieStickinessPolicy   `json:"LBCookieStickinessPolicy,omitempty"`
			LoadBalancerName          string                       `json:"LoadBalancerName,omitempty"`
			Listeners                 []Listener                   `json:"Listeners,omitempty"`
			Policies                  []ElasticLoadBalancingPolicy `json:"Policies,omitempty"`
			Scheme                    string                       `json:"Scheme,omitempty"`
			SecurityGroups            []interface{}                `json:"SecurityGroups,omitempty"`
			Subnets                   []string                     `json:"Subnets,omitempty"`
			Tags                      []cfn.Tag                    `json:"Tags,omitempty"`
		} `json:"Properties"`
	}

	AccessLoggingPolicy struct {
		EmitInterval   int    `json:"EmitInterval,omitempty"`
		Enabled        bool   `json:"Enabled,omitempty"`
		S3BucketName   string `json:"S3BucketName,omitempty"`
		S3BucketPrefix string `json:"S3BucketPrefix,omitempty"`
	}

	AppCookieStickinessPolicy struct {
		CookieName string `json:"CookieName,omitempty"`
		PolicyName string `json:"PolicyName,omitempty"`
	}

	ConnectionDrainingPolicy struct {
		Enabled bool `json:"Enabled,omitempty"`
		Timeout int  `json:"Timeout,omitempty"`
	}

	ConnectionSettings struct {
		IdleTimeout int `json:"IdleTimeout,omitempty"`
	}

	ElasticLoadBalancingPolicy struct {
		Attributes []struct {
			Name  string `json:"Name,omitempty"`
			Value string `json:"Value,omitempty"`
		} `json:"Attributes,omitempty"`
		InstancePorts     []string `json:"InstancePorts,omitempty"`
		LoadBalancerPorts []string `json:"LoadBalancerPorts,omitempty"`
		PolicyName        string   `json:"PolicyName,omitempty"`
		PolicyType        string   `json:"PolicyType,omitempty"`
	}

	HealthCheck struct {
		HealthyThreshold   string `json:"HealthyThreshold,omitempty"`
		Interval           string `json:"Interval,omitempty"`
		Target             string `json:"Target,omitempty"`
		Timeout            string `json:"Timeout,omitempty"`
		UnhealthyThreshold string `json:"UnhealthyThreshold,omitempty"`
	}

	LBCookieStickinessPolicy struct {
		CookieExpirationPeriod string `json:"CookieExpirationPeriod,omitempty"`
		PolicyName             string `json:"PolicyName,omitempty"`
	}

	Listener struct {
		InstancePort     string   `json:"InstancePort,omitempty"`
		InstanceProtocol string   `json:"InstanceProtocol,omitempty"`
		LoadBalancerPort string   `json:"LoadBalancerPort,omitempty"`
		PolicyNames      []string `json:"PolicyNames,omitempty"`
		Protocol         string   `json:"Protocol,omitempty"`
		SSLCertificateId string   `json:"SSLCertificateId,omitempty"`
	}
)

func NewLoadBalancer() LoadBalancer {
	return LoadBalancer{
		Resource: cfn.Resource{
			Type: "AWS::ElasticLoadBalancing::LoadBalancer",
		},
	}
}
