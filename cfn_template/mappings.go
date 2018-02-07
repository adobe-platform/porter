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
package cfn_template

// RegionToAmazonLinuxAMI is a mapping of AWS region to the latest released
// version the Amazon Linux AMI (currently 2015.09.1)
func RegionToAmazonLinuxAMI() map[string]interface{} {
	return regionToAmazonLinuxAMI2015091()
}

// https://aws.amazon.com/amazon-linux-ami/2015.09-release-notes/
func regionToAmazonLinuxAMI2015091() map[string]interface{} {
	m := map[string]interface{}{
		"us-west-2": map[string]interface{}{
			"Key": "ami-f0091d91",
		},
		"us-east-1": map[string]interface{}{
			"Key": "ami-60b6c60a",
		},
		"us-west-1": map[string]interface{}{
			"Key": "ami-d5ea86b5",
		},
		"eu-west-1": map[string]interface{}{
			"Key": "ami-bff32ccc",
		},
		"eu-central-1": map[string]interface{}{
			"Key": "ami-bc5b48d0",
		},
		"ap-southeast-1": map[string]interface{}{
			"Key": "ami-c9b572aa",
		},
		"ap-northeast-1": map[string]interface{}{
			"Key": "ami-383c1956",
		},
		"ap-southeast-2": map[string]interface{}{
			"Key": "ami-48d38c2b",
		},
		"ap-northeast-2": map[string]interface{}{
			"Key": "ami-249b554a",
		},
		"sa-east-1": map[string]interface{}{
			"Key": "ami-6817af04",
		},
	}

	return m
}

// https://aws.amazon.com/amazon-linux-ami/2015.03-release-notes/
func regionToAmazonLinuxAMI2015031() map[string]interface{} {
	m := map[string]interface{}{
		"us-west-2": map[string]interface{}{
			"Key": "ami-d5c5d1e5",
		},
		"us-east-1": map[string]interface{}{
			"Key": "ami-0d4cfd66",
		},
		"us-west-1": map[string]interface{}{
			"Key": "ami-87ea13c3",
		},
		"eu-west-1": map[string]interface{}{
			"Key": "ami-e4d18e93",
		},
		"eu-central-1": map[string]interface{}{
			"Key": "ami-a6b0b7bb",
		},
		"ap-southeast-1": map[string]interface{}{
			"Key": "ami-d44b4286",
		},
		"ap-northeast-1": map[string]interface{}{
			"Key": "ami-1c1b9f1c",
		},
		"ap-southeast-2": map[string]interface{}{
			"Key": "ami-db7b39e1",
		},
		/* This version doesn't exist in the recently released region

		aws ec2 describe-images \
		--filters Name=name,Values=amzn-ami-hvm-2015.03.1.x86_64-gp2 \
		--region ap-northeast-2

		"ap-northeast-2": map[string]interface{}{
			"Key": "ami-249b554a",
		},*/
		"sa-east-1": map[string]interface{}{
			"Key": "ami-55098148",
		},
	}

	return m
}
