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
package provision

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision_state"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/inconshreveable/log15"
)

type (
	// A struct for manipulating a Cloudformation stack in a single region
	stackCreator struct {
		log log15.Logger

		config      conf.Config
		environment conf.Environment
		region      conf.Region

		servicePayloadKey      string
		servicePayloadChecksum string

		secretsKey      string
		secretsLocation string

		roleSession *session.Session

		// Stack creation is mostly the same between CreateStack and UpdateStack
		// The difference is in the API call to CloudFormation
		cfnAPI func(*cloudformation.CloudFormation, CfnApiInput) (string, bool)

		templateTransforms map[string][]MapResource

		asgMin     int
		asgDesired int
		asgMax     int
	}
)

const (
	s3KeyOptTemplate = 1 << iota
	s3KeyOptDeployment
)

func (recv *stackCreator) createUpdateStackForRegion(regionState *provision_state.Region) bool {

	asgId := new(string)

	if !recv.getAsgId(asgId) {
		return false
	}

	if *asgId != "" {
		if !recv.getAsgSize(*asgId, regionState) {
			return false
		}
	}

	checksum, success := recv.uploadServicePayload()
	if !success {
		// uploadServicePayload logs errors. all we care about is success
		return false
	}

	if !recv.uploadSecrets(checksum) {
		// uploadSecrets logs errors. all we care about is success
		return false
	}

	stackId, success := recv.createStack()
	if !success {
		// createStack logs errors. all we care about is success
		return false
	}

	regionState.StackId = stackId

	return true
}

func (recv *stackCreator) getAsgId(asgId *string) (success bool) {

	if len(recv.region.ELBs) == 0 {
		success = true
		return
	} else if len(recv.region.ELBs) > 1 {
		recv.log.Info("ASG size matching only works on regions where a single ELB is defined")
		success = true
		return
	}

	elbName := recv.region.ELBs[0].Name

	log := recv.log.New("LoadBalancerName", elbName)
	log.Info("Getting ELB tags")

	elbClient := elb.New(recv.roleSession)
	cfnClient := cloudformation.New(recv.roleSession)

	var tagDescriptions []*elb.TagDescription

	retryMsg := func(i int) { log.Warn("elb:DescribeTags retrying", "Count", i) }
	if !util.SuccessRetryer(5, retryMsg, func() bool {

		describeTagsInput := &elb.DescribeTagsInput{
			LoadBalancerNames: []*string{aws.String(elbName)},
		}

		log.Info("elb:DescribeTags")
		describeTagsOutput, err := elbClient.DescribeTags(describeTagsInput)
		if err != nil {
			log.Error("elb:DescribeTags", "Error", err)
			return false
		}

		tagDescriptions = describeTagsOutput.TagDescriptions

		return true
	}) {
		log.Crit("Failed to elb:DescribeTags")
		return
	}

	var stackId *string
	var err error

tagLoop:
	for _, tagDescription := range tagDescriptions {

		for _, tag := range tagDescription.Tags {
			if tag.Key == nil || *tag.Key != constants.PorterStackIdTag {
				continue
			}

			log.Info("Found ELB tag of currently promoted stack. Getting ASG physical id")

			stackId = new(string)
			*stackId = *tag.Value

			break tagLoop
		}
	}

	if stackId == nil || *stackId == "" {
		log.Warn("Did not find ELB tag of currently promoted stack")
		log.Warn("If this is the first deployment into this ELB then this is normal")
		log.Warn("Otherwise you should investigate why the ELB is missing the tag " + constants.PorterStackIdTag)
		log.Warn("ASG size matching will not occur meaning whatever is in the CloudFormation template will be used")
		success = true
		return
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: stackId,
	}
	var describeStacksOutput *cloudformation.DescribeStacksOutput

	retryMsg = func(i int) { log.Warn("cloudformation:DescribeStacks retrying", "Count", i) }
	if !util.SuccessRetryer(5, retryMsg, func() bool {

		log.Info("cloudformation:DescribeStacks")
		describeStacksOutput, err = cfnClient.DescribeStacks(describeStacksInput)
		if err != nil {
			log.Error("cloudformation:DescribeStacks", "Error", err)
			return false
		}

		return true
	}) {
		log.Crit("Failed to cloudformation:DescribeStacks")
		return
	}

	if len(describeStacksOutput.Stacks) != 1 {
		log.Error(fmt.Sprintf("Found %d stacks", len(describeStacksOutput.Stacks)))
		return
	}

	stackStatus := *describeStacksOutput.Stacks[0].StackStatus

	switch stackStatus {
	case cfn.CREATE_COMPLETE,
		cfn.UPDATE_COMPLETE,
		cfn.UPDATE_ROLLBACK_COMPLETE,
		cfn.UPDATE_ROLLBACK_FAILED:

	// error cases that should cause a failure
	case cfn.UPDATE_COMPLETE_CLEANUP_IN_PROGRESS,
		cfn.UPDATE_IN_PROGRESS,
		cfn.UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS,
		cfn.UPDATE_ROLLBACK_IN_PROGRESS:

		log.Error("Stack is currently updating so ASG size can not be determined safely",
			"StackStatus", stackStatus)
		return

	// error cases that should NOT cause a failure
	default:

		log.Warn("Stack is not in a state that ASG size can be used",
			"StackStatus", stackStatus)
		log.Warn("ASG size matching will not occur meaning whatever is in the CloudFormation template will be used")
		success = true
		return
	}

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: stackId,
	}
	var describeStackResourcesOutput *cloudformation.DescribeStackResourcesOutput

	retryMsg = func(i int) { log.Warn("cloudformation:DescribeStackResources retrying", "Count", i) }
	if !util.SuccessRetryer(5, retryMsg, func() bool {

		log.Info("cloudformation:DescribeStackResources")
		describeStackResourcesOutput, err = cfnClient.DescribeStackResources(describeStackResourcesInput)
		if err != nil {
			log.Error("cloudformation:DescribeStackResources", "Error", err)
			return false
		}

		for _, stackResource := range describeStackResourcesOutput.StackResources {

			if *stackResource.ResourceType == cfn.AutoScaling_AutoScalingGroup {

				*asgId = *stackResource.PhysicalResourceId
				break
			}
		}

		return true
	}) {
		log.Crit("Failed to cloudformation:DescribeStackResources")
		return
	}

	if *asgId == "" {
		log.Warn("Found the ELB tag of a previously promoted stack but that stack appears to be gone")
		log.Warn("This should only happen if the stack was intentionally deleted")
		log.Warn("This is an unexpected case but is not an error")
		log.Warn("ASG size matching will not occur meaning whatever is in the CloudFormation template will be used")
		success = true
		return
	}

	success = true
	return
}

func (recv *stackCreator) getAsgSize(asgName string, regionState *provision_state.Region) (success bool) {

	var (
		err                             error
		describeAutoScalingGroupsOutput *autoscaling.DescribeAutoScalingGroupsOutput
	)

	asgClient := autoscaling.New(recv.roleSession)

	describeAutoScalingGroupsInput := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}

	log := recv.log.New("AutoScalingPhysicalResourceId", asgName)
	log.Info("Getting currently promoted stack's ASG size")

	retryMsg := func(i int) { log.Warn("autoscaling:DescribeAutoScalingGroups retrying", "Count", i) }
	if !util.SuccessRetryer(5, retryMsg, func() bool {

		log.Info("autoscaling:DescribeAutoScalingGroups")
		describeAutoScalingGroupsOutput, err = asgClient.DescribeAutoScalingGroups(describeAutoScalingGroupsInput)
		if err != nil {
			log.Error("autoscaling:DescribeAutoScalingGroups", "Error", err)
			return false
		}
		if len(describeAutoScalingGroupsOutput.AutoScalingGroups) != 1 {
			log.Error("autoscaling:DescribeAutoScalingGroups did not return a ASG")
			return false
		}

		asg := describeAutoScalingGroupsOutput.AutoScalingGroups[0]
		asgMin := int(*asg.MinSize)
		asgDesired := int(*asg.DesiredCapacity)
		asgMax := int(*asg.MaxSize)

		log.Info("Will match currently promoted stack's ASG size to preserve scaling events that have occurred",
			"MinSize", asgMin,
			"MazSize", asgMax,
			"DesiredCapacity", asgDesired)

		regionState.AsgMin = asgMin
		regionState.AsgDesired = asgDesired
		regionState.AsgMax = asgMax

		recv.asgMin = asgMin
		recv.asgDesired = asgDesired
		recv.asgMax = asgMax

		return true
	}) {
		log.Crit("Failed to autoscaling:DescribeAutoScalingGroups")
		return
	}

	success = true
	return
}

func (recv *stackCreator) uploadServicePayload() (checksum string, success bool) {

	payloadBytes, err := ioutil.ReadFile(constants.PayloadPath)
	if err != nil {
		recv.log.Error("ReadFile payload", "Error", err)
		return
	}

	s3Client := s3.New(recv.roleSession)

	// TODO don't use a digest that requires everything to be in memory
	checksumArray := sha256.Sum256(payloadBytes)
	checksum = hex.EncodeToString(checksumArray[:])
	recv.servicePayloadChecksum = checksum
	recv.servicePayloadKey = fmt.Sprintf("%s/%s.tar", recv.s3KeyRoot(s3KeyOptDeployment), checksum)

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(recv.region.S3Bucket),
		Key:    aws.String(recv.servicePayloadKey),
	}

	headObjectOutput, err := s3Client.HeadObject(headObjectInput)
	if err == nil {
		if headObjectOutput.ContentLength != nil && *headObjectOutput.ContentLength > 0 {
			recv.log.Info("Service payload exists", "S3key", recv.servicePayloadKey)
			success = true
			return
		}
	} else if !strings.Contains(err.Error(), "404") {
		recv.log.Error("HeadObject", "Error", err)
		if strings.Contains(err.Error(), "403") {
			recv.log.Error("s3:GetObject and s3:ListBucket are needed for this operation to work")
		}
		return
	}

	uploadInput := &s3manager.UploadInput{
		Bucket:          aws.String(recv.region.S3Bucket),
		Key:             aws.String(recv.servicePayloadKey),
		Body:            bytes.NewReader(payloadBytes),
		ContentType:     aws.String("application/x-tar"),
		ContentEncoding: aws.String("gzip"),
		StorageClass:    aws.String("STANDARD_IA"),
	}

	if recv.region.SSEKMSKeyId != nil {
		uploadInput.SSEKMSKeyId = recv.region.SSEKMSKeyId
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading service payload",
		"S3key", recv.servicePayloadKey,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(uploadInput)
	if err != nil {
		recv.log.Error("Upload failure", "Error", err)
		return
	}

	success = true
	return
}

func (recv *stackCreator) createStack() (stackId string, success bool) {

	client := cloudformation.New(recv.roleSession)

	templateBytes, creationSuccess := recv.createTemplate()
	if !creationSuccess {
		return
	}

	err := ioutil.WriteFile(constants.CloudFormationTemplatePath, templateBytes, 0644)
	if err != nil {
		errorMessage := fmt.Sprintf("Unable to write %s file", constants.CloudFormationTemplatePath)
		recv.log.Error(errorMessage, "Error", err)
		return
	}

	checksumArray := sha256.Sum256(templateBytes)
	checksum := hex.EncodeToString(checksumArray[:])
	templateS3Key := fmt.Sprintf("%s/%s", recv.s3KeyRoot(s3KeyOptTemplate), checksum)

	uploadInput := &s3manager.UploadInput{
		Bucket:      aws.String(recv.region.S3Bucket),
		Key:         aws.String(templateS3Key),
		Body:        bytes.NewReader(templateBytes),
		ContentType: aws.String("application/json"),
	}

	if recv.region.SSEKMSKeyId != nil {
		uploadInput.SSEKMSKeyId = recv.region.SSEKMSKeyId
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading CloudFormation template",
		"S3bucket", recv.region.S3Bucket,
		"S3key", templateS3Key,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(uploadInput)
	if err != nil {
		recv.log.Error("Upload failure", "Error", err)
		return
	}

	templateUrl := fmt.Sprintf("https://s3.amazonaws.com/%s/%s",
		recv.region.S3Bucket, templateS3Key)

	params := CfnApiInput{
		Environment: recv.environment.Name,
		Region:      recv.region.Name,
		SecretsKey:  recv.secretsKey,
		SecretsLoc:  recv.secretsLocation,
		TemplateUrl: templateUrl,
	}

	stackId, success = recv.cfnAPI(client, params)
	return
}

func (recv *stackCreator) createTemplate() (templateBytes []byte, success bool) {

	var err error
	template := cfn.NewTemplate()

	stackDefinitionPath, err := recv.environment.GetStackDefinitionPath(recv.region.Name)
	if err != nil {
		recv.log.Error("GetStackDefinitionPath", "Error", err)
		return
	}

	if stackDefinitionPath != "" {
		recv.log.Info("Using custom stack definition", "Path", stackDefinitionPath)

		stackFile, err := os.Open(stackDefinitionPath)
		if err != nil {
			recv.log.Error("os.Open",
				"Path", stackDefinitionPath,
				"Error", err)
			return
		}

		err = json.NewDecoder(stackFile).Decode(template)
		if err != nil {
			recv.log.Error("json.Decode",
				"Path", stackDefinitionPath,
				"Error", err)
			return
		}
	}

	template.ParseResources()

	if !recv.mutateTemplate(template) {
		return
	}

	// serialize expanded template
	templateBytes, err = json.Marshal(template)
	if err != nil {
		recv.log.Error("json.Marshal", "Error", err)
		return
	}

	success = true
	return
}

func (recv *stackCreator) mutateTemplate(template *cfn.Template) (success bool) {

	template.Description = fmt.Sprintf("%s (powered by porter %s)", recv.config.ServiceName, constants.Version)

	success = recv.ensureResources(template)
	if !success {
		return
	}

	success = recv.mapResources(template)
	if !success {
		return
	}

	success = true
	return
}

func (recv *stackCreator) s3KeyRoot(prefixOpt int) string {
	var prefix string
	if prefixOpt&s3KeyOptTemplate == s3KeyOptTemplate {
		prefix = "porter-template"
	} else if prefixOpt&s3KeyOptDeployment == s3KeyOptDeployment {
		prefix = "porter-deployment"
	} else {
		panic(fmt.Errorf("invalid option %d", prefixOpt))
	}

	return fmt.Sprintf("%s/%s/%s/%s",
		prefix, recv.config.ServiceName, recv.environment.Name, recv.config.ServiceVersion)
}
