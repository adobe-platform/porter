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
package util

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"gopkg.in/inconshreveable/log15.v2"
)

func GetStacks(log log15.Logger, config *conf.Config, environment *conf.Environment,
	cfnClient *cloudformation.CloudFormation, stackId *string, validState func(string) bool) (stackList []*cloudformation.Stack, success bool) {

	var stackName string
	var err error

	if config != nil && environment != nil {
		stackName, err = GetStackName(config.ServiceName, environment.Name, false)
		if err != nil {
			log.Error("GetStackName", "Error", err)
			return
		}
	}

	stackList = make([]*cloudformation.Stack, 0)

	var nextToken *string

	for {

		describeStacksInput := &cloudformation.DescribeStacksInput{
			NextToken: nextToken,
			StackName: stackId,
		}
		var describeStacksOutput *cloudformation.DescribeStacksOutput

		retryMsg := func(i int) { log.Warn("cloudformation:DescribeStacks retrying", "Count", i) }
		if !util.SuccessRetryer(4, retryMsg, func() bool {

			log.Info("cloudformation:DescribeStacks")
			describeStacksOutput, err = cfnClient.DescribeStacks(describeStacksInput)
			if err != nil {
				log.Error("cloudformation:DescribeStacks", "Error", err)
				return false
			}

			return true
		}) {
			log.Error("Failed to cloudformation:DescribeStacks")
			return
		}

		for _, stack := range describeStacksOutput.Stacks {

			if stack != nil &&
				(stackName == "" || strings.HasPrefix(*stack.StackName, stackName)) &&
				validState(*stack.StackStatus) {

				stackList = append(stackList, stack)
				log.Info("Found stack", "StackName", *stack.StackName)
			}
		}

		nextToken = describeStacksOutput.NextToken
		if nextToken == nil {
			break
		} else {
			log.Debug("DescribeStacks", "NextToken", *nextToken)
		}
	}

	sort.Sort(sort.Reverse(ByDate(stackList)))

	success = true
	return
}

func GetStackName(serviceName, environment string, addTimestamp bool) (stackName string, err error) {
	if serviceName == "" {
		err = errors.New("service name is empty")
		return
	}

	whoAmIBytes, err := exec.Command("whoami").Output()
	if err != nil {
		return
	}
	whoAmI := strings.TrimSpace(string(whoAmIBytes))

	if addTimestamp {
		epoch := time.Now().Unix()

		stackName = fmt.Sprintf("%s-%s-%s-%d", serviceName, environment, whoAmI, epoch)
	} else {

		stackName = fmt.Sprintf("%s-%s-%s", serviceName, environment, whoAmI)
	}

	return
}
