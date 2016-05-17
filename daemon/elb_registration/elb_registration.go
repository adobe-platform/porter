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
package elb_registration

import (
	"os"
	"strings"

	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon/identity"
	"github.com/adobe-platform/porter/logger"
	elblib "github.com/aws/aws-sdk-go/service/elb"
	"github.com/inconshreveable/log15"
)

func Call() {
	log := logger.Daemon("AWS_STACKID", os.Getenv("AWS_STACKID"))
	log.Info("Auto ELB instance registration")

	elbCSV := os.Getenv("ELBS")
	if elbCSV == "" {
		log.Warn("No ELB list found for autoregistration")
		return
	}

	ii, err := identity.Get(log)
	if err != nil {
		return
	}

	elbClient := elb.New(aws_session.Get(ii.AwsCreds.Region))

	elbNames := strings.Split(elbCSV, ",")

outer:
	for _, elbName := range elbNames {
		log = log.New("LoadBalancerName", elbName)

		tagDescriptions, err := elb.DescribeTags(elbClient, elbName)
		if err != nil {
			log.Error("elb.DescribeTags", "Error", err)
			log.Warn("instance autoregistration is broken")
			continue
		}

		for _, tagDescription := range tagDescriptions {
			for _, tag := range tagDescription.Tags {
				if tag.Key != nil && *tag.Key == constants.PorterStackIdTag {
					if os.Getenv("AWS_STACKID") == *tag.Value {
						log.Info("Instance found its stack tagged to the ELB")
						go tryRegistration(log, elbClient, elbName)
					} else {
						log.Info("Instance did not find its stack tagged to the ELB")
					}
					continue outer
				}
			}
		}

		log.Warn("Didn't find tag key " + constants.PorterStackIdTag)
	}
}

func tryRegistration(log log15.Logger, elbClient *elblib.ELB, elbName string) {
	ii, err := identity.Get(log)
	if err != nil {
		return
	}

	instanceIds := []string{ii.Instance.InstanceID}

	log.Info("RegisterInstancesWithLoadBalancer", "InstanceId", ii.Instance.InstanceID)
	_, err = elb.RegisterInstancesWithLoadBalancer(elbClient, elbName, instanceIds)
	if err != nil {
		log.Error("RegisterInstancesWithLoadBalancer", "Error", err)
		log.Error("instance autoregistration failed")
	}
}
