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
	"github.com/adobe-platform/porter/util"
	elblib "github.com/aws/aws-sdk-go/service/elb"
)

func Call() {
	stackId := os.Getenv("AWS_STACKID")
	log := logger.Daemon("AWS_STACKID", stackId)

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
	instanceIds := []string{ii.Instance.InstanceID}

	elbClient := elb.New(aws_session.Get(ii.AwsCreds.Region))

	elbNames := strings.Split(elbCSV, ",")

outer:
	for _, elbName := range elbNames {
		log := log.New("LoadBalancerName", elbName)

		var tagDescriptions []*elblib.TagDescription
		var err error

		retryMsg := func(i int) { log.Warn("elb.DescribeTags retrying", "Count", i) }
		if !util.SuccessRetryer(8, retryMsg, func() bool {

			tagDescriptions, err = elb.DescribeTags(elbClient, elbName)
			if err != nil {
				log.Error("elb.DescribeTags", "Error", err)
				return false
			}

			return true
		}) {
			log.Warn("elb.DescribeTags failed")
			continue
		}

		for _, tagDescription := range tagDescriptions {
			for _, tag := range tagDescription.Tags {
				if tag.Key == nil || *tag.Key != constants.PorterStackIdTag {
					continue
				}

				if *tag.Value != stackId {
					log.Info("Instance is NOT associated with a stack that was promoted into this ELB")
					continue outer
				}

				log.Info("Instance IS associated with a stack that was promoted into this ELB")
				log.Info("RegisterInstancesWithLoadBalancer", "InstanceId", ii.Instance.InstanceID)

				retryMsg := func(i int) { log.Warn("elb.RegisterInstancesWithLoadBalancer retrying", "Count", i) }
				if !util.SuccessRetryer(8, retryMsg, func() bool {

					_, err = elb.RegisterInstancesWithLoadBalancer(elbClient, elbName, instanceIds)
					if err != nil {
						log.Error("RegisterInstancesWithLoadBalancer", "Error", err)
						return false
					}

					return true
				}) {
					log.Error("Instance Registration failed")
				}

				continue outer
			}
		}

		log.Warn("Didn't find tag key " + constants.PorterStackIdTag)
	}
}
