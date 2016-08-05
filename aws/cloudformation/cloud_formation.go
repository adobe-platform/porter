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
package cloudformation

import (
	"errors"
	"fmt"

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
	input := &cfnlib.CreateStackInput{
		StackName: aws.String(stackName),
		Capabilities: []*string{
			aws.String("CAPABILITY_IAM"), // Required
		},
		OnFailure:        aws.String("ROLLBACK"),
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

// DescribeStackResource using AWS http://docs.aws.amazon.com/sdk-for-go/api/service/cloudformation/CloudFormation.html#DescribeStackResource-instance_method
func DescribeStackResource(client *cfnlib.CloudFormation, stackName string, logicalID string) (string, error) {
	params := &cfnlib.DescribeStackResourceInput{
		LogicalResourceId: aws.String(logicalID), // Required
		StackName:         aws.String(stackName), // Required
	}

	resp, err := client.DescribeStackResource(params)
	if err != nil {
		return "", err
	}

	if resp == nil || resp.StackResourceDetail == nil {
		return "", errors.New("DescribeStackResources is empty")
	}

	return *resp.StackResourceDetail.PhysicalResourceId, nil
}

// DescribeStack using http://docs.aws.amazon.com/sdk-for-go/api/service/cloudformation/CloudFormation.html#DescribeStacks-instance_method
func DescribeStack(client *cfnlib.CloudFormation, stackName ...string) (*cfnlib.DescribeStacksOutput, error) {
	param := &cfnlib.DescribeStacksInput{}
	if len(stackName) > 1 {
		return nil, fmt.Errorf("StackName should be size of one, current size=%d", len(stackName))
	}
	if len(stackName) == 1 {
		param.StackName = aws.String(stackName[0])
	}
	return client.DescribeStacks(param)
}
