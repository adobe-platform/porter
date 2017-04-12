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
package iam

import "github.com/adobe-platform/porter/cfn"

// TODO define PolicyDocument struct per
// http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html
// and replace Role.Properities.AssumeRolePolicyDocument and Policy.PolicyDocument

type (
	Role struct {
		cfn.Resource

		Properties struct {
			// AssumeRolePolicyDocument is arbitrary JSON
			AssumeRolePolicyDocument map[string]interface{} `json:"AssumeRolePolicyDocument,omitempty"`
			ManagedPolicyArns        []string               `json:"ManagedPolicyArns,omitempty"`
			Path                     string                 `json:"Path,omitempty"`
			Policies                 []Policy               `json:"Policies,omitempty"`
		} `json:"Properties"`
	}

	Policy struct {
		// PolicyDocument is arbitrary JSON
		PolicyDocument map[string]interface{} `json:"PolicyDocument,omitempty"`
		PolicyName     string                 `json:"PolicyName,omitempty"`
	}
)

func NewRole() Role {
	return Role{
		Resource: cfn.Resource{
			Type: "AWS::IAM::Role",
		},
	}
}
