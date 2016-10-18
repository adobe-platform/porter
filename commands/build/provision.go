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
package build

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision"
	"github.com/adobe-platform/porter/provision_output"
	"github.com/adobe-platform/porter/util"
	"github.com/phylake/go-cli"
)

var sleepDuration = constants.StackCreationPollInterval()

type (
	ProvisionStackCmd struct{}
)

func (recv *ProvisionStackCmd) Name() string {
	return "provision"
}

func (recv *ProvisionStackCmd) ShortHelp() string {
	return "Provision a new stack"
}

func (recv *ProvisionStackCmd) LongHelp() string {
	return `NAME
    provision -- Provision a new stack

SYNOPSIS
    provision -e <environment out of .porter/config>

DESCRIPTION
    Provision a new stack for a given environment.

    This command is similar to create-stack but it works with multiple regions
    and should be run from a build box.`
}

func (recv *ProvisionStackCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *ProvisionStackCmd) Execute(args []string) bool {

	if len(args) > 0 {
		var environment string
		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&environment, "e", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		ProvisionStack(environment)
		return true
	}

	return false
}

func ProvisionStack(env string) {

	log := logger.CLI("cmd", "build-provision")

	provisionOutput := provision_output.Environment{
		Environment: env,
		Regions:     make([]provision_output.Region, 0),
	}

	config, success := conf.GetAlteredConfig(log)
	if !success {
		os.Exit(1)
	}

	environment, err := config.GetEnvironment(env)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		os.Exit(1)
	}

	err = environment.IsWithinBlackoutWindow()
	if err != nil {
		log.Error("Blackout window is active", "Error", err, "Environment", environment.Name)
		os.Exit(1)
	}

	_, err = os.Stat(constants.PayloadPath)
	if err != nil {
		log.Error("Service payload not found", "ServicePayloadPath", constants.PayloadPath, "Error", err)
		os.Exit(1)
	}

	stackArgs := provision.StackArgs{
		Environment: env,
	}

	commandSuccess := hook.Execute(log, constants.HookPreProvision, env, nil, true)

	if commandSuccess {
		stackOutput, success := provision.CreateStack(log, config, stackArgs)
		if !success {
			os.Exit(1)
		}

		regionCount := len(environment.Regions)
		outputChan := make(chan provision_output.Region, regionCount)
		failureChan := make(chan struct{}, regionCount)

		for _, regionOutput := range stackOutput.Regions {
			go provisionStackPoll(environment, regionOutput, outputChan, failureChan)
		}

		for i := 0; i < regionCount; i++ {
			select {
			case regionOutput := <-outputChan:
				provisionOutput.Regions = append(provisionOutput.Regions, regionOutput)
			case _ = <-failureChan:
				commandSuccess = false
			}
		}

		if commandSuccess {

			provisionBytes, err := json.Marshal(provisionOutput)
			if err != nil {
				log.Error("json.Marshal", "Error", err)
				os.Exit(1)
			}

			// write the stackoutput into porter tmp directory
			err = ioutil.WriteFile(constants.ProvisionOutputPath, provisionBytes, 0644)
			if err != nil {
				log.Error("Unable to write provision output", "Error", err)
				os.Exit(1)
			}

		} else {

			if len(provisionOutput.Regions) > 0 {
				log.Warn("Some regions failed to create. Deleting the successful ones")

				for _, pr := range provisionOutput.Regions {
					roleARN, err := environment.GetRoleARN(pr.AWSRegion)
					if err != nil {
						log.Error("GetRoleARN", "Error", err)
						continue
					}

					roleSession := aws_session.STS(pr.AWSRegion, roleARN, 0)
					cfnClient := cloudformation.New(roleSession)

					log.Info("DeleteStack", "StackId", pr.StackId)
					cloudformation.DeleteStack(cfnClient, pr.StackId)
				}
			}
		}
	}

	commandSuccess = hook.Execute(log, constants.HookPostProvision,
		env, provisionOutput.Regions, commandSuccess)

	if !commandSuccess {
		os.Exit(1)
	}
}

func provisionStackPoll(environment *conf.Environment, stackRegionOutput provision.CreateStackRegionOutput, outputChan chan provision_output.Region, failureChan chan struct{}) {
	var (
		stackProvisioned   bool
		elbLogicalId       string
		physicalResourceID string
	)

	log := logger.CLI("cmd", "build-provision", "Region", stackRegionOutput.Region)

	region, err := environment.GetRegion(stackRegionOutput.Region)
	if err != nil {
		log.Error("GetRegion", "Error", err)
		return
	}

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, constants.StackCreationTimeout())
	cfnClient := cloudformation.New(roleSession)

	n := int(constants.StackCreationTimeout().Seconds() / sleepDuration.Seconds())

stackEventPoll:
	for i := 0; i < n; i++ {

		stackStatus, err := cloudformation.DescribeStack(cfnClient, stackRegionOutput.StackId)
		if err != nil {
			log.Error("DescribeStack", "Error", err)
			failureChan <- struct{}{}
			return
		}
		if stackStatus == nil || len(stackStatus.Stacks) != 1 {
			log.Error("unexpected stack status")
			failureChan <- struct{}{}
			return
		}

		log.Info("Stack status", "StackStatus", *stackStatus.Stacks[0].StackStatus)

		switch *stackStatus.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			stackProvisioned = true
			break stackEventPoll
		case "CREATE_FAILED":
			log.Error("Stack creation failed")
			failureChan <- struct{}{}
			return
		case "DELETE_IN_PROGRESS":
			log.Error("Stack is being deleted")
			failureChan <- struct{}{}
			return
		case "ROLLBACK_IN_PROGRESS":
			log.Error("Stack is rolling back")
			failureChan <- struct{}{}
			return
		}

		time.Sleep(sleepDuration)
	}

	if !stackProvisioned {
		log.Error("stack provision timeout")
		failureChan <- struct{}{}
		return
	}

	cfnTemplateByte, err := ioutil.ReadFile(constants.CloudFormationTemplatePath)
	if err != nil {
		log.Error("CloudFormationTemplate read file error", "Error", err)
	}

	cfnTemplate := cfn.NewTemplate()

	err = json.Unmarshal(cfnTemplateByte, &cfnTemplate)
	if err != nil {
		log.Error("json unmarshal error on cfn template", "Error", err)
		failureChan <- struct{}{}
		return
	}

	cfnTemplate.ParseResources()

	if region.PrimaryTopology() == conf.Topology_Inet {

		elbLogicalId, err = cfnTemplate.GetResourceName(cfn.ElasticLoadBalancing_LoadBalancer)
		if err != nil {
			log.Error("GetResourceName", "Error", err)
			failureChan <- struct{}{}
			return
		}

		//Once stack provisioned get the provisoned elb
		retryMsg := func(i int) { log.Warn("DescribeStackResource retrying", "Count", i) }
		if !util.SuccessRetryer(9, retryMsg, func() bool {
			physicalResourceID, err = cloudformation.DescribeStackResource(cfnClient, stackRegionOutput.StackId, elbLogicalId)
			if err != nil {
				log.Error("Error on getting physicalResourceId", "Error", err)
				return false
			}
			return true
		}) {
			failureChan <- struct{}{}
			return
		}
	}

	provisionDetails := provision_output.Region{
		AWSRegion:          region.Name,
		StackId:            stackRegionOutput.StackId,
		ProvisionedELBName: physicalResourceID,
	}

	outputChan <- provisionDetails

}
