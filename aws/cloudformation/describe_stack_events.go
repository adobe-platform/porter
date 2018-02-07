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
package cloudformation

import (
	"github.com/aws/aws-sdk-go/aws"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
)

type StackEventByTime []*cfnlib.StackEvent

func (a StackEventByTime) Len() int      { return len(a) }
func (a StackEventByTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a StackEventByTime) Less(i, j int) bool {
	if a[i].Timestamp != nil && a[j].Timestamp != nil {
		ti := *a[i].Timestamp
		tj := *a[j].Timestamp
		return ti.Before(tj)
	}
	return false
}

// StackEventState keeps the state needed for multiple calls to DescribeStackEvents
type StackEventState struct {
	client    *cfnlib.CloudFormation
	stackName *string
	nextToken *string
}

func NewStackEventState(client *cfnlib.CloudFormation, stackName string) *StackEventState {
	return &StackEventState{
		client:    client,
		stackName: aws.String(stackName),
	}
}

func (recv *StackEventState) DescribeStackEvents() ([]*cfnlib.StackEvent, error) {
	input := &cfnlib.DescribeStackEventsInput{
		StackName: recv.stackName,
		NextToken: recv.nextToken,
	}

	output, err := recv.client.DescribeStackEvents(input)
	if err != nil {
		return nil, err
	}

	recv.nextToken = output.NextToken

	return output.StackEvents, nil
}
