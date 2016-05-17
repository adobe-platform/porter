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
package promote

import (
	"time"

	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision_output"
	"github.com/aws/aws-sdk-go/aws"
	elblib "github.com/aws/aws-sdk-go/service/elb"
	"github.com/inconshreveable/log15"
)

const (
	sleepDuration = 10 * time.Second
	pollDuration  = 10 * time.Minute
)

func Promote(log log15.Logger, config *conf.Config, provisionedEnv *provision_output.Environment, elb string) (success bool) {

	environment := provisionedEnv.Environment

	stackCount := len(provisionedEnv.Regions)
	promoteServiceChan := make(chan bool, stackCount)

	for _, provisionedRegion := range provisionedEnv.Regions {
		go func(provisionedRegion provision_output.Region) {
			promoteServiceChan <- promoteService(log, environment, provisionedRegion, config, elb)
		}(provisionedRegion)
	}

	for i := 0; i < stackCount; i++ {
		select {
		case promoteSuccess := <-promoteServiceChan:
			if !promoteSuccess {
				return
			}
		}
	}

	success = true
	return
}

func promoteService(log log15.Logger, env string, provisionedRegion provision_output.Region, config *conf.Config, elbTag string) (success bool) {
	log = log.New("Region", provisionedRegion.AWSRegion)

	environment, err := config.GetEnvironment(env)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		return
	}

	region, err := environment.GetRegion(provisionedRegion.AWSRegion)
	if err != nil {
		log.Error("GetRegion", "Error", err)
		return
	}

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, 1*time.Hour)
	elbClient := elb.New(roleSession)

	destinationELB, err := environment.GetELBForRegion(region.Name, elbTag)
	if err != nil || destinationELB == "" {
		log.Error("Unable to find the ELB", "Environment", environment.Name)
		return
	}
	log.Info("Destination ELB", "LoadBalancerName", destinationELB)
	log.Info("Source ELB", "LoadBalancerName", provisionedRegion.ProvisionedELBName)

	oldInstanceStates, err := elb.DescribeInstanceHealth(elbClient, destinationELB)
	if err != nil {
		log.Error("DescribeInstanceHealth", "LoadBalancerName", destinationELB, "Error", err)
		return
	}

	newInstanceStates, err := elb.DescribeInstanceHealth(elbClient, provisionedRegion.ProvisionedELBName)
	if err != nil {
		log.Error("DescribeInstanceHealth", "LoadBalancerName", provisionedRegion.ProvisionedELBName, "Error", err)
		return
	}
	if newInstanceStates == nil {
		log.Error("DescribeInstanceHealth response is null", "LoadBalancerName", provisionedRegion.ProvisionedELBName)
		return
	}

	newInstances := make([]*elblib.Instance, 0)
	newInstanceIdToInService := make(map[string]bool)
	for _, newInstanceState := range newInstanceStates {
		newInstanceIdToInService[*newInstanceState.InstanceId] = false

		instance := &elblib.Instance{
			InstanceId: aws.String(*newInstanceState.InstanceId),
		}
		newInstances = append(newInstances, instance)
	}

	//cover cases for new instances that are already in the live ELB (promote re-run)
	oldInstances := make([]*elblib.Instance, 0)
	for _, oldInstanceState := range oldInstanceStates {
		if oldInstanceState == nil || oldInstanceState.InstanceId == nil {
			continue
		}

		if _, exists := newInstanceIdToInService[*oldInstanceState.InstanceId]; !exists {
			instance := &elblib.Instance{
				InstanceId: aws.String(*oldInstanceState.InstanceId),
			}
			oldInstances = append(oldInstances, instance)
		}
	}

	log.Info("Waiting for newly provisioned instances to be InService", "LoadBalancerName", provisionedRegion.ProvisionedELBName)
	if ok := waitForInServiceInstances(log, elbClient, provisionedRegion.ProvisionedELBName, nil); !ok {
		log.Error("Instances never became InService in the newly provisioned ELB", "LoadBalancerName", provisionedRegion.ProvisionedELBName)
		return
	}

	newInstanceIds := make([]string, len(newInstanceStates))
	for i := 0; i < len(newInstanceStates); i++ {
		newInstanceIds[i] = *newInstanceStates[i].InstanceId
	}

	log.Info("RegisterInstancesWithLoadBalancer", "LoadBalancerName", destinationELB)
	_, err = elb.RegisterInstancesWithLoadBalancer(elbClient, destinationELB, newInstanceIds)
	if err != nil {
		log.Error("RegisterInstancesWithLoadBalancer", "Error", err, "LoadBalancerName", destinationELB)
		return
	}

	// Wait for all newly registered instances (ignore old instances) to be InService
	// If this fails attempt to deregister the registered instances so there isn't a mixed set waiting around
	log.Info("Waiting for newly registered instances to be InService", "LoadBalancerName", destinationELB)
	if ok := waitForInServiceInstances(log, elbClient, destinationELB, newInstanceIdToInService); !ok {
		log.Error("Instances never became InService in the destination ELB", "LoadBalancerName", destinationELB)
		deregisterInstances(log, elbClient, destinationELB, newInstances)
		return
	}

	if len(oldInstances) > 0 {
		deregisterInstances(log, elbClient, destinationELB, oldInstances)
	} else {
		log.Warn("Nothing to remove from ELB", "LoadBalancerName", destinationELB)
	}

	elbTags := make(map[string]string)
	elbTags[constants.PorterStackIdTag] = provisionedRegion.StackId
	err = elb.AddTags(elbClient, destinationELB, elbTags)
	if err != nil {
		log.Warn("elb.AddTags", "Error", err)
		log.Warn("Instance autoregistration will be broken")
	}

	success = true
	return

}

func waitForInServiceInstances(log log15.Logger, elbClient *elblib.ELB, elbName string, instanceIdToInService map[string]bool) bool {
	log = log.New("LoadBalancerName", elbName)

	iterations := int(pollDuration.Seconds() / sleepDuration.Seconds())
	for i := 0; i < iterations; i++ {

		if i > 0 {
			time.Sleep(sleepDuration)
		}

		instanceStates, err := elb.DescribeInstanceHealth(elbClient, elbName)
		if err != nil {
			log.Crit("DescribeInstanceHealth", "Error", err)
			return false
		}

		allInService := true
		for _, instanceState := range instanceStates {

			if instanceState == nil {
				break
			}

			inService := *instanceState.State == elb.InService
			log.Info("Instance",
				"InstanceId", *instanceState.InstanceId,
				"InstanceState", *instanceState.State,
			)

			if _, exists := instanceIdToInService[*instanceState.InstanceId]; exists {
				instanceIdToInService[*instanceState.InstanceId] = inService
			}

			if !inService {
				allInService = false
			}
		}

		// if a list of instances was input, only check if those are all
		// InService
		if len(instanceIdToInService) > 0 {

			allInputInService := true
			for _, inService := range instanceIdToInService {
				if !inService {
					allInputInService = false
					break
				}
			}

			if allInputInService {
				log.Info("All ELB instances are InService")
				return true
			}
		} else if allInService {
			log.Info("All ELB instances are InService")
			return true
		}
	}

	log.Crit("Never found all InService instances")
	return false
}

func deregisterInstances(log log15.Logger, elbClient *elblib.ELB, elbName string, instances []*elblib.Instance) {
	log = log.New("LoadBalancerName", elbName)

	log.Info("DeregisterInstancesFromLoadBalancer")
	deRegResp, err := elb.DeregisterInstancesFromLoadBalancer(elbClient, instances, elbName)
	if err != nil {
		log.Error("DeregisterInstancesFromLoadBalancer", "Error", err)
		return
	}
	if deRegResp != nil && deRegResp.Instances != nil {
		for _, instance := range deRegResp.Instances {
			if instance.InstanceId == nil {
				log.Warn("nil InstanceId")
				continue
			}
			log.Info("DeregisterInstancesFromLoadBalancer response", "InstanceId", *instance.InstanceId)
		}
	} else {
		log.Warn("DeregisterInstancesFromLoadBalancer empty response")
	}
}
