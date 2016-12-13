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
	"os/exec"
	"sync"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision_state"
	"github.com/aws/aws-sdk-go/aws"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/inconshreveable/log15"
)

type (
	CfnApiInput struct {
		Environment string
		Region      string
		SecretsKey  string
		SecretsLoc  string
		TemplateUrl string
	}
)

func CreateStack(log log15.Logger, config *conf.Config, stack *provision_state.Stack) bool {

	var err error

	defer exec.Command("rm", "-rf", constants.PayloadPath).Run()

	stack.Name, err = GetStackName(config.ServiceName, stack.Environment, true)
	if err != nil {
		log.Error("Failed to get stack name", "Error", err)
		return false
	}

	var fLock sync.RWMutex

	cfnAPI := func(client *cfnlib.CloudFormation, input CfnApiInput) (stackId string, success bool) {
		fLock.Lock()
		defer fLock.Unlock()

		parameters := []*cfnlib.Parameter{
			{
				ParameterKey:   aws.String(constants.ParameterStackName),
				ParameterValue: aws.String(stack.Name),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsKey),
				ParameterValue: aws.String(input.SecretsKey),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsLoc),
				ParameterValue: aws.String(input.SecretsLoc),
			},
		}

		stackId, err := cloudformation.CreateStack(client, stack.Name, input.TemplateUrl, parameters)
		if err != nil {
			log.Error("CreateStack API call failed", "Error", err)
			return
		}
		success = true
		return
	}

	return createUpdateStack(log, stack, config, false, cfnAPI)
}

func UpdateStack(log log15.Logger, config *conf.Config, stack provision_state.Stack) bool {

	var fLock sync.RWMutex

	log.Debug("UpdateStack", "stack.Name", stack.Name)

	cfnAPI := func(client *cfnlib.CloudFormation, input CfnApiInput) (stackId string, success bool) {
		fLock.Lock()
		defer fLock.Unlock()

		regionOutput, exists := stack.Regions[input.Region]
		if !exists {
			log.Error("Missing stack name for region", "region", input.Region)
			return
		}

		parameters := []*cfnlib.Parameter{
			{
				ParameterKey:   aws.String(constants.ParameterStackName),
				ParameterValue: aws.String(stack.Name),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsKey),
				ParameterValue: aws.String(input.SecretsKey),
			},
			{
				ParameterKey:   aws.String(constants.ParameterSecretsLoc),
				ParameterValue: aws.String(input.SecretsLoc),
			},
		}

		err := cloudformation.UpdateStack(client, regionOutput.StackId, input.TemplateUrl, parameters)
		if err != nil {
			log.Error("UpdateStack API call failed", "Error", err)
			return
		}

		stackId = regionOutput.StackId
		success = true
		return
	}

	return createUpdateStack(log, &stack, config, true, cfnAPI)
}

func createUpdateStack(
	log log15.Logger,
	stack *provision_state.Stack,
	config *conf.Config,
	updateStack bool,
	cfnAPI func(*cfnlib.CloudFormation, CfnApiInput) (string, bool)) (success bool) {

	environment, err := config.GetEnvironment(stack.Environment)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		return
	}

	if stack.Regions == nil {
		stack.Regions = make(map[string]*provision_state.Region)
	}

	successChan := make(chan bool)

	for _, region := range environment.Regions {

		roleARN, err := environment.GetRoleARN(region.Name)
		if err != nil {
			log.Error("GetRoleARN", "Error", err)
			return
		}

		roleSession := aws_session.STS(region.Name, roleARN, 1*time.Hour)

		recv := &stackCreator{
			log: log.New("Region", region.Name),

			config:      *config,
			environment: *environment,
			region:      *region,

			roleSession: roleSession,

			cfnAPI:      cfnAPI,
			updateStack: updateStack,

			templateTransforms: make(map[string][]MapResource),
		}

		var regionState *provision_state.Region
		var exists bool
		if regionState, exists = stack.Regions[region.Name]; !exists {
			regionState = &provision_state.Region{}
		}

		stack.Regions[region.Name] = regionState

		go func(recv *stackCreator, regionState *provision_state.Region) {

			successChan <- recv.createUpdateStackForRegion(regionState)

		}(recv, regionState)
	}

	success = true

	for i := 0; i < len(environment.Regions); i++ {
		regionSuccess := <-successChan
		success = success && regionSuccess
	}

	return
}
