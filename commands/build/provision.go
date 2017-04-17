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
package build

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	awsutil "github.com/adobe-platform/porter/aws/util"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision"
	"github.com/adobe-platform/porter/provision_state"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/phylake/go-cli"
	"gopkg.in/inconshreveable/log15.v2"
)

var sleepDuration = constants.StackCreationPollInterval()

type (
	ProvisionStackCmd struct{}

	hotswapStruct struct {
		shouldHotswap bool
		stackStatus   string
		stackId       string
		stackName     string
		region        string
	}
)

func (recv *ProvisionStackCmd) Name() string {
	return "provision"
}

func (recv *ProvisionStackCmd) ShortHelp() string {
	return "Provision a new stack"
}

func (recv *ProvisionStackCmd) LongHelp() string {
	return `NAME
    provision -- Provision a new stack

SYNOPSIS
    provision -e <environment out of .porter/config>

DESCRIPTION
    Provision a new stack for a given environment.

    This command is similar to create-stack but it works with multiple regions
    and should be run from a build box.`
}

func (recv *ProvisionStackCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *ProvisionStackCmd) Execute(args []string) bool {

	if len(args) > 0 {
		var environment string
		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&environment, "e", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		if !ProvisionOrHotswapStack(environment) {
			os.Exit(1)
		}
		return true
	}

	return false
}

func ProvisionOrHotswapStack(env string) (success bool) {
	log := logger.CLI("cmd", "provision")

	config, getAlteredConfigSuccess := conf.GetAlteredConfig(log)
	if !getAlteredConfigSuccess {
		return
	}

	environment, err := config.GetEnvironment(env)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		return
	}

	err = environment.IsWithinBlackoutWindow()
	if err != nil {
		log.Error("Blackout window is active", "Error", err, "Environment", environment.Name)
		return
	}

	_, err = os.Stat(constants.PayloadPath)
	if err != nil {
		log.Error("Service payload not found", "ServicePayloadPath", constants.PayloadPath, "Error", err)
		return
	}

	if environment.Hotswap {

		log.Info("Hot swap enabled")

		hotswapStructs := make([]hotswapStruct, 0)
		hotswapChan := make(chan hotswapStruct)
		failureChan := make(chan struct{})

		for _, region := range environment.Regions {

			go func(environment *conf.Environment, region *conf.Region) {

				if hotswapData, ok := checkShouldHotswapRegion(log, config, environment, region); ok {

					hotswapChan <- hotswapData
				} else {

					failureChan <- struct{}{}
				}

			}(environment, region)
		}

		// provision if any region's stack is too old
		shouldHotswap := true

		for i := 0; i < len(environment.Regions); i++ {
			select {
			case hotswapData := <-hotswapChan:

				switch hotswapData.stackStatus {
				case cfn.CREATE_COMPLETE, cfn.UPDATE_COMPLETE:
					// only states eligible for hot swap

				case cfn.UPDATE_COMPLETE_CLEANUP_IN_PROGRESS,
					cfn.UPDATE_IN_PROGRESS,
					cfn.UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS,
					cfn.UPDATE_ROLLBACK_IN_PROGRESS:

					log.Error("A previous hot swap is still in progress",
						"StackStatus", hotswapData.stackStatus)
					return

				default:

					hotswapData.shouldHotswap = false
				}

				shouldHotswap = shouldHotswap && hotswapData.shouldHotswap
				hotswapStructs = append(hotswapStructs, hotswapData)

			case _ = <-failureChan:
				return
			}
		}

		if shouldHotswap {
			success = HotswapStack(log, config, environment, hotswapStructs)
		} else {
			success = ProvisionStack(log, config, environment)
		}

	} else {

		success = ProvisionStack(log, config, environment)
	}

	return
}

func checkShouldHotswapRegion(log log15.Logger, config *conf.Config,
	environment *conf.Environment, region *conf.Region) (hotswapData hotswapStruct, success bool) {

	log = log.New("Region", region.Name)

	log.Debug("checkShouldHotswapRegion BEGIN")
	defer log.Debug("checkShouldHotswapRegion END")
	log.Info("Checking if this region is eligible for hot swap")

	hotswapData.region = region.Name

	if len(region.ELBs) > 1 {
		success = true
		return
	}

	// set true until we find out otherwise
	hotswapData.shouldHotswap = true

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, 0)

	asgClient := autoscaling.New(roleSession)
	elbClient := elb.New(roleSession)
	cfnClient := cloudformation.New(roleSession)

	var stackId *string

	if region.HasELB() {

		elbName := region.ELBs[0].Name

		log = log.New("LoadBalancerName", elbName)

		var tagDescriptions []*elb.TagDescription

		retryMsg := func(i int) { log.Warn("elb:DescribeTags retrying", "Count", i) }
		if !util.SuccessRetryer(3, retryMsg, func() bool {

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

		var foundStackId, foundPorterVersion bool

	tagLoop:
		for _, tagDescription := range tagDescriptions {

			for _, tag := range tagDescription.Tags {
				if tag.Key == nil {
					continue
				}

				switch *tag.Key {
				case constants.PorterStackIdTag:
					foundStackId = true

					log.Info("Found ELB tag of currently promoted stack. Getting stack info")

					stackId = tag.Value

				case constants.PorterVersionTag:
					foundPorterVersion = true

					if *tag.Value != constants.Version {
						log.Info("porter version mismatch will skip hot swap",
							"porter_deployed_version", *tag.Value)

						hotswapData.shouldHotswap = false
						success = true
						return
					}
				}

				if foundStackId && foundPorterVersion {
					break tagLoop
				}
			}
		}

		if !foundStackId || !foundPorterVersion {
			log.Info("ELB missing required tags to determine hot swap eligibility")
			hotswapData.shouldHotswap = false
			success = true
			return
		}
	}

	// stackId may be null in which case we're going to retrieve all stacks and
	// use the newest one to hot swap. it's important to support (1) the use of
	// an ELB with the possibility of older stacks being promoted
	// (active/passive setup), and (2) no ELB in which case all we can do is use
	// the most recent stack.
	stackList, getStacksSuccess := awsutil.GetStacks(log, config, environment,
		cfnClient, stackId, cfn.CheckHotswapStatus)
	if !getStacksSuccess {
		return
	}

	if len(stackList) == 0 {
		hotswapData.shouldHotswap = false
		success = true
		return
	}

	stack := stackList[0]

	log = log.New("StackName", *stack.StackName)

	hotswapData.stackStatus = *stack.StackStatus
	hotswapData.stackName = *stack.StackName
	hotswapData.stackId = *stack.StackId
	log.Info("DescribeStacks output", "StackId", hotswapData.stackId)

	creationTime := *stack.CreationTime
	hotswapCutoffTime := creationTime.Add(constants.InfrastructureTTL)
	now := time.Now()

	log.Info("Times",
		"CreationTime", creationTime.Format(time.UnixDate),
		"HotswapCutoffTime", hotswapCutoffTime.Format(time.UnixDate),
		"Now", now.Format(time.UnixDate))

	if now.After(hotswapCutoffTime) {

		log.Info("Region is NOT eligible for hot swap. Cutoff time exceeded")
		hotswapData.shouldHotswap = false
		success = true
		return
	} else {

		log.Info("Region is eligible for hot swap")
	}

	// in the case we don't have an ELB then we need to get the stack's ASG
	// which was tagged w/ porter version to do the last determination of
	// hotswap eligibility
	if !region.HasELB() {

		describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
			StackName: stack.StackName,
		}
		var describeStackResourcesOutput *cloudformation.DescribeStackResourcesOutput

		retryMsg := func(i int) { log.Warn("cloudformation:DescribeStackResources retrying", "Count", i) }
		if !util.SuccessRetryer(3, retryMsg, func() bool {

			log.Info("cloudformation:DescribeStackResources")
			describeStackResourcesOutput, err = cfnClient.DescribeStackResources(describeStackResourcesInput)
			if err != nil {
				log.Error("cloudformation:DescribeStackResources", "Error", err)
				return false
			}

			return true
		}) {
			log.Error("Failed to cloudformation:DescribeStackResources")
			return
		}

		var asgName *string
		for _, resource := range describeStackResourcesOutput.StackResources {
			if *resource.ResourceType == cfn.AutoScaling_AutoScalingGroup {
				asgName = resource.PhysicalResourceId
				break
			}
		}

		if asgName == nil {
			log.Error("could not find a " + cfn.AutoScaling_AutoScalingGroup + " in the stack")
			return
		}

		log = log.New("ASGName", *asgName)

		var describeTagsOutput *autoscaling.DescribeTagsOutput

		retryMsg = func(i int) { log.Warn("autoscaling:DescribeTags retrying", "Count", i) }
		if !util.SuccessRetryer(3, retryMsg, func() bool {

			describeTagsInput := &autoscaling.DescribeTagsInput{
				Filters: []*autoscaling.Filter{
					{
						Name:   aws.String("auto-scaling-group"),
						Values: []*string{asgName},
					},
				},
			}

			log.Info("autoscaling:DescribeTags")
			describeTagsOutput, err = asgClient.DescribeTags(describeTagsInput)
			if err != nil {
				log.Error("autoscaling:DescribeTags", "Error", err)
				return false
			}

			return true
		}) {
			log.Crit("Failed to autoscaling:DescribeTags")
			return
		}

		var foundPorterVersion bool

		for _, tag := range describeTagsOutput.Tags {
			if tag.Key == nil {
				continue
			}

			log.Debug("Tag", "Key", *tag.Key, "Value", *tag.Value)

			if *tag.Key == constants.PorterVersionTag {
				foundPorterVersion = true

				if *tag.Value != constants.Version {
					log.Info("porter version mismatch will skip hot swap",
						"porter_deployed_version", *tag.Value)

					hotswapData.shouldHotswap = false
					success = true
					return
				}
				break
			}
		}

		if !foundPorterVersion {
			log.Info("ASG missing required tags to determine hot swap eligibility")
			hotswapData.shouldHotswap = false
			success = true
			return
		}
	}

	success = true
	return
}

func HotswapStack(log log15.Logger, config *conf.Config,
	environment *conf.Environment, hotswapStructs []hotswapStruct) (success bool) {

	var stackName string
	stackRegions := make(map[string]*provision_state.Region)

	for _, hotswapStruct := range hotswapStructs {
		if stackName == "" {
			stackName = hotswapStruct.stackName
		}

		log.Debug("hotswapStruct",
			"shouldHotswap", hotswapStruct.shouldHotswap,
			"stackId", hotswapStruct.stackId,
			"stackName", hotswapStruct.stackName,
			"region", hotswapStruct.region,
		)

		if stackName != hotswapStruct.stackName {
			log.Error("invariant violation: mismatching stack names",
				"current", stackName, "next", hotswapStruct.stackName)
			return
		}

		stackRegions[hotswapStruct.region] = &provision_state.Region{
			StackId: hotswapStruct.stackId,
		}
	}

	stack := provision_state.Stack{
		Environment: environment.Name,
		Hotswap:     true,
		Name:        stackName,
		Regions:     stackRegions,
	}

	defer func() {

		log.Debug("defer post-hook execute")

		postHookSuccess := hook.Execute(log, constants.HookPostHotswap,
			environment.Name, stack.Regions, success)

		success = success && postHookSuccess
	}()

	if !hook.Execute(log, constants.HookPreHotswap, environment.Name, nil, true) {
		return
	}

	if !provision.UpdateStack(log, config, stack) {
		return
	}

	successChan := make(chan bool)

	for regionName, regionState := range stack.Regions {

		go func(environment *conf.Environment, regionName string, regionState *provision_state.Region) {

			successChan <- hotswapStackPoll(log, environment, regionName, regionState)

		}(environment, regionName, regionState)
	}

	success = true

	for i := 0; i < len(environment.Regions); i++ {
		regionSuccess := <-successChan
		success = success && regionSuccess
	}

	if success {
		success = writeProvisionOutput(log, stack)
	}

	if success {
		log.Info("Hot swap complete")
	} else {
		log.Info("Hot swap failed")
	}

	return
}

func hotswapStackPoll(log log15.Logger, environment *conf.Environment,
	regionName string, regionState *provision_state.Region) (success bool) {

	log.Info("Polling for hotswap completion")

	var (
		receiveMessageOutput *sqs.ReceiveMessageOutput

		queueUrl string
	)

	roleARN, err := environment.GetRoleARN(regionName)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(regionName, roleARN, 0)
	sqsClient := sqs.New(roleSession)

	if !getQueueUrl(log, roleSession, regionState.StackId, &queueUrl) {
		return
	}

	// from cloudformation:UpdateStack there are a few timings that inform this
	// loop
	// 1. ~ 1min: cfn-hup polls every 60 seconds to detect the stack update and
	//            call porter_hotswap
	// 2. variable: time to download and start the service
	// 3. HC_HealthyThreshold * HC_Interval seconds: health check on each container
	// 4. ~ 1min: to complete haproxy reload which is the Keep-Alive time from ELB
	// 5. 1min: timeout on hot swap signal
	//
	// That's 3m 15s excluding step 2
	// Allow 10 mins for step 2
	receiveSuccess := 0
	loopCount := 0
	for regionState.AsgDesired != receiveSuccess {

		if loopCount == 40 {
			log.Error("Never received messages from all EC2 instances")
			return
		}

		receiveMessageInput := &sqs.ReceiveMessageInput{
			QueueUrl:        aws.String(queueUrl),
			WaitTimeSeconds: aws.Int64(20),
		}

		log.Info("sqs:ReceiveMessage")
		retryMsg := func(i int) { log.Warn("sqs:ReceiveMessage retrying", "Count", i) }
		if !util.SuccessRetryer(3, retryMsg, func() bool {
			receiveMessageOutput, err = sqsClient.ReceiveMessage(receiveMessageInput)
			if err != nil {
				log.Error("sqs:ReceiveMessage", "Error", err)
				return false
			}
			return true
		}) {
			return
		}

		for _, message := range receiveMessageOutput.Messages {

			if message.Body != nil && *message.Body == "success" {
				log.Info("Received success")
				receiveSuccess++
			} else {
				log.Error("A EC2 instance reported an error during hot swap")
				log.Error("If deploying to multiple regions it's possible they completed and this one failed")
				return
			}
		}

		loopCount++
	}

	log.Info("All EC2 instances in this region reported hot swap success")

	success = true
	return
}

func getQueueUrl(log log15.Logger, roleSession *session.Session,
	stackId string, queueUrl *string) (success bool) {

	var (
		describeStackResourcesOutput *cloudformation.DescribeStackResourcesOutput
		err                          error
	)

	log.Debug("getQueueUrl", "stackId", stackId)

	cfnClient := cloudformation.New(roleSession)

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackId),
	}

	log.Info(fmt.Sprintf("Getting ASG logical name and %s URL", constants.SignalQueue))
	log.Info("cloudformation:DescribeStackResources")
	retryMsg := func(i int) { log.Warn("cloudformation:DescribeStackResources retrying", "Count", i) }
	if !util.SuccessRetryer(3, retryMsg, func() bool {

		describeStackResourcesOutput, err = cfnClient.DescribeStackResources(describeStackResourcesInput)
		if err != nil {
			log.Error("cloudformation:DescribeStackResources", "Error", err)
			return false
		}

		for _, stackResource := range describeStackResourcesOutput.StackResources {

			if *stackResource.ResourceType == cfn.SQS_Queue {

				*queueUrl = *stackResource.PhysicalResourceId
				break
			}
		}

		return true
	}) {
		log.Crit("Failed to cloudformation:DescribeStackResources")
		return
	}

	log.Debug("getQueueUrl", "queueUrl", *queueUrl)

	success = true
	return
}

func ProvisionStack(log log15.Logger, config *conf.Config, environment *conf.Environment) (success bool) {

	stack := &provision_state.Stack{
		Environment: environment.Name,
	}

	defer func() {

		log.Debug("defer post-hook execute")

		postHookSuccess := hook.Execute(log, constants.HookPostProvision,
			environment.Name, stack.Regions, success)

		success = success && postHookSuccess
	}()

	if !hook.Execute(log, constants.HookPreProvision, environment.Name, nil, true) {
		return
	}

	if !provision.CreateStack(log, config, stack) {
		return
	}

	successChan := make(chan bool)

	for regionName, regionState := range stack.Regions {

		go func(environment *conf.Environment, regionName string, regionState *provision_state.Region) {

			successChan <- provisionStackPoll(log, environment, regionName, regionState)

		}(environment, regionName, regionState)
	}

	success = true

	for i := 0; i < len(environment.Regions); i++ {
		regionSuccess := <-successChan
		success = success && regionSuccess
	}

	if success {

		success = writeProvisionOutput(log, *stack)

	} else {

		if len(stack.Regions) > 0 {
			log.Warn("Some regions failed to create. Deleting the successful ones")

			for regionName, regionState := range stack.Regions {
				roleARN, err := environment.GetRoleARN(regionName)
				if err != nil {
					log.Error("GetRoleARN", "Error", err)
					continue
				}

				roleSession := aws_session.STS(regionName, roleARN, 0)
				cfnClient := cloudformation.New(roleSession)

				log.Info("cloudformation:DeleteStack", "StackId", regionState.StackId)
				deleteStackInput := &cloudformation.DeleteStackInput{
					StackName: aws.String(regionState.StackId),
				}

				cfnClient.DeleteStack(deleteStackInput)
			}
		}
	}

	return
}

func provisionStackPoll(log log15.Logger, environment *conf.Environment,
	regionName string, regionState *provision_state.Region) (success bool) {

	var (
		stackProvisioned            bool
		elbLogicalId                string
		describeStackResourceOutput *cloudformation.DescribeStackResourceOutput
	)

	log = log.New("Region", regionName)

	region, err := environment.GetRegion(regionName)
	if err != nil {
		log.Error("GetRegion", "Error", err)
		return
	}

	roleARN, err := environment.GetRoleARN(region.Name)
	if err != nil {
		log.Error("GetRoleARN", "Error", err)
		return
	}

	roleSession := aws_session.STS(region.Name, roleARN, constants.StackCreationTimeout())
	cfnClient := cloudformation.New(roleSession)

	n := int(constants.StackCreationTimeout().Seconds() / sleepDuration.Seconds())

stackEventPoll:
	for i := 0; i < n; i++ {

		describeStacksInput := &cloudformation.DescribeStacksInput{
			StackName: aws.String(regionState.StackId),
		}

		describeStackOutput, err := cfnClient.DescribeStacks(describeStacksInput)
		if err != nil {
			log.Error("cloudformation:DescribeStack", "Error", err)
			return
		}
		if describeStackOutput == nil || len(describeStackOutput.Stacks) != 1 {
			log.Error("cloudformation:DescribeStack unexpected output")
			return
		}

		log.Info("Stack status", "StackStatus", *describeStackOutput.Stacks[0].StackStatus)

		switch *describeStackOutput.Stacks[0].StackStatus {
		case cfn.CREATE_COMPLETE:
			stackProvisioned = true
			break stackEventPoll
		case cfn.CREATE_FAILED:
			log.Error("Stack creation failed")
			return
		case cfn.DELETE_IN_PROGRESS:
			log.Error("Stack is being deleted")
			return
		case cfn.ROLLBACK_IN_PROGRESS:
			log.Error("Stack is rolling back")
			return
		}

		time.Sleep(sleepDuration)
	}

	if !stackProvisioned {
		log.Error("stack provision timeout")
		return
	}

	cfnTemplateBytes, err := ioutil.ReadFile(constants.CloudFormationTemplatePath)
	if err != nil {
		log.Error("CloudFormationTemplate read file error", "Error", err)
		return
	}

	cfnTemplate := cfn.NewTemplate()

	err = json.Unmarshal(cfnTemplateBytes, &cfnTemplate)
	if err != nil {
		log.Error("json.Unmarshal", "Error", err)
		return
	}

	cfnTemplate.ParseResources()

	if region.PrimaryTopology() == conf.Topology_Inet && region.HasELB() {

		elbLogicalId, err = cfnTemplate.GetResourceName(cfn.ElasticLoadBalancing_LoadBalancer)
		if err != nil {
			log.Error("GetResourceName", "Error", err)
			return
		}

		describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
			LogicalResourceId: aws.String(elbLogicalId),
			StackName:         aws.String(regionState.StackId),
		}

		//Once stack provisioned get the provisoned elb
		retryMsg := func(i int) { log.Warn("cloudformation:DescribeStackResource retrying", "Count", i) }
		if !util.SuccessRetryer(3, retryMsg, func() bool {
			describeStackResourceOutput, err = cfnClient.DescribeStackResource(describeStackResourceInput)
			if err != nil {
				log.Error("cloudformation:DescribeStackResource", "Error", err)
				return false
			}
			return true
		}) {
			return
		}
	}

	if describeStackResourceOutput != nil {
		regionState.ProvisionedELBName =
			*describeStackResourceOutput.StackResourceDetail.PhysicalResourceId
	}

	success = true
	return
}

func writeProvisionOutput(log log15.Logger, stack provision_state.Stack) (success bool) {

	provisionBytes, err := json.Marshal(stack)
	if err != nil {
		log.Error("json.Marshal", "Error", err)
		return
	}

	// write the stackoutput into porter tmp directory
	err = ioutil.WriteFile(constants.ProvisionOutputPath, provisionBytes, 0644)
	if err != nil {
		log.Error("Unable to write provision output", "Error", err)
		return
	}

	success = true
	return
}
