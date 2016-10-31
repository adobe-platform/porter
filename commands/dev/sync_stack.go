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
package dev

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision"
	"github.com/adobe-platform/porter/provision_state"
	"github.com/inconshreveable/log15"
	"github.com/phylake/go-cli"
)

// Hot swap is shared by dev stacks and build boxes
type SyncStackCmd struct{}

func (recv *SyncStackCmd) Name() string {
	return "sync-stack"
}

func (recv *SyncStackCmd) ShortHelp() string {
	return "Reload local code into your developer stack"
}

func (recv *SyncStackCmd) LongHelp() string {
	return `NAME
    sync-stack -- Sync a created developer stack

SYNOPSIS
    sync-stack -e <environment out of .porter/config>

DESCRIPTION
    Sync a developer stack created with create-stack.`
}

func (recv *SyncStackCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *SyncStackCmd) Execute(args []string) bool {
	if len(args) > 0 {

		var environment string

		set := flag.NewFlagSet("", flag.ExitOnError)
		set.StringVar(&environment, "e", "", "") // don't use flag description
		set.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		set.Parse(args)

		UpdateStack(environment)
		return true
	}

	return false
}

func UpdateStack(environmentStr string) {

	log := logger.CLI()

	config, success := conf.GetConfig(log, true)
	if !success {
		os.Exit(1)
	}

	environment, err := config.GetEnvironment(environmentStr)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		os.Exit(1)
	}

	outputFile, err := os.Open(constants.CreateStackOutputPath)
	if err != nil {
		log.Error("os.Open", "Error", err)
		os.Exit(1)
	}

	stack := provision_state.Stack{}
	err = json.NewDecoder(outputFile).Decode(&stack)
	if err != nil {
		log.Error("json.Decode", "Error", err)
		os.Exit(1)
	}

	if len(stack.Regions) != 1 {
		msg := fmt.Sprintf("sync-stack works with a single region. found %d",
			len(stack.Regions))
		log.Error(msg)
		os.Exit(1)
	}

	var regionState *provision_state.Region
	for _, output := range stack.Regions {
		regionState = output
	}

	if success := provision.Package(log, config); !success {
		os.Exit(1)
	}

	if success := provision.UpdateStack(log, config, stack); !success {
		log.Error("Update stack failed")
		os.Exit(1)
	}

	log.Info("Called UpdateStack. Waiting for UPDATE_COMPLETE")
	pollUpdateComplete(log, environment, regionState.StackId)
}

func pollUpdateComplete(log log15.Logger, environment *conf.Environment, stackId string) {

	// the upper bound we think stack updation will take
	tokenLeaseDuration := 10 * time.Minute
	sleepDuration := 4 * time.Second

	region := environment.Regions[0]

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, tokenLeaseDuration)
	cfnClient := cloudformation.New(roleSession)

	stackEventState := cloudformation.NewStackEventState(cfnClient, stackId)

	// don't continue polling longer than our token's lease duration
	ticks := int(tokenLeaseDuration.Seconds() / sleepDuration.Seconds())
	for i := 0; i < ticks; i++ {

		stackEvents, err := stackEventState.DescribeStackEvents()
		if err != nil {
			log.Error("DescribeStackEvents", "Error", err)
			os.Exit(1)
		}

		for _, stackEvent := range stackEvents {
			if stackEvent.ResourceType != nil && stackEvent.ResourceStatus != nil {

				switch *stackEvent.ResourceStatus {
				case cfn.UPDATE_COMPLETE:
					switch *stackEvent.ResourceType {
					case "AWS::CloudFormation::Stack":
						log.Info("Received AWS::CloudFormation::Stack UPDATE_COMPLETE")
						log.Debug("NOTE: the upper bound to complete the hot swap from this point is")
						log.Debug(fmt.Sprintf("%d minute for the hot swap code to be called", constants.CfnHupPollIntervalMinutes))
						log.Debug("+ time to start the service")
						log.Debug("+ time to receive a 200 response on the health check")
						log.Debug("+ time to swap the old and new containers")
						return
					}
				}
			}
		}

		time.Sleep(sleepDuration)
	}

	log.Error("Never received AWS::CloudFormation::Stack UPDATE_COMPLETE")
	os.Exit(1)
}
