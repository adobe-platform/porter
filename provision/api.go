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
	"sync"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/aws/aws-sdk-go/aws"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/inconshreveable/log15"
)

type (
	StackArgs struct {
		Environment  string
		KeepCount    int
		PayloadS3Key string
	}

	CfnApiInput struct {
		Environment string
		Region      string
		SecretsKey  string
		TemplateUrl string
	}

	CreateStackOutput struct {
		StackName string
		Regions   map[string]CreateStackRegionOutput
	}

	CreateStackRegionOutput struct {
		StackId string
		Region  string
	}
)

func CreateStack(log log15.Logger, config *conf.Config, args StackArgs) (CreateStackOutput, bool) {

	stackName, err := GetStackName(config.ServiceName, args.Environment, true)
	if err != nil {
		log.Error("Failed to get stack name", "Error", err)
		return CreateStackOutput{}, false
	}

	var fLock sync.RWMutex

	cfnAPI := func(client *cfnlib.CloudFormation, input CfnApiInput) (stackId string, success bool) {
		fLock.Lock()
		defer fLock.Unlock()

		parameters := []*cfnlib.Parameter{
			{
				ParameterKey:   aws.String(constants.ParameterStackName),
				ParameterValue: aws.String(stackName),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsKey),
				ParameterValue: aws.String(input.SecretsKey),
			},
		}

		stackId, err := cloudformation.CreateStack(client, stackName, input.TemplateUrl, parameters)
		if err != nil {
			log.Error("CreateStack API call failed", "Error", err)
			return
		}
		success = true
		return
	}

	output, success := createUpdateStack(log, stackName, config, args, cfnAPI)

	return output, success
}

func UpdateStack(log log15.Logger, config *conf.Config, args StackArgs, createStackOutput CreateStackOutput) bool {

	var fLock sync.RWMutex

	cfnAPI := func(client *cfnlib.CloudFormation, input CfnApiInput) (stackId string, success bool) {
		fLock.Lock()
		defer fLock.Unlock()

		regionOutput, exists := createStackOutput.Regions[input.Region]
		if !exists {
			log.Error("Missing stack name for region", "region", input.Region)
			return
		}

		parameters := []*cfnlib.Parameter{
			{
				ParameterKey:   aws.String(constants.ParameterStackName),
				ParameterValue: aws.String(createStackOutput.StackName),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsKey),
				ParameterValue: aws.String(input.SecretsKey),
			},
		}

		err := cloudformation.UpdateStack(client, regionOutput.StackId, input.TemplateUrl, parameters)
		if err != nil {
			log.Error("UpdateStack API call failed", "Error", err)
			return
		}

		success = true
		return
	}

	_, success := createUpdateStack(log, createStackOutput.StackName, config, args, cfnAPI)
	return success
}

func createUpdateStack(
	log log15.Logger,
	stackName string,
	config *conf.Config,
	args StackArgs,
	cfnAPI func(*cfnlib.CloudFormation, CfnApiInput) (string, bool)) (stackOutput CreateStackOutput, success bool) {

	environment, err := config.GetEnvironment(args.Environment)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		return
	}

	regionCount := len(environment.Regions)

	regionOutputs := make(map[string]CreateStackRegionOutput)
	outChan := make(chan CreateStackRegionOutput, regionCount)
	errChan := make(chan struct{}, regionCount)

	for _, region := range environment.Regions {

		roleARN, err := environment.GetRoleARN(region.Name)
		if err != nil {
			log.Error("GetRoleARN", "Error", err)
			return
		}

		roleSession := aws_session.STS(region.Name, roleARN, 1*time.Hour)

		recv := &stackCreator{
			log:  log.New("Region", region.Name),
			args: args,

			stackName: stackName,

			config:      *config,
			environment: *environment,
			region:      *region,

			roleSession: roleSession,

			cfnAPI: cfnAPI,

			templateTransforms: make(map[string][]MapResource),
		}

		go recv.createUpdateStackForRegion(outChan, errChan)
	}

	for i := 0; i < regionCount; i++ {
		select {
		case regionOutput := <-outChan:
			regionOutputs[regionOutput.Region] = regionOutput
		case _ = <-errChan:
			return
		}
	}

	stackOutput = CreateStackOutput{
		StackName: stackName,
		Regions:   regionOutputs,
	}

	success = true
	return
}
