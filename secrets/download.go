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
package secrets

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
	"os"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/inconshreveable/log15.v2"
)

func Download(log log15.Logger, region *conf.Region) (secretsPayload Payload, success bool) {

	log.Debug("secrets.Download() BEGIN")
	defer log.Debug("secrets.Download() END")

	symmetricKey, secretsLocation, getSecretsKeySuccess := getSecretsKey(log, region.Name)
	if !getSecretsKeySuccess {
		return
	}

	s3Client := s3.New(aws_session.Get(region.Name))

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(region.S3Bucket),
		Key:    aws.String(secretsLocation),
	}

	getObjectOutput, err := s3Client.GetObject(getObjectInput)
	if err != nil {
		log.Crit("GetObject", "Error", err)
		return
	}
	defer getObjectOutput.Body.Close()

	secretsPayloadBytesEnc, err := ioutil.ReadAll(getObjectOutput.Body)
	if err != nil {
		log.Crit("ioutil.ReadAll", "Error", err)
		return
	}

	secretsPayloadBytes, err := Decrypt(secretsPayloadBytesEnc, symmetricKey)
	if err != nil {
		log.Crit("secrets.Decrypt", "Error", err)
		return
	}

	secretsPayload = Payload{}

	err = gob.NewDecoder(bytes.NewReader(secretsPayloadBytes)).Decode(&secretsPayload)
	if err != nil {
		log.Crit("gob.Decode", "Error", err)
		return
	}

	success = true
	return
}

func getSecretsKey(log log15.Logger, region string) (symmetricKey []byte, secretsPayloadLoc string, success bool) {

	log.Debug("getSecretsKey() BEGIN")
	defer log.Debug("getSecretsKey() END")

	var (
		describeStacksOutput *cloudformation.DescribeStacksOutput
		err                  error
	)

	cfnClient := cloudformation.New(aws_session.Get(region))

	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(os.Getenv("AWS_STACKID")),
	}

	retryMsg := func(i int) { log.Warn("DescribeStacks retrying", "Count", i) }
	if !util.SuccessRetryer(7, retryMsg, func() bool {
		describeStacksOutput, err = cfnClient.DescribeStacks(describeStacksInput)
		if err != nil {
			log.Error("DescribeStacks", "Error", err)
			return false
		}
		if len(describeStacksOutput.Stacks) == 0 {
			log.Error("len(describeStacksOutput.Stacks == 0)")
			return false
		}
		return true
	}) {
		log.Crit("Failed to DescribeStacks")
		return
	}

	if len(describeStacksOutput.Stacks) != 1 {
		log.Crit("len(describeStacksOutput.Stacks != 1)")
		return
	}

	stack := describeStacksOutput.Stacks[0]
	for _, param := range stack.Parameters {
		switch *param.ParameterKey {
		case constants.ParameterSecretsKey:
			symmetricKey, err = hex.DecodeString(*param.ParameterValue)
			if err != nil {
				log.Crit("hex.DecodeString", "Error", err)
				return
			}
		case constants.ParameterSecretsLoc:
			secretsPayloadLoc = *param.ParameterValue
		}
	}

	if len(symmetricKey) == 0 {
		log.Crit("missing parameter key " + constants.ParameterSecretsKey)
		return
	}

	if len(secretsPayloadLoc) == 0 {
		log.Crit("missing parameter key " + constants.ParameterSecretsLoc)
		return
	}

	success = true
	return
}
