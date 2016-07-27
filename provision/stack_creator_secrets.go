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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/constants"
	dockerutil "github.com/adobe-platform/porter/docker/util"
	"github.com/adobe-platform/porter/secrets"
	"github.com/adobe-platform/porter/stdin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (recv *stackCreator) getSecrets() (containerSecrets map[string]string, success bool) {
	recv.log.Debug("BEGIN getSecrets()")
	defer recv.log.Debug("END getSecrets()")

	stdinBytes, err := stdin.GetBytes()

	recv.log.Debug("STDIN", "STDIN", string(stdinBytes))

	if err == nil && len(stdinBytes) > 0 {

		containerSecrets, success = recv.getStdinSecrets(stdinBytes)
	} else {

		containerSecrets, success = recv.getS3Secrets()
	}

	for containerName := range containerSecrets {
		recv.log.Debug(fmt.Sprintf("Container [%s] has secrets", containerName))
	}

	success = true
	return
}

func (recv *stackCreator) getStdinSecrets(stdinBytes []byte) (containerSecrets map[string]string, success bool) {

	containerSecrets = make(map[string]string)

	jsonRaw := make(map[string]interface{})
	if err := json.Unmarshal(stdinBytes, &jsonRaw); err == nil {

		if secretsRaw, ok := jsonRaw["container_secrets"].(map[string]interface{}); ok {
			recv.log.Info("Got container_secrets from stdin")

			if containerRaw, ok := secretsRaw[recv.region.Name].(map[string]interface{}); ok {

				for _, container := range recv.region.Containers {

					recv.log.Debug("Trying to match container from config", "ConfigContainer", container.OriginalName)

					nameMatch := false
					for containerName, containerSecretsRaw := range containerRaw {

						if secretKvps, ok := containerSecretsRaw.(map[string]interface{}); ok {

							recv.log.Debug("Candidate container from stdin", "StdinContainer", containerName)

							if container.OriginalName == containerName {
								secretEnvVars := make([]string, 0)

								for sKey, sValue := range secretKvps {
									secretEnvVars = append(secretEnvVars, sKey+"="+sValue.(string))
								}
								containerSecrets[container.OriginalName] = strings.Join(secretEnvVars, "\n")

								nameMatch = true
								break
							}
						} else {
							msg := fmt.Sprintf("container_secrets.%s.%s must be an object of key-value pairs that are strings", recv.region.Name, containerName)
							recv.log.Crit(msg)
							return
						}
					}

					if !nameMatch {
						recv.log.Warn("No secrets for " + container.OriginalName)
					}
				}

			} else {
				recv.log.Warn("Missing region in container_secrets config")
			}
		} else {
			recv.log.Warn("stdin contained valid JSON but no container_secrets key")
		}
	} else {
		recv.log.Crit("Invalid JSON on os.Stdin")
		return
	}

	success = true
	return
}

func (recv *stackCreator) getS3Secrets() (containerSecrets map[string]string, success bool) {
	s3DstClient := s3.New(recv.roleSession)

	roleArn, err := recv.environment.GetRoleARN(recv.region.Name)
	if err != nil {
		recv.log.Crit("GetRoleARN", "Error", err)
		return
	}

	containerSecrets = make(map[string]string)

	for _, container := range recv.region.Containers {

		if container.SrcEnvFile == nil {
			continue
		}
		recv.log.Info("Copying environment file")

		var s3SrcClient *s3.S3

		if container.SrcEnvFile.S3Region == "" {
			s3SrcClient = s3DstClient
		} else {
			srcSession := aws_session.STS(container.SrcEnvFile.S3Region, roleArn, 0)
			s3SrcClient = s3.New(srcSession)
		}

		getObjectInput := &s3.GetObjectInput{
			Bucket: aws.String(container.SrcEnvFile.S3Bucket),
			Key:    aws.String(container.SrcEnvFile.S3Key),
		}

		getObjectOutput, err := s3SrcClient.GetObject(getObjectInput)
		if err != nil {
			recv.log.Crit("GetObject",
				"Error", err,
				"Container", container.Name,
				"SrcEnvFile.S3Bucket", container.SrcEnvFile.S3Bucket,
				"SrcEnvFile.S3Key", container.SrcEnvFile.S3Key,
			)
			return
		}
		defer getObjectOutput.Body.Close()

		getObjectBytes, err := ioutil.ReadAll(getObjectOutput.Body)
		if err != nil {
			recv.log.Crit("ioutil.ReadAll",
				"Error", err,
				"Container", container.Name,
				"SrcEnvFile.S3Bucket", container.SrcEnvFile.S3Bucket,
				"SrcEnvFile.S3Key", container.SrcEnvFile.S3Key,
			)
			return
		}

		containerSecrets[container.OriginalName] = string(getObjectBytes)
	}

	success = true
	return
}

func (recv *stackCreator) uploadSecrets() (success bool) {
	s3DstClient := s3.New(recv.roleSession)

	containerToSecrets, getSecretsSuccess := recv.getSecrets()
	if !getSecretsSuccess {
		return
	}

	containerNameToDstKey := make(map[string]string)

	symmetricKey, err := secrets.GenerateKey()
	if err != nil {
		recv.log.Crit("secrets.GenerateKey", "Error", err)
		return
	}
	recv.secretsKey = hex.EncodeToString(symmetricKey)

	for _, container := range recv.region.Containers {

		containerSecretString, exists := containerToSecrets[container.OriginalName]
		if !exists {
			recv.log.Warn("Secrets config exists but not for this a container in this region",
				"ContainerName", container.OriginalName)
			continue
		}
		containerSecretString = dockerutil.CleanEnvFile(containerSecretString)

		if len(containerSecretString) == 0 {
			recv.log.Warn("After cleaning the env file it's empty",
				"ContainerName", container.OriginalName)
			continue
		}

		containerSecrets, err := secrets.Encrypt([]byte(containerSecretString), symmetricKey)
		if err != nil {
			recv.log.Crit("Secrets encryption failed",
				"Error", err,
				"ContainerName", container.OriginalName)
			return
		}

		checksumArray := md5.Sum(containerSecrets)
		checksum := hex.EncodeToString(checksumArray[:])

		dstEnvFileS3Key := fmt.Sprintf("%s/%s.env-file", recv.s3KeyRoot(), checksum)
		recv.log.Info("Set destination env file", "S3Key", dstEnvFileS3Key)

		putObjectInput := &s3.PutObjectInput{
			Bucket: aws.String(container.DstEnvFile.S3Bucket),
			Key:    aws.String(dstEnvFileS3Key),
			Body:   bytes.NewReader(containerSecrets),
		}

		if container.DstEnvFile.KMSARN != nil {
			putObjectInput.SSEKMSKeyId = container.DstEnvFile.KMSARN
			putObjectInput.ServerSideEncryption = aws.String("aws:kms")
		}

		_, err = s3DstClient.PutObject(putObjectInput)
		if err != nil {
			recv.log.Crit("PutObject",
				"Error", err,
				"Container", container.Name,
				"DstEnvFile.S3Bucket", container.DstEnvFile.S3Bucket,
				"DstEnvFile.KMSARN", container.DstEnvFile.KMSARN,
			)
			return
		}

		containerNameToDstKey[container.Name] = dstEnvFileS3Key
	}

	fnMeta := func(recv *stackCreator, template *cfn.Template, resource map[string]interface{}) bool {
		var (
			metadata map[string]interface{}
			ok       bool
		)

		if metadata, ok = resource["Metadata"].(map[string]interface{}); !ok {
			metadata = make(map[string]interface{})
			resource["Metadata"] = metadata
		}

		metadata[constants.MetadataAsEnvFiles] = containerNameToDstKey
		return true
	}

	// The problem is tagging. porterd already needs the EC2 instance to be
	// tagged with constants.PorterWaitConditionHandleLogicalIdTag so it can get
	// that resource handle and call the wait condition on behalf of a service.
	//
	// The next problem is where to put this metadata.
	//
	// While it doesn't make a ton of sense to put it on the
	// WaitConditionHandle, it makes less sense to tag the EC2 instance (which
	// has a tag limit) with the logical resource id of some other resource
	// where we might put this metadata
	var (
		fns []MapResource
		ok  bool
	)
	if fns, ok = recv.templateTransforms[cfn.CloudFormation_WaitConditionHandle]; !ok {
		fns = make([]MapResource, 0)
	}

	fns = append(fns, fnMeta)
	recv.templateTransforms[cfn.CloudFormation_WaitConditionHandle] = fns

	success = true
	return
}
