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
package cloudformation

import (
	"os"

	"github.com/adobe-platform/porter/constants"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
)

// Don't force clients of this package to import
// "github.com/aws/aws-sdk-go/service/cloudformation"
func New(config *session.Session) *cfnlib.CloudFormation {
	return cfnlib.New(config)
}

// CreateStack using AWS http://docs.aws.amazon.com/sdk-for-go/api/service/cloudformation/CloudFormation.html#CreateStack-instance_method
func CreateStack(client *cfnlib.CloudFormation, stackName string, cfnTemplateUrl string, parameters []*cfnlib.Parameter) (string, error) {

	onFailure := os.Getenv(constants.EnvStackCreationOnFailure)
	switch onFailure {
	case "ROLLBACK", "DO_NOTHING":
	default:
		onFailure = "DELETE"
	}

	input := &cfnlib.CreateStackInput{
		StackName: aws.String(stackName),
		Capabilities: []*string{
			aws.String("CAPABILITY_IAM"), // Required
		},
		OnFailure:        aws.String(onFailure),
		Parameters:       parameters,
		TemplateURL:      aws.String(cfnTemplateUrl),
		TimeoutInMinutes: aws.Int64(int64(constants.StackCreationTimeout().Minutes())),
	}

	output, err := client.CreateStack(input)
	if err != nil {
		return "", err
	}

	return *output.StackId, nil
}

func DeleteStack(client *cfnlib.CloudFormation, stackName string) error {
	input := &cfnlib.DeleteStackInput{
		StackName: aws.String(stackName),
	}

	_, err := client.DeleteStack(input)
	return err
}

func UpdateStack(client *cfnlib.CloudFormation, stackName string, cfnTemplateUrl string, parameters []*cfnlib.Parameter) error {
	input := &cfnlib.UpdateStackInput{
		StackName:    aws.String(stackName),
		TemplateURL:  aws.String(cfnTemplateUrl),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters:   parameters,
	}

	_, err := client.UpdateStack(input)
	return err
}
