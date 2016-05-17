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
package elb

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	elblib "github.com/aws/aws-sdk-go/service/elb"
)

const (
	//instance states
	InService    = "InService"
	OutOfService = "OutOfService"
	Unknown      = "Unknown"
)

// Don't force clients of this package to import
// "github.com/aws/aws-sdk-go/service/elb"
func New(config *session.Session) *elblib.ELB {
	return elblib.New(config)
}

func AddTags(client *elblib.ELB, elbName string, kvps map[string]string) error {

	if len(kvps) == 0 {
		return errors.New("must specify at least one tag")
	}

	tags := make([]*elblib.Tag, 0)
	for key, value := range kvps {
		tag := &elblib.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		}
		tags = append(tags, tag)
	}

	input := &elblib.AddTagsInput{
		LoadBalancerNames: []*string{aws.String(elbName)},
		Tags:              tags,
	}

	_, err := client.AddTags(input)
	if err != nil {
		return err
	}

	return nil
}

// DescribeInstanceHealth using http://docs.aws.amazon.com/sdk-for-go/api/service/elb/ELB.html#DescribeInstanceHealth-instance_method
func DescribeInstanceHealth(client *elblib.ELB, elbName string, instances ...*elblib.Instance) ([]*elblib.InstanceState, error) {

	input := &elblib.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	if len(instances) > 0 {
		input.Instances = instances
	}

	output, err := client.DescribeInstanceHealth(input)
	if err != nil {
		return nil, err
	}

	return output.InstanceStates, nil
}

func DescribeLoadBalancers(client *elblib.ELB, elbNames ...string) ([]*elblib.LoadBalancerDescription, error) {

	if len(elbNames) == 0 {
		return nil, errors.New("must specify at least one elb name")
	}

	loadBalancerNames := make([]*string, 0)
	for _, elbName := range elbNames {
		loadBalancerNames = append(loadBalancerNames, aws.String(elbName))
	}

	input := &elblib.DescribeLoadBalancersInput{
		LoadBalancerNames: loadBalancerNames,
	}

	output, err := client.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	return output.LoadBalancerDescriptions, nil
}

func DescribeTags(client *elblib.ELB, elbNames ...string) ([]*elblib.TagDescription, error) {

	if len(elbNames) == 0 {
		return nil, errors.New("must specify at least one elb name")
	}

	loadBalancerNames := make([]*string, 0)
	for _, elbName := range elbNames {
		loadBalancerNames = append(loadBalancerNames, aws.String(elbName))
	}

	input := &elblib.DescribeTagsInput{
		LoadBalancerNames: loadBalancerNames,
	}

	output, err := client.DescribeTags(input)
	if err != nil {
		return nil, err
	}

	return output.TagDescriptions, nil
}

// RegisterInstancesWithLoadBalancer using http://docs.aws.amazon.com/sdk-for-go/api/service/elb/ELB.html#RegisterInstancesWithLoadBalancer-instance_method
func RegisterInstancesWithLoadBalancer(client *elblib.ELB, elbName string, instanceIds []string) (*elblib.RegisterInstancesWithLoadBalancerOutput, error) {

	instanceList := make([]*elblib.Instance, len(instanceIds))
	for i := 0; i < len(instanceIds); i++ {
		instance := &elblib.Instance{
			InstanceId: aws.String(instanceIds[i]),
		}
		instanceList[i] = instance
	}

	params := &elblib.RegisterInstancesWithLoadBalancerInput{
		Instances:        instanceList,
		LoadBalancerName: aws.String(elbName),
	}
	return client.RegisterInstancesWithLoadBalancer(params)
}

// DeregisterInstancesFromLoadBalancer using http://docs.aws.amazon.com/sdk-for-go/api/service/elb/ELB.html#DeregisterInstancesFromLoadBalancer-instance_method
func DeregisterInstancesFromLoadBalancer(client *elblib.ELB, instances []*elblib.Instance, elbname string) (*elblib.DeregisterInstancesFromLoadBalancerOutput, error) {

	params := &elblib.DeregisterInstancesFromLoadBalancerInput{
		Instances:        instances,
		LoadBalancerName: aws.String(elbname),
	}
	return client.DeregisterInstancesFromLoadBalancer(params)
}
