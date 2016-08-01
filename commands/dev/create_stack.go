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
	"os/signal"
	"sort"
	"time"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws/ec2"
	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision"
	"github.com/adobe-platform/porter/prune"
	"github.com/adobe-platform/porter/util"
	ec2lib "github.com/aws/aws-sdk-go/service/ec2"
	elblib "github.com/aws/aws-sdk-go/service/elb"
	"github.com/inconshreveable/log15"
	"github.com/phylake/go-cli"
)

const sleepDuration = 10 * time.Second

type CreateStackCmd struct{}

func (recv *CreateStackCmd) Name() string {
	return "create-stack"
}

func (recv *CreateStackCmd) ShortHelp() string {
	return "Create a developer CloudFormation stack"
}

func (recv *CreateStackCmd) LongHelp() string {
	return `NAME
    create-stack -- Create a developer CloudFormation stack

SYNOPSIS
    create-stack -e <environment out of .porter/config> [-keep <int>] [-block=f]

DESCRIPTION
    Create a developer stack in a single AWS region with a single instance.

    This is similar to the combination of
    'porter build pack && porter build provision' commands but it meant to be
    run locally to create infrastructure as close as possible to a real
    production environment.

    Stack formation events will stream in. Once you see the "EC2 instance"
    message your EC2 had been created and is initializing. Wait a few seconds
    for sshd to initialize and then you can SSH into the instance.

    For this command to work you need permission to call AssumeRole on the Role
    ARN defined in the environment you're targeting. If my user's arn is
    arn:aws:iam::123456789012:user/bob then I should find in the trust policy of
    the role arn for my environment an entry like this

        {
          "Version": "2012-10-17",
          "Statement": [
            {
              "Effect": "Allow",
              "Principal": {
                "AWS": [
                  "arn:aws:iam::123456789012:user/bob"
                ]
              },
              "Action": "sts:AssumeRole"
            }
          ]
        }

OPTIONS
    -e  environment from .porter/config

    -keep
        The number of stacks to keep with the status CREATE_COMPLETE or
        UPDATE_COMPLETE.

        The default is 0 meaning only no stacks will be kept.

    -block
        Return after CREATE_COMPLETE instead of blocking.
        The default is t. Set -block=f to return`
}

func (recv *CreateStackCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *CreateStackCmd) Execute(args []string) bool {
	if len(args) > 0 {

		var (
			environment string
			keepCount   int
			block       bool
		)

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)
		flagSet.StringVar(&environment, "e", "", "")
		flagSet.IntVar(&keepCount, "keep", 0, "")
		flagSet.BoolVar(&block, "block", true, "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		if environment == "" || keepCount < 0 {
			return false
		}

		stackArgs := provision.StackArgs{
			Environment: environment,
			KeepCount:   keepCount,
		}

		CreateStack(stackArgs, block)
		return true
	}
	return false
}

func CreateStack(stackArgs provision.StackArgs, block bool) {
	var (
		environment  *conf.Environment
		regionName   string
		stackCreated bool
	)

	log := logger.CLI()

	config, success := conf.GetConfig(log)
	if !success {
		os.Exit(1)
	}

	if os.Getenv(constants.EnvConfig) != "" {
		config.Print()
	}

	environment, err := config.GetEnvironment(stackArgs.Environment)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		os.Exit(1)
	}

	if !prune.Do(log, config, environment, stackArgs.KeepCount, false, "") {
		os.Exit(1)
	}

	regionCount := len(environment.Regions)
	if regionCount != 1 {
		msg := fmt.Sprintf("create-stack deploys to a single region. Found %d", regionCount)
		log.Error(msg)
		os.Exit(1)
	}

	if !preFlight(log) {
		os.Exit(1)
	}

	if !provision.Package(log, config) {
		os.Exit(1)
	}

	stackOutput, success := provision.CreateStack(log, config, stackArgs)
	if !success {
		log.Error("Create stack failed")
		os.Exit(1)
	}

	if len(stackOutput.Regions) != 1 {
		log.Error("unexpected number of regions in stack output", "RegionCount", len(stackOutput.Regions))
		os.Exit(1)
	}

	regionName = environment.Regions[0].Name
	stackId := stackOutput.Regions[regionName].StackId

	outputFile, err := os.Create(constants.CreateStackOutputPath)
	if err != nil {
		log.Error("couldn't write stack output")
		os.Exit(1)
	}

	err = json.NewEncoder(outputFile).Encode(stackOutput)
	if err != nil {
		log.Error("couldn't write stack output")
		os.Exit(1)
	}

	roleARN, err := environment.GetRoleARN(regionName)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(regionName, roleARN, constants.StackCreationTimeout())
	cfnClient := cloudformation.New(roleSession)
	ec2Client := ec2.New(roleSession)
	elbClient := elb.New(roleSession)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		log.Warn("Received SIGINT. Deleting stack", "StackId", stackId)
		cloudformation.DeleteStack(cfnClient, stackId)

		// http://tldp.org/LDP/abs/html/exitcodes.html
		os.Exit(130)
	}()

	stackEventState := cloudformation.NewStackEventState(cfnClient, stackId)
	eventIds := make(map[string]interface{})

	log.Info("Polling for stack events every " + sleepDuration.String())
	log.Info("The service payload is located at " + constants.PayloadPath)
	log.Info("The CloudFormation template is located at " + constants.CloudFormationTemplatePath)
	log.Info("Press Ctrl+C to exit and delete the stack")

	n := int(constants.StackCreationTimeout().Seconds() / sleepDuration.Seconds())
	// spin until the stack will rollback
outer:
	for i := 0; i < n; i++ {

		stackEvents, err := stackEventState.DescribeStackEvents()
		if err != nil {
			log.Error("DescribeStackEvents", "Error", err)
			os.Exit(1)
		}
		sort.Sort(cloudformation.StackEventByTime(stackEvents))

		for _, stackEvent := range stackEvents {
			if _, exists := eventIds[*stackEvent.EventId]; exists == true {
				continue
			} else {
				eventIds[*stackEvent.EventId] = nil
			}

			if stackEvent.ResourceType != nil &&
				stackEvent.ResourceStatus != nil &&
				stackEvent.LogicalResourceId != nil {

				switch *stackEvent.ResourceStatus {
				case "CREATE_IN_PROGRESS":
					log.Info("CREATE_IN_PROGRESS",
						"LogicalId", *stackEvent.LogicalResourceId,
						"Type", *stackEvent.ResourceType,
					)
				case "CREATE_COMPLETE":
					switch *stackEvent.ResourceType {
					case cfn.AutoScaling_AutoScalingGroup:
						go logInstanceDNS(log, ec2Client, stackId)
					case cfn.ElasticLoadBalancing_LoadBalancer:
						go logELBDNS(log, elbClient, *stackEvent.PhysicalResourceId)
					case cfn.CloudFormation_Stack:
						stackCreated = true
						break outer
					}

					log.Info("CREATE_COMPLETE",
						"LogicalId", *stackEvent.LogicalResourceId,
						"Type", *stackEvent.ResourceType,
					)
				case "CREATE_FAILED":
					log.Error("CREATE_FAILED",
						"LogicalId", *stackEvent.LogicalResourceId,
						"Type", *stackEvent.ResourceType,
					)
				default:
					log.Info("Stack event",
						"LogicalId", *stackEvent.LogicalResourceId,
						"Type", *stackEvent.ResourceType,
						"Status", *stackEvent.ResourceStatus,
					)
				}
			}
		}

		time.Sleep(sleepDuration)
	}

	if stackCreated {
		log.Info("Stack creation is complete")

		if block {

			log.Debug("Ctrl+C to delete it and exit")

			// continue blocking so SIGINT can delete the stack to avoid needing to
			// interact with the AWS console
			for {
				time.Sleep(60 * time.Minute)
			}
		}
	} else {

		log.Warn("Stack has been rolled back. Exiting.")
	}

	return
}

func preFlight(log log15.Logger) bool {
	if !util.OverrideDockerClient(log) {
		return false
	}

	util.GitIgnoreTempDir(log)
	util.DockerIgnoreTempDir(log)

	return true
}

func logInstanceDNS(log log15.Logger, ec2Client *ec2lib.EC2, stackId string) {
	filters := make(map[string][]string)
	filters["tag:"+constants.AwsCfnStackIdTag] = []string{stackId}

	for {
		time.Sleep(1 * time.Second)

		reservations, err := ec2.DescribeInstances(ec2Client, filters)
		if err != nil {
			log.Error("DescribeInstances", "Error", err)
			break
		}

		if reservations == nil || len(reservations) != 1 || reservations[0] == nil {
			continue
		}

		instances := reservations[0].Instances

		if instances == nil || len(instances) != 1 {
			continue
		}

		instance := instances[0]

		if instance.PublicDnsName != nil && *instance.PublicDnsName != "" {
			log.Info("EC2 instance",
				"User", constants.AmazonLinuxUser,
				"PublicDnsName", *instance.PublicDnsName,
			)
		} else if instance.PublicIpAddress != nil && *instance.PublicIpAddress != "" {
			log.Info("EC2 instance",
				"User", constants.AmazonLinuxUser,
				"PublicIpAddress", *instance.PublicIpAddress,
			)
		} else if instance.PrivateIpAddress != nil && *instance.PrivateIpAddress != "" {
			log.Info("EC2 instance",
				"User", constants.AmazonLinuxUser,
				"PrivateIpAddress", *instance.PrivateIpAddress,
			)
		}

		break
	}
}

func logELBDNS(log log15.Logger, client *elblib.ELB, elbName string) {
	output, err := elb.DescribeLoadBalancers(client, elbName)
	if err != nil {
		log.Error("DescribeLoadBalancers", "Error", err)
		return
	}
	if len(output) != 1 {
		log.Error("Unexpected number of load balancers", "Count", len(output))
		return
	}
	log.Info("ELB", "PublicDnsName", *output[0].DNSName)
}
