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
package ec2

import (
	"fmt"
	"time"

	"github.com/adobe-platform/porter/aws/util"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2lib "github.com/aws/aws-sdk-go/service/ec2"
)

// Don't force clients of this package to import
// "github.com/aws/aws-sdk-go/service/ec2"
func New(config *session.Session) *ec2lib.EC2 {
	return ec2lib.New(config)
}

// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_AllocateAddress.html
func AllocateAddress(client *ec2lib.EC2, domain string) string {
	if domain == "" {
		domain = "vpc"
	}

	input := &ec2lib.AllocateAddressInput{
		Domain: &domain,
	}

	output, err := client.AllocateAddress(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.AllocationId

	return resourceId
}

func AssociateAddress(client *ec2lib.EC2, allocationId, networkInterfaceId string) {

	input := &ec2lib.AssociateAddressInput{
		AllocationId:       &allocationId,
		NetworkInterfaceId: &networkInterfaceId,
	}

	_, err := client.AssociateAddress(input)
	if err != nil {
		panic(err)
	}
}

func AssociateRouteTable(client *ec2lib.EC2, routeTableId, subnetId string) {

	input := &ec2lib.AssociateRouteTableInput{
		RouteTableId: &routeTableId,
		SubnetId:     &subnetId,
	}

	_, err := client.AssociateRouteTable(input)
	if err != nil {
		panic(err)
	}
}

func AttachInternetGateway(client *ec2lib.EC2, vpcId, gatewayId string) {

	input := &ec2lib.AttachInternetGatewayInput{
		VpcId:             &vpcId,
		InternetGatewayId: &gatewayId,
	}

	_, err := client.AttachInternetGateway(input)
	if err != nil {
		panic(err)
	}
}

func CreateInternetGateway(client *ec2lib.EC2, name string) string {

	input := &ec2lib.CreateInternetGatewayInput{}

	output, err := client.CreateInternetGateway(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.InternetGateway.InternetGatewayId

	NameResource(client, resourceId, name)

	return resourceId
}

func CreateNetworkInterface(client *ec2lib.EC2, subnetId, name string) string {

	input := &ec2lib.CreateNetworkInterfaceInput{
		SubnetId: &subnetId,
	}

	output, err := client.CreateNetworkInterface(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.NetworkInterface.NetworkInterfaceId

	NameResource(client, resourceId, name)

	return resourceId
}

func CreateRouteForInternetGateway(client *ec2lib.EC2, routeTableId, gatewayId string) {

	cidrBlock := "0.0.0.0/0"

	input := &ec2lib.CreateRouteInput{
		RouteTableId:         &routeTableId,
		GatewayId:            &gatewayId,
		DestinationCidrBlock: &cidrBlock,
	}

	_, err := client.CreateRoute(input)
	if err != nil {
		panic(err)
	}
}

func CreateRouteForNetworkInterface(client *ec2lib.EC2, routeTableId, networkInterfaceId string) {

	cidrBlock := "0.0.0.0/0"

	input := &ec2lib.CreateRouteInput{
		RouteTableId:         &routeTableId,
		NetworkInterfaceId:   &networkInterfaceId,
		DestinationCidrBlock: &cidrBlock,
	}

	_, err := client.CreateRoute(input)
	if err != nil {
		panic(err)
	}
}

func CreateRouteTable(client *ec2lib.EC2, vpcId, name string) string {

	input := &ec2lib.CreateRouteTableInput{
		VpcId: &vpcId,
	}

	output, err := client.CreateRouteTable(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.RouteTable.RouteTableId

	NameResource(client, resourceId, name)

	return resourceId
}

func CreateSubnet(client *ec2lib.EC2, vpcId, cidrBlock, name string) string {

	input := &ec2lib.CreateSubnetInput{
		CidrBlock: &cidrBlock,
		VpcId:     &vpcId,
	}

	output, err := client.CreateSubnet(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.Subnet.SubnetId

	NameResource(client, resourceId, name)

	return resourceId
}

func CreateVpc(client *ec2lib.EC2, cidrBlock, name string) string {

	input := &ec2lib.CreateVpcInput{
		CidrBlock: &cidrBlock,
	}

	output, err := client.CreateVpc(input)
	if err != nil {
		panic(err)
	}

	resourceId := *output.Vpc.VpcId

	NameResource(client, resourceId, name)

	return resourceId
}

// http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_Scenario2.html
func CreateVpcScenario2(region, tag string, createNATInstance bool) string {
	if !util.ValidRegion(region) {
		panic("invalid region " + region)
	}

	var (
		allocationId        string
		gatewayId           string
		networkInterfaceId  string
		privateRouteTableId string
		publicRouteTableId  string
		publicSubnetId      string
	)

	client := ec2lib.New(aws_session.Get(region))

	vpcId := CreateVpc(client, "10.0.0.0/16", tag)

	vpcFilter := map[string][]string{
		"vpc-id": {vpcId},
	}

	for i := 0; i < 5; i++ {

		// one route table comes with a new VPC.
		// get it and tag it before we create more
		rts := DescribeRouteTables(client, vpcFilter)
		if rts == nil || len(rts) != 1 {
			fmt.Println("waiting for default route table creation...")
			time.Sleep(1)
			continue
		} else {
			privateRouteTableId = *rts[0].RouteTableId
			err := NameResource(client, privateRouteTableId, tag+" - private")
			if err != nil {
				fmt.Println("failed to tag default route table")
			}
			break
		}
	}

	publicSubnetId = CreateSubnet(client, vpcId, "10.0.0.0/24", tag+" - public")
	CreateSubnet(client, vpcId, "10.0.1.0/24", tag+" - private")

	publicRouteTableId = CreateRouteTable(client, vpcId, tag+" - public")
	AssociateRouteTable(client, publicRouteTableId, publicSubnetId)

	gatewayId = CreateInternetGateway(client, tag)
	AttachInternetGateway(client, vpcId, gatewayId)
	CreateRouteForInternetGateway(client, publicRouteTableId, gatewayId)

	allocationId = AllocateAddress(client, "vpc")

	networkInterfaceId = CreateNetworkInterface(client, publicSubnetId, tag)

	AssociateAddress(client, allocationId, networkInterfaceId)
	CreateRouteForNetworkInterface(client, privateRouteTableId, networkInterfaceId)

	if createNATInstance {
		// TODO NAT instance
	}

	return vpcId
}

func DescribeInstances(client *ec2lib.EC2, kvps map[string][]string, instanceIds ...string) ([]*ec2lib.Reservation, error) {

	input := &ec2lib.DescribeInstancesInput{
		Filters: kvpsToFilters(kvps),
	}

	if len(instanceIds) > 0 {
		ids := make([]*string, 0)
		for _, id := range instanceIds {
			ids = append(ids, &id)
		}
		input.InstanceIds = ids
	}

	output, err := client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	return output.Reservations, nil
}

func DescribeRouteTables(client *ec2lib.EC2, kvps map[string][]string, routeTableIds ...string) []*ec2lib.RouteTable {

	input := &ec2lib.DescribeRouteTablesInput{
		Filters: kvpsToFilters(kvps),
	}

	if len(routeTableIds) > 0 {
		ids := make([]*string, 0)
		for _, id := range routeTableIds {
			ids = append(ids, &id)
		}
		input.RouteTableIds = ids
	}

	output, err := client.DescribeRouteTables(input)
	if err != nil {
		panic(err)
	}

	return output.RouteTables
}

func DescribeSubnets(client *ec2lib.EC2, kvps map[string][]string, subnetIds ...string) []*ec2lib.Subnet {

	input := &ec2lib.DescribeSubnetsInput{
		Filters: kvpsToFilters(kvps),
	}

	if len(subnetIds) > 0 {
		ids := make([]*string, 0)
		for _, id := range subnetIds {
			ids = append(ids, &id)
		}
		input.SubnetIds = ids
	}

	output, err := client.DescribeSubnets(input)
	if err != nil {
		panic(err)
	}

	return output.Subnets
}

func NameResource(client *ec2lib.EC2, resourceId, tagValue string) (err error) {

	tagKey := "Name"
	tagValueCopy := tagValue

	porterTag := ec2lib.Tag{
		Key:   &tagKey,
		Value: &tagValueCopy,
	}

	createTagsInput := &ec2lib.CreateTagsInput{
		Resources: []*string{&resourceId},
		Tags:      []*ec2lib.Tag{&porterTag},
	}

	_, err = client.CreateTags(createTagsInput)
	return
}

func kvpsToFilters(kvps map[string][]string) []*ec2lib.Filter {
	filters := make([]*ec2lib.Filter, 0)

	for k, vs := range kvps {

		pvs := make([]*string, 0)

		for _, v := range vs {
			pvs = append(pvs, &v)
		}

		filter := &ec2lib.Filter{
			Name:   &k,
			Values: pvs,
		}

		filters = append(filters, filter)
	}

	return filters
}
