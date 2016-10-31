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
package prune

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws/ec2"
	"github.com/adobe-platform/porter/aws/elb"
	awsutil "github.com/adobe-platform/porter/aws/util"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision"
	"github.com/aws/aws-sdk-go/aws/session"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/inconshreveable/log15"
)

// exponential backoff
const mixedStackRetryCount = 6

func Do(log log15.Logger, config *conf.Config, environment *conf.Environment,
	keepCount int, elbFilter bool, elbTag string) (success bool) {

	stackName, err := provision.GetStackName(config.ServiceName, environment.Name, false)
	if err != nil {
		log.Error("provision.GetStackName", "Error", err)
		return
	}

	err = environment.IsWithinBlackoutWindow()
	if err != nil {
		log.Error("Blackout window is active", "Error", err, "Environment", environment.Name)
		return
	}

	regionCount := len(environment.Regions)
	pruneStackChan := make(chan bool, regionCount)

	for _, region := range environment.Regions {
		switch region.PrimaryTopology() {
		case conf.Topology_Inet:
			go pruneStacks(log, region, environment,
				stackName, keepCount, pruneStackChan, elbFilter, elbTag)
		case conf.Topology_Worker:
			go pruneStacks(log, region, environment,
				stackName, keepCount+1, pruneStackChan, false, elbTag)
		default:
			log.Error("Unsupported topology")
			return
		}
	}

	// wait for all prune commands to complete regardless of success
	commandFailed := false
	for i := 0; i < regionCount; i++ {
		pruneSuccess := <-pruneStackChan
		if !pruneSuccess {
			commandFailed = true
		}
	}

	if commandFailed {
		return
	}

	success = true
	return
}

func pruneStacks(log log15.Logger, region *conf.Region,
	environment *conf.Environment, stackName string, keepCount int,
	pruneStackChan chan bool, elbFilter bool, elbTag string) {

	log = log.New("Region", region.Name)

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, constants.StackCreationTimeout())
	cfnClient := cloudformation.New(roleSession)

	//filter stacks based on StackName for a given service
	stackList := make([]*cfnlib.Stack, 0)
	var nextToken *string

	for {

		describeStackInput := &cfnlib.DescribeStacksInput{
			NextToken: nextToken,
		}

		//get all stacks
		describeStackOutput, err := cfnClient.DescribeStacks(describeStackInput)
		if err != nil {
			log.Error("DescribeStack", "Error", err)
			pruneStackChan <- false
			return
		}

		if describeStackOutput == nil {
			log.Error("DescribeStack response is nil")
			pruneStackChan <- false
			return
		}

		for _, stack := range describeStackOutput.Stacks {
			if stack != nil && strings.HasPrefix(*stack.StackName, stackName) {
				switch *stack.StackStatus {
				case "CREATE_COMPLETE",
					"UPDATE_COMPLETE",
					"ROLLBACK_COMPLETE":
					stackList = append(stackList, stack)
					log.Info("Found stack", "StackName", *stack.StackName)
				}
			}
		}

		nextToken = describeStackOutput.NextToken
		if nextToken == nil {
			break
		} else {
			log.Debug("DescribeStacks", "NextToken", *nextToken)
		}
	}

	var pruneList []*cfnlib.Stack

	if elbFilter {

		var getListSuccess bool
		pruneList, getListSuccess = getELBPruneList(log, environment,
			region, stackList, roleSession, elbTag)

		if !getListSuccess {
			pruneStackChan <- false
			return
		}
	} else {

		pruneList = stackList
	}

	//sort the stacks by CreationTime
	sort.Sort(sort.Reverse(awsutil.ByDate(pruneList)))

	for i, stack := range pruneList {
		if i >= keepCount {

			log.Info("DeleteStack", "StackId", *stack.StackId)
			err := cloudformation.DeleteStack(cfnClient, *stack.StackId)
			if err != nil {
				log.Error("DeleteStack", "StackId", *stack.StackId, "Error", err)
				pruneStackChan <- false
				return
			}
		} else {
			log.Info("Keeping stack", "StackId", *stack.StackId)
		}
	}

	pruneStackChan <- true
	return
}

func getELBPruneList(log log15.Logger, environment *conf.Environment,
	region *conf.Region, stackList []*cfnlib.Stack,
	roleSession *session.Session, elbTag string) (pruneList []*cfnlib.Stack, success bool) {

	ec2Client := ec2.New(roleSession)
	elbClient := elb.New(roleSession)

	pruneList = make([]*cfnlib.Stack, 0)

	// the elb name as found in the AWS console
	elbName, err := environment.GetELBForRegion(region.Name, elbTag)
	if err != nil || elbName == "" {
		log.Error("Unable to find ELB tagged "+elbTag, "Environment", environment.Name, "Error", err)
		return
	}

	log = log.New("LoadBalancerName", elbName)

	var ineligibleStacks map[string]int
	for i := 0; i <= mixedStackRetryCount; i++ {

		log.Info("DescribeInstanceHealth")
		instanceStates, err := elb.DescribeInstanceHealth(elbClient, elbName)
		if err != nil {
			log.Error("DescribeInstanceHealth", "Error", err)
			return
		}
		if instanceStates == nil {
			log.Error("DescribeInstanceHealth response is null")
			return
		}
		if len(instanceStates) == 0 {
			log.Error("No instances in the ELB")
			return
		}

		// Add stacks with instances attached to the given ELB to the list of
		// stacks ineligible for deletion. Mixed stacks occur when ELB reports
		// instances that were already deregistered as still InService
		ineligibleStacks = make(map[string]int)
		for _, instanceState := range instanceStates {

			if *instanceState.State != elb.InService {
				continue
			}

			filters := make(map[string][]string)
			filters["instance-id"] = []string{*instanceState.InstanceId}
			reservations, err := ec2.DescribeInstances(ec2Client, filters)
			if err != nil {
				log.Error("DescribeInstances", "InstanceId", *instanceState.InstanceId, "Error", err)
				return
			}

			if len(reservations) == 1 && reservations[0] != nil {
				for _, instance := range reservations[0].Instances {
					for _, tags := range instance.Tags {
						if tags != nil && *tags.Key == constants.AwsCfnStackIdTag {
							ineligibleStacks[*tags.Value] = ineligibleStacks[*tags.Value] + 1
						}
					}
				}
			}
		}

		if len(ineligibleStacks) != 1 {
			log.Warn("Found InService instances belonging to more than one stack")
			for stackId, instanceCount := range ineligibleStacks {
				log.Warn("Mixed stack", "StackId", stackId, "InstanceCount", instanceCount)
			}

			if i == mixedStackRetryCount {
				log.Error("Prune failed after waiting for mixed stacks to resolve")
				return
			} else {
				waitTime := time.Duration(10*math.Pow(2, float64(i))) * time.Second
				log.Info("Mixed stacks occur when ELB reports instances that have been deregistered as still InService")
				log.Info(fmt.Sprintf("Waiting %s for instance deregistration", waitTime.String()))
				time.Sleep(waitTime)
				continue
			}
		}

		break
	}

	// len(ineligibleStacks) == 1 by this point
	for stackId := range ineligibleStacks {
		log.Info("Found stack with instances registered to a destination elb", "StackId", stackId)
	}

	// nevertheless this logic prevents bad things from happening if the above
	// assumption isn't true
	// i.e. build the prune list don't prune the stackList
	for _, stack := range stackList {
		if _, exists := ineligibleStacks[*stack.StackId]; !exists {
			pruneList = append(pruneList, stack)
		}
	}

	success = true
	return
}
