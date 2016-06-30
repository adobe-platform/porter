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
package cfn_template

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/files"
)

type (
	UserDataContext struct {
		LogicalId string
	}

	AWSCloudFormationInitCtx struct {
		PorterVersion string
		Environment   string
		Region        string
		EnvFile       string

		ServiceName    string
		ServiceVersion string

		ServicePayloadBucket     string
		ServicePayloadKey        string
		ServicePayloadConfigPath string

		PrimaryContainer    PrimaryContainer
		SecondaryContainers []SecondaryContainer

		PorterBinaryUrl string

		DevMode bool

		EC2BootstrapScript string

		Elbs string
	}

	PrimaryContainer struct {
		Name              string
		InetPort          int
		HealthCheckMethod string
		HealthCheckPath   string
	}

	SecondaryContainer struct { // TODO figure out linking between containers
		Name string
	}
)

// ImageIdInMap works with a mapping like the following to select an AMI id for
// the current region
//
// {
//   "Mappings": {
//     "RegionToAmazonLinuxAMI": {
//       "ap-northeast-1": { "Key": "ami-1c1b9f1c" },
//       "ap-southeast-1": { "Key": "ami-d44b4286" },
//       "ap-southeast-2": { "Key": "ami-db7b39e1" },
//       "eu-central-1":   { "Key": "ami-a6b0b7bb" },
//       "eu-west-1":      { "Key": "ami-e4d18e93" },
//       "sa-east-1":      { "Key": "ami-55098148" },
//       "us-east-1":      { "Key": "ami-0d4cfd66" },
//       "us-west-1":      { "Key": "ami-87ea13c3" },
//       "us-west-2":      { "Key": "ami-d5c5d1e5" }
//     }
//   }
// }
func ImageIdInMap(mapId string) map[string]interface{} {
	m := map[string]interface{}{
		"Fn::FindInMap": []interface{}{
			mapId,
			map[string]string{"Ref": "AWS::Region"},
			"Key",
		},
	}

	return m
}

// UserData is the initial launch script run by an EC2 instance
//
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AmazonLinuxAMIBasics.html
func UserData(autoScalingLaunchConfigurationLogicalId string) (map[string]interface{}, error) {

	tmpl, err := template.New("").Parse(files.CloudInitJson)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	context := UserDataContext{
		LogicalId: autoScalingLaunchConfigurationLogicalId,
	}

	err = tmpl.Execute(&buf, context)
	if err != nil {
		return nil, err
	}

	userData := make(map[string]interface{})
	json.Unmarshal(buf.Bytes(), &userData)

	b64 := map[string]interface{}{
		"Fn::Base64": userData,
	}

	return b64, nil
}

// AWSCloudFormationInit produces the value of the AWS::CloudFormation::Init
// type and should be placed on the Metadata of a AWS::EC2::Instance or
// AWS::AutoScale::LaunchConfiguration
//
// It works with the UserData field of a AWS::AutoScale::LaunchConfiguration
// (or AWS::EC2::Instance) to bootstrap an EC2 instance when it's created
//
// Example use:
//
// {
//   "Resources": {
//     "AutoScalingLaunchConfiguration": {
//       "Type": "AWS::AutoScaling::LaunchConfiguration",
//       "Metadata": {
//         "AWS::CloudFormation::Init": {{ .Resources.AutoScaling.LaunchConfiguration.Metadata.AWSCloudFormationInit }}
//       },
//       "Properties": {
//         "UserData": {{ .Resources.AutoScaling.LaunchConfiguration.UserData }}
//       }
//     }
//   }
// }
func AWSCloudFormationInit(autoScalingLaunchConfigurationLogicalId string, context AWSCloudFormationInitCtx) (map[string]interface{}, error) {
	/*
		[main]
		stack={ "Ref": "AWS::StackId" }
		region={ "Ref": "AWS::Region" }
		interval=1
	*/
	cfnHupConf := map[string]interface{}{
		"content": map[string]interface{}{
			"Fn::Join": []interface{}{
				"\n",
				[]interface{}{
					"[main]",
					map[string]interface{}{
						"Fn::Join": []interface{}{
							"=",
							[]interface{}{
								"stack",
								map[string]string{"Ref": "AWS::StackId"},
							},
						},
					},
					map[string]interface{}{
						"Fn::Join": []interface{}{
							"=",
							[]interface{}{
								"region",
								map[string]string{"Ref": "AWS::Region"},
							},
						},
					},
					"interval=" + strconv.Itoa(constants.CfnHupPollIntervalMinutes),
					"",
				},
			},
		},
		"mode":  "000644",
		"owner": "root",
		"group": "root",
	}

	/*
		[hotswap]
		triggers=post.update
		path=Resources.<AWS::AutoScalingLaunchConfiguration LogicalId>
		action='logger -p daemon.info updated!'
		runas=ec2-user
	*/
	hooksConf := map[string]interface{}{
		"content": map[string]interface{}{
			"Fn::Join": []interface{}{
				"",
				[]interface{}{
					"[hotswap]\n",
					"triggers=post.update\n",
					"path=Resources.", autoScalingLaunchConfigurationLogicalId, "\n",
					// "action=logger -p daemon.info hotswap triggered\n",
					"action=/opt/aws/bin/cfn-init -c hotswap",
					" --region ", map[string]string{"Ref": "AWS::Region"},
					" --stack ", map[string]string{"Ref": "AWS::StackId"},
					" -r ", autoScalingLaunchConfigurationLogicalId, "\n",
					"runas=root\n",
					"\n",
				},
			},
		},
		"mode":  "000644",
		"owner": "root",
		"group": "root",
	}

	tmpl, err := template.New("").Parse(files.PorterBootstrap)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, context)
	if err != nil {
		return nil, err
	}

	bootstrapFile := map[string]interface{}{
		"content": map[string]interface{}{
			// Split only to Fn::Join to make the template more human-readable
			"Fn::Join": []interface{}{
				"\n",
				strings.Split(buf.String(), "\n"),
			},
		},
		"mode":  "000755",
		"owner": "root",
		"group": "root",
	}

	buf.Reset()

	tmpl, err = template.New("").Parse(files.PorterHotswap)
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(&buf, context)
	if err != nil {
		return nil, err
	}

	hotSwapContents := []interface{}{
		"#!/bin/bash -e\n",
		"export AWS_STACKID=", map[string]string{"Ref": "AWS::StackId"}, "\n",
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		hotSwapContents = append(hotSwapContents, line+"\n")
	}

	hotswapFile := map[string]interface{}{
		"content": map[string]interface{}{
			"Fn::Join": []interface{}{
				"",
				hotSwapContents,
			},
		},
		"mode":  "000755",
		"owner": "root",
		"group": "root",
	}

	motd := map[string]interface{}{
		"content": map[string]interface{}{
			// Split only to Fn::Join to make the template more human-readable
			"Fn::Join": []interface{}{
				"\n",
				strings.Split(files.Motd, "\n"),
			},
		},
		"mode":  "000744",
		"owner": "root",
		"group": "root",
	}

	logRotate := map[string]interface{}{
		"content": files.LogrotatePorter,
		"mode":    "000644",
		"owner":   "root",
		"group":   "root",
	}

	awsCloudformationInit := map[string]interface{}{
		"configSets": map[string]interface{}{
			"bootstrap": []string{"bootstrapConfig"},
			"hotswap":   []string{"hotswapConfig"},
		},
		"bootstrapConfig": map[string]interface{}{
			"services": map[string]interface{}{
				"sysvinit": map[string]interface{}{
					"cfn-hup": map[string]interface{}{
						"enabled":       "true",
						"ensureRunning": "true",
						"files":         []string{"/etc/cfn/cfn-hup.conf", "/etc/cfn/hooks.conf"},
					},
					"haproxy": map[string]interface{}{
						"enabled":       "true",
						"ensureRunning": "true",
					},
				},
			},
			"files": map[string]interface{}{
				"/etc/cfn/cfn-hup.conf":        cfnHupConf,
				"/etc/cfn/hooks.conf":          hooksConf,
				"/usr/bin/porter_bootstrap":    bootstrapFile,
				"/usr/bin/porter_hotswap":      hotswapFile,
				"/etc/update-motd.d/99-porter": motd,
				"/etc/logrotate.d/porter":      logRotate,
			},
			"users": map[string]interface{}{
				"porter-docker": map[string]interface{}{
					"groups":  []string{},
					"uid":     constants.ContainerUserUid,
					"homeDir": "/home/porter-docker",
				},
			},
		},
		// Why not just call /usr/bin/porter_hotswap again?
		// We need to install the rewritten file first
		"hotswapConfig": map[string]interface{}{
			"commands": map[string]interface{}{
				"performHotswap": map[string]interface{}{
					"command": "/usr/bin/porter_hotswap",
					"cwd":     "/",
				},
			},
			"files": map[string]interface{}{
				"/usr/bin/porter_hotswap": hotswapFile,
			},
		},
	}
	return awsCloudformationInit, nil
}

func cfnReadOnly(value string) map[string]interface{} {
	return map[string]interface{}{
		"content": map[string]interface{}{
			// Split only to Fn::Join to make the template more human-readable
			"Fn::Join": []interface{}{
				"\n",
				strings.Split(value, "\n"),
			},
		},
		"mode":  "000644",
		"owner": "root",
		"group": "root",
	}
}

func cfnExecutable(value string) map[string]interface{} {
	return map[string]interface{}{
		"content": map[string]interface{}{
			// Split only to Fn::Join to make the template more human-readable
			"Fn::Join": []interface{}{
				"\n",
				strings.Split(value, "\n"),
			},
		},
		"mode":  "000755",
		"owner": "root",
		"group": "root",
	}
}
