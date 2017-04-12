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
	"strings"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/phylake/go-cli"
)

type S3Cmd struct{}

func (recv *S3Cmd) Name() string {
	return "s3"
}

func (recv *S3Cmd) ShortHelp() string {
	return "Create S3 buckets needed by porter"
}

func (recv *S3Cmd) LongHelp() string {
	return `NAME
    s3 -- Create S3 buckets needed by porter

SYNOPSIS
    s3 -prefix <bucket prefix>

DESCRIPTION
    Create S3 buckets for every AWS region which porter places service payloads
    into as part of deployment. The prefix should be related to your group or
    organization. This command will continue attempting to create buckets even
    when bucket creation fails (usually due to a name collision).

OPTIONS
    -prefix
        The bucket prefix. The AWS region will be appended to create a bucket
        in each region. Remember that bucket names must be globally unique
        within all of AWS.

EXAMPLES
    Services are put in each bucket in their own folder so if my
    organization is called 'Some Org' I would call this command like this:

        porter bootstrap s3 -prefix some-org-builds

    See the service payload docs for info on the s3 key structure
    https://github.com/adobe-platform/porter/blob/master/docs/detailed_design/service-payload.md

    Go to https://console.aws.amazon.com/iam/home?#users and create a user with
    the following inline policy:

        {
          "Version": "2012-10-17",
          "Statement": [
            {
              "Effect": "Allow",
              "Action": [
                "s3:CreateBucket"
              ],
              "Resource": [
                "*"
              ]
            }
          ]
        }

    Get the user's credentials and use them in this command (all one line):

        AWS_ACCESS_KEY_ID=abc123 AWS_SECRET_ACCESS_KEY=secret
        porter bootstrap s3 -prefix some-org-builds

    If you get an error with "AccessDenied" it could be because you didn't
    attach the policy or because the bucket couldn't be created.

    Once you're finished you SHOULD delete this user or drop its privileges`
}

func (recv *S3Cmd) SubCommands() []cli.Command {
	return nil
}

func (recv *S3Cmd) Execute(args []string) bool {
	if len(args) > 0 {
		var bucketPrefix string

		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&bucketPrefix, "prefix", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		bootstrapS3(bucketPrefix)
		return true
	}
	return false
}

func bootstrapS3(bucketPrefix string) {
	log := logger.CLI()

	for region := range constants.AwsRegions {
		client := s3.New(session.New(aws.NewConfig().WithRegion(region)))

		bucketName := strings.TrimSuffix(bucketPrefix, "-") + "-" + region

		input := &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		}

		_, err := client.CreateBucket(input)
		if err != nil {
			log.Error("CreateBucket", "Error", err)
		} else {
			log.Info("Created bucket " + bucketName)
		}
	}
}
