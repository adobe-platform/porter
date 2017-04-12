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
package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/phylake/go-cli"
)

type ElbCmd struct{}

func (recv *ElbCmd) Name() string {
	return "elb"
}

func (recv *ElbCmd) ShortHelp() string {
	return "Create an ELB to promote instances into"
}

func (recv *ElbCmd) LongHelp() string {
	return `NAME
    elb -- Create an ELB to promote instances into

SYNOPSIS
    elb -name <string> -region <string> [-ssl-arn <string>] [-subnet-ids <string>[,<string>]] [-sg-id <string>]

DESCRIPTION
    Create an ELB with settings that work with porter.

OPTIONS
    -name
        The name of the ELB to create

    -region
        The region into which to create a ELB

    -ssl-arn
    	The ARN of an SSL cert to enable HTTPS

    -subnet-ids
        The ids of subnets for the ELB to load balance across
        (VPC only)

    -sg-id
        The id of a security group to attach to the ELB
        (VPC only)`
}

func (recv *ElbCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *ElbCmd) Execute(args []string) bool {
	if len(args) > 0 {
		var elbName, sslArn, region, subnetCSV, securityGroupId string

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)
		flagSet.StringVar(&elbName, "name", "", "")
		flagSet.StringVar(&region, "region", "", "")
		flagSet.StringVar(&sslArn, "ssl-arn", "", "")
		flagSet.StringVar(&subnetCSV, "subnet-ids", "", "")
		flagSet.StringVar(&securityGroupId, "sg-id", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		if elbName == "" {
			return false
		}

		if _, exists := constants.AwsRegions[region]; !exists {
			return false
		}

		if subnetCSV != "" && securityGroupId == "" {
			return false
		}

		if securityGroupId != "" && subnetCSV == "" {
			return false
		}

		subnetIds := strings.Split(subnetCSV, ",")
		for i := 0; i < len(subnetIds); i++ {
			subnetId := subnetIds[i]
			if !strings.HasPrefix(subnetId, "subnet-") {
				subnetIds[i] = "subnet-" + subnetId
			}
		}

		if securityGroupId != "" && !strings.HasPrefix(securityGroupId, "sg-") {
			securityGroupId = "sg-" + securityGroupId
		}

		if !bootstrapELB(elbName, region, sslArn, subnetIds, securityGroupId) {
			os.Exit(1)
		}

		return true
	}
	return false
}

func bootstrapELB(elbName, region, sslArn string, subnetIds []string, securityGroupId string) (success bool) {
	log := logger.CLI()

	session := session.New(aws.NewConfig().WithRegion(region))
	ec2Client := ec2.New(session)
	elbClient := elb.New(session)

	var azs, subnets, securityGroups []*string

	if len(subnetIds) > 0 {
		subnets = aws.StringSlice(subnetIds)
		securityGroups = []*string{&securityGroupId}
	} else {

		describeAvailabilityZonesInput := &ec2.DescribeAvailabilityZonesInput{}

		describeAvailabilityZonesOutput, err := ec2Client.DescribeAvailabilityZones(describeAvailabilityZonesInput)
		if err != nil {
			log.Error("DescribeAvailabilityZones", "Error", err)
			return
		}

		azs = make([]*string, 0)
		for _, availabilityZone := range describeAvailabilityZonesOutput.AvailabilityZones {
			azs = append(azs, availabilityZone.ZoneName)
		}
	}

	listeners := []*elb.Listener{
		{
			InstancePort:     aws.Int64(80),
			InstanceProtocol: aws.String("HTTP"),
			LoadBalancerPort: aws.Int64(80),
			Protocol:         aws.String("HTTP"),
		},
	}

	if sslArn != "" {
		listener := &elb.Listener{
			InstancePort:     aws.Int64(8080),
			InstanceProtocol: aws.String("HTTP"),
			LoadBalancerPort: aws.Int64(443),
			Protocol:         aws.String("HTTPS"),
			SSLCertificateId: aws.String(sslArn),
		}
		listeners = append(listeners, listener)
	}

	createLoadBalancerInput := &elb.CreateLoadBalancerInput{
		AvailabilityZones: azs,
		Listeners:         listeners,
		LoadBalancerName:  aws.String(elbName),
		SecurityGroups:    securityGroups,
		Subnets:           subnets,
	}

	_, err := elbClient.CreateLoadBalancer(createLoadBalancerInput)
	if err != nil {
		log.Error("CreateLoadBalancer", "Error", err)
		return
	}
	log.Info("Created ELB: " + elbName)

	configureHealthCheckInput := &elb.ConfigureHealthCheckInput{
		HealthCheck: &elb.HealthCheck{
			HealthyThreshold:   aws.Int64(constants.HC_HealthyThreshold),
			Interval:           aws.Int64(constants.HC_Interval),
			Target:             aws.String("HTTP:80/health"),
			Timeout:            aws.Int64(constants.HC_Timeout),
			UnhealthyThreshold: aws.Int64(constants.HC_UnhealthyThreshold),
		},
		LoadBalancerName: aws.String(elbName),
	}

	_, err = elbClient.ConfigureHealthCheck(configureHealthCheckInput)
	if err != nil {
		log.Error("ConfigureHealthCheck", "Error", err)
		return
	}
	log.Info("Health Check: HTTP:80/health")

	modifyLoadBalancerAttributesInput := &elb.ModifyLoadBalancerAttributesInput{
		LoadBalancerAttributes: &elb.LoadBalancerAttributes{
			ConnectionDraining: &elb.ConnectionDraining{
				Enabled: aws.Bool(true),
				Timeout: aws.Int64(300),
			},
			ConnectionSettings: &elb.ConnectionSettings{
				IdleTimeout: aws.Int64(60),
			},
			CrossZoneLoadBalancing: &elb.CrossZoneLoadBalancing{
				Enabled: aws.Bool(true),
			},
		},
		LoadBalancerName: aws.String(elbName),
	}

	_, err = elbClient.ModifyLoadBalancerAttributes(modifyLoadBalancerAttributesInput)
	if err != nil {
		log.Error("ModifyLoadBalancerAttributes", "Error", err)
		return
	}

	log.Info("Cross-Zone Load Balancing: Enabled")
	log.Info("Connection Settings: Idle Timeout: 60 seconds")
	log.Info("Connection Draining: Enabled, 300 seconds")

	success = true
	return
}
