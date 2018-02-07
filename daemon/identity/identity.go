/*
 * (c) 2016-2018 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package identity

import (
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon/flags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/inconshreveable/log15.v2"
)

var (
	ec2Client            *ec2.EC2
	instanceIdentity     *InstanceIdentity
	instanceIdentityLock sync.RWMutex
)

type InstanceIdentity struct {
	Instance struct {
		InstanceID string `json:"instance-id"`
	} `json:"instance"`
	Tags  map[string]string `json:"Tags"`
	Stack struct {
		Parameters struct {
			Environment string `json:"EnvironmentType"`
		} `json:"Parameters"`
	} `json:"Stack"`
	AwsCreds struct {
		Region string `json:"region"`
	} `json:"aws_creds"`
}

func Get(log log15.Logger) (*InstanceIdentity, error) {
	instanceIdentityLock.RLock()

	if instanceIdentity == nil {

		instanceIdentityLock.RUnlock()
		instanceIdentityLock.Lock()
		defer instanceIdentityLock.Unlock()

		// Multiple readers could be at instanceIdentityLock.Lock()
		// Check condition again
		if instanceIdentity == nil {
			if err := populateInstanceIdentity(log); err != nil {
				return nil, err
			}
		}
	} else {
		instanceIdentityLock.RUnlock()
	}

	return instanceIdentity, nil
}

func populateInstanceIdentity(log log15.Logger) error {

	log = log.New("Method", "identity.Get")

	//Get instance id
	instanceIdResp, err := http.Get(constants.EC2MetadataURL + "/instance-id")
	if err != nil {
		log.Error("Error on instanceIdResp", "Error", err)
		return err
	}
	defer instanceIdResp.Body.Close()

	//Get AWS Region
	awsRegionResp, err := http.Get(constants.EC2MetadataURL + "/placement/availability-zone")
	if err != nil {
		log.Error("Error on awsRegionResp", "Error", err)
		return err
	}
	defer awsRegionResp.Body.Close()

	bs, err := ioutil.ReadAll(instanceIdResp.Body)
	if err != nil {
		log.Error("ioutil.ReadAll instanceIdResp", "Error", err)
		return err
	}

	region, err := ioutil.ReadAll(awsRegionResp.Body)
	if err != nil {
		log.Error("ioutil.ReadAll awsRegionResp", "Error", err)
		return err
	}
	awsRegion := string(region)
	//strip down the AZ char
	awsRegion = awsRegion[:len(awsRegion)-1]
	instanceId := string(bs)

	ec2Client = ec2.New(aws_session.Get(awsRegion))

	// Only get tags for our ec2 instance.
	// https://github.com/aws/aws-sdk-go/blob/ea83c25c44525da47e8044bbd21e4045758ea39b/service/autoscaling/api.go#L2818
	tagsResp, err := ec2Client.DescribeTags(&ec2.DescribeTagsInput{
		// https://github.com/aws/aws-sdk-go/blob/ea83c25c44525da47e8044bbd21e4045758ea39b/service/autoscaling/api.go#L3141
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []*string{aws.String(instanceId)},
			},
		},
	})
	if err != nil {
		log.Error("ec2Client.DescribeTags", "Error", err)
		return err
	}

	instanceIdentity = &InstanceIdentity{}
	instanceIdentity.Instance.InstanceID = instanceId
	instanceIdentity.AwsCreds.Region = awsRegion
	instanceIdentity.Stack.Parameters.Environment = flags.Environment
	instanceIdentity.Tags = make(map[string]string)

	// https://github.com/aws/aws-sdk-go/blob/ea83c25c44525da47e8044bbd21e4045758ea39b/service/autoscaling/api.go#L4029
	for _, tagDescription := range tagsResp.Tags {
		// `aws --region us-west-2 ec2 describe-tags --filter Name=resource-id,Values=i-abc123`

		if tagDescription.Key != nil && tagDescription.Value != nil {

			instanceIdentity.Tags[*tagDescription.Key] = *tagDescription.Value
		}
	}

	return nil
}

func SetIdentityForTest(value *InstanceIdentity) {
	instanceIdentity = value
}
