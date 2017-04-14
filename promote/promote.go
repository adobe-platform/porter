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
package promote

import (
	"time"

	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision_state"
	"github.com/aws/aws-sdk-go/aws"
	elblib "github.com/aws/aws-sdk-go/service/elb"
	"gopkg.in/inconshreveable/log15.v2"
)

const (
	sleepDuration = 10 * time.Second
	pollDuration  = 10 * time.Minute
)

func Promote(log log15.Logger, config *conf.Config, stack *provision_state.Stack, elb string) (success bool) {

	successChan := make(chan bool)

	for regionName, regionState := range stack.Regions {

		go func(regionName string, regionState *provision_state.Region) {

			successChan <- promoteService(log, stack.Environment, regionName,
				regionState, config, elb)

		}(regionName, regionState)
	}

	for i := 0; i < len(stack.Regions); i++ {
		promoteSuccess := <-successChan
		if !promoteSuccess {
			return
		}
	}

	success = true
	return
}

func promoteService(log log15.Logger, env, regionName string,
	regionState *provision_state.Region, config *conf.Config, elbTag string) (success bool) {

	log = log.New("Region", regionName)

	environment, err := config.GetEnvironment(env)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		return
	}

	region, err := environment.GetRegion(regionName)
	if err != nil {
		log.Error("GetRegion", "Error", err)
		return
	}

	if region.PrimaryTopology() != conf.Topology_Inet || !region.HasELB() {
		success = true
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
	log.Info("Source ELB", "LoadBalancerName", regionState.ProvisionedELBName)

	oldInstanceStates, err := elb.DescribeInstanceHealth(elbClient, destinationELB)
	if err != nil {
		log.Error("DescribeInstanceHealth", "LoadBalancerName", destinationELB, "Error", err)
		return
	}

	newInstanceStates, err := elb.DescribeInstanceHealth(elbClient, regionState.ProvisionedELBName)
	if err != nil {
		log.Error("DescribeInstanceHealth", "LoadBalancerName", regionState.ProvisionedELBName, "Error", err)
		return
	}
	if newInstanceStates == nil {
		log.Error("DescribeInstanceHealth response is null", "LoadBalancerName", regionState.ProvisionedELBName)
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

	log.Info("Waiting for newly provisioned instances to be InService", "LoadBalancerName", regionState.ProvisionedELBName)
	if ok := waitForInServiceInstances(log, elbClient, regionState.ProvisionedELBName, nil); !ok {
		log.Error("Instances never became InService in the newly provisioned ELB", "LoadBalancerName", regionState.ProvisionedELBName)
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
	elbTags[constants.PorterStackIdTag] = regionState.StackId
	elbTags[constants.PorterVersionTag] = constants.Version
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
