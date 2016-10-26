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
package bootstrap

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/phylake/go-cli"
)

const deploymentPolicy = "porter-deployment"

type IamCmd struct{}

func (recv *IamCmd) Name() string {
	return "iam"
}

func (recv *IamCmd) ShortHelp() string {
	return "Create IAM roles and policies needed by porter"
}

func (recv *IamCmd) LongHelp() string {
	return `NAME
    iam -- Create IAM roles and policies needed by porter

SYNOPSIS
    iam -role <service name> -arns <trusted ARN>[,<trusted ARN> ...]

DESCRIPTION
    Create IAM roles needed by porter and other porter bootstrap commands. This
    is the one command that cannot be automated in any fashion. Given an empty
    AWS account porter needs a manually created user in order to create other
    roles which it will assume, which can then create additional infrastructure
    with which porter expects to interact.

OPTIONS
    -role
        The name of the role to create. A good convention is to have the service
        name appended with -deployment to indicate this role is used in
        deployment of the service.
        e.g. For my service foo I specify foo-deployment.

        The porter-deployment policy that's created will be attached to this
        role. As new services are created the policy can be attached to those
        roles as well.

    -arns
        The list of ARNs allowed to assume this role. Minimally this is the role
        ARN of your EC2 instance that runs CD software. Usually it's good to
        add the user ARNs of yourself and members of your team so they can run
        porter commands locally.

EXAMPLES
    Go to https://console.aws.amazon.com/iam/home?#users and create a user with
    the following inline policy (OR simply attach the AdministratorAccess policy
    to the user and forgo the inline policy):

    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": [
            "iam:AttachRolePolicy",
            "iam:CreatePolicy",
            "iam:CreateRole"
          ],
          "Resource": [
            "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:role/porter-*"
          ]
        }
      ]
    }

    Get the user's credentials and use them in this command (all one line):

        AWS_ACCESS_KEY_ID=abc123 AWS_SECRET_ACCESS_KEY=secret
        porter bootstrap iam
        -role foo-deployment
        -arns arn:aws:iam::123456789012:role/build-box

    Once you're finished you SHOULD delete this user or drop its privileges`
}

func (recv *IamCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *IamCmd) Execute(args []string) bool {
	if len(args) > 0 {
		var arnsCSV, roleName string

		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&arnsCSV, "arns", "", "")
		flagSet.StringVar(&roleName, "role", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		if !bootstrapIAM(roleName, strings.Split(arnsCSV, ",")) {
			os.Exit(1)
		}

		return true
	}
	return false
}

const porterDeploymentPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "autoscaling:CreateAutoScalingGroup",
        "autoscaling:CreateLaunchConfiguration",
        "autoscaling:DeleteAutoScalingGroup",
        "autoscaling:DeleteLaunchConfiguration",
        "autoscaling:DescribeAutoScalingGroups",
        "autoscaling:DescribeLaunchConfigurations",
        "autoscaling:DescribeScalingActivities",
        "autoscaling:UpdateAutoScalingGroup",
        "cloudformation:CreateStack",
        "cloudformation:DeleteStack",
        "cloudformation:DescribeStackEvents",
        "cloudformation:DescribeStackResource",
        "cloudformation:DescribeStackResources",
        "cloudformation:DescribeStacks",
        "cloudformation:UpdateStack",
        "ec2:AuthorizeSecurityGroupEgress",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:CreateSecurityGroup",
        "ec2:DeleteSecurityGroup",
        "ec2:DescribeAccountAttributes",
        "ec2:DescribeAvailabilityZones",
        "ec2:DescribeInstances",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeSubnets",
        "ec2:RevokeSecurityGroupEgress",
        "elasticloadbalancing:AddTags",
        "elasticloadbalancing:ConfigureHealthCheck",
        "elasticloadbalancing:CreateLoadBalancer",
        "elasticloadbalancing:DeleteLoadBalancer",
        "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
        "elasticloadbalancing:DescribeInstanceHealth",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTags",
        "elasticloadbalancing:ModifyLoadBalancerAttributes",
        "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
        "elasticloadbalancing:SetLoadBalancerPoliciesOfListener",
        "iam:AddRoleToInstanceProfile",
        "iam:CreateInstanceProfile",
        "iam:CreateRole",
        "iam:DeleteInstanceProfile",
        "iam:DeleteRole",
        "iam:DeleteRolePolicy",
        "iam:PassRole",
        "iam:PutRolePolicy",
        "iam:RemoveRoleFromInstanceProfile",
        "kms:Decrypt",
        "kms:Encrypt",
        "kms:GenerateDataKey",
        "route53:ChangeResourceRecordSets",
        "route53:GetChange",
        "route53:ListHostedZones",
        "route53:ListResourceRecordSets",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:PutObject",
        "sqs:CreateQueue",
        "sqs:DeleteQueue",
        "sqs:GetQueueAttributes",
        "sqs:GetQueueUrl",
        "sqs:ReceiveMessage"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}`

func bootstrapIAM(roleName string, assumeRoleARNs []string) (success bool) {
	log := logger.CLI()

	for i, arn := range assumeRoleARNs {
		assumeRoleARNs[i] = strconv.Quote(strings.TrimSpace(arn))
	}
	trustPolicyContext := TrustPolicyContext{
		AssumeRoleARNs: strings.Join(assumeRoleARNs, ","),
	}

	client := iam.New(session.New(aws.NewConfig()))

	temp, err := template.New("").Parse(string(trustPolicy))
	if err != nil {
		log.Error("text/template Parse", "Error", err)
		return
	}

	var trustPolicyBuf bytes.Buffer

	err = temp.Execute(&trustPolicyBuf, trustPolicyContext)
	if err != nil {
		log.Error("text/template Execute", "Error", err)
		return
	}

	var marker *string
	var policy *iam.Policy

outer:
	for {
		listPoliciesInput := &iam.ListPoliciesInput{
			Marker: marker,
			Scope:  aws.String("Local"),
		}

		listPoliciesOutput, err := client.ListPolicies(listPoliciesInput)
		if err != nil {
			log.Error("ListPolicies", "Error", err)
			return
		}

		marker = listPoliciesOutput.Marker
		for _, maybePolicy := range listPoliciesOutput.Policies {
			if *maybePolicy.PolicyName == deploymentPolicy {
				log.Info(fmt.Sprintf("Found %s policy", deploymentPolicy))
				policy = maybePolicy
				break outer
			}
		}

		if marker == nil {
			break
		}
	}

	if policy == nil {

		createPolicyInput := &iam.CreatePolicyInput{
			PolicyName:     aws.String(deploymentPolicy),
			PolicyDocument: aws.String(porterDeploymentPolicy),
		}

		createPolicyOutput, err := client.CreatePolicy(createPolicyInput)
		if err != nil {
			log.Error("CreatePolicy", "Error", err)
			return
		}

		policy = createPolicyOutput.Policy
		log.Info("Created policy", "PolicyName", deploymentPolicy)
	}

	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicyBuf.String()),
	}

	createRoleOutput, err := client.CreateRole(createRoleInput)
	if err != nil {
		log.Error("CreateRole", "RoleName", roleName, "Error", err)
		return
	}

	roleArn := *createRoleOutput.Role.Arn

	log.Info("Created role", "RoleName", roleName, "ARN", roleArn)

	msg := "Add this ARN to " + constants.ConfigPath + " so porter can deploy your service"
	log.Info(msg, "ARN", roleArn)

	attachRolePolicyInput := &iam.AttachRolePolicyInput{
		PolicyArn: policy.Arn,
		RoleName:  aws.String(roleName),
	}

	_, err = client.AttachRolePolicy(attachRolePolicyInput)
	if err != nil {
		log.Error("AttachRolePolicy", "RoleName", roleName, "Error", err)
		return
	}

	log.Info(fmt.Sprintf("Attached %s policy to %s", deploymentPolicy, roleName))

	success = true
	return
}

type TrustPolicyContext struct {
	AssumeRoleARNs string
}

const trustPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": [{{.AssumeRoleARNs}}]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`
