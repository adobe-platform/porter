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
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	dockerutil "github.com/adobe-platform/porter/docker/util"
	"github.com/adobe-platform/porter/secrets"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (recv *stackCreator) getSecrets() (containerSecrets map[string]string, success bool) {
	recv.log.Debug("getSecrets() BEGIN")
	defer recv.log.Debug("getSecrets() END")

	containerSecrets = make(map[string]string)

	for _, container := range recv.region.Containers {

		if container.SrcEnvFile == nil {
			continue
		}

		var envFile string
		if container.SrcEnvFile.ExecName != "" {

			envFile, success = recv.getExecSecrets(container)
		} else if container.SrcEnvFile.S3Bucket != "" && container.SrcEnvFile.S3Key != "" {

			envFile, success = recv.getS3Secrets(container)
		} else {
			continue
		}

		if !success {
			return
		}

		envFile = dockerutil.CleanEnvFile(envFile)

		containerSecrets[container.OriginalName] = envFile
	}

	for containerName := range containerSecrets {
		recv.log.Debug(fmt.Sprintf("Container [%s] has secrets", containerName))
	}

	success = true
	return
}

func (recv *stackCreator) getExecSecrets(container *conf.Container) (containerSecrets string, success bool) {

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	log := recv.log.New("ContainerName", container.OriginalName)
	log.Info("Getting secrets from exec")

	cmd := exec.Command(container.SrcEnvFile.ExecName, container.SrcEnvFile.ExecArgs...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		log.Error("exec.Command", "Error", err, "Stderr", stderrBuf.String())
		return
	}

	containerSecrets = stdoutBuf.String()
	success = true
	return
}

func (recv *stackCreator) getS3Secrets(container *conf.Container) (containerSecrets string, success bool) {
	s3DstClient := s3.New(recv.roleSession)
	log := recv.log.New("ContainerName", container.OriginalName)

	roleArn, err := recv.environment.GetRoleARN(recv.region.Name)
	if err != nil {
		log.Crit("GetRoleARN", "Error", err)
		return
	}

	log.Info("Getting secrets from S3")

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
		log.Crit("GetObject",
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
		log.Crit("ioutil.ReadAll",
			"Error", err,
			"Container", container.Name,
			"SrcEnvFile.S3Bucket", container.SrcEnvFile.S3Bucket,
			"SrcEnvFile.S3Key", container.SrcEnvFile.S3Key,
		)
		return
	}

	containerSecrets = string(getObjectBytes)
	success = true
	return
}

func (recv *stackCreator) uploadSecrets() (success bool) {
	recv.log.Debug("uploadSecrets() BEGIN")
	defer recv.log.Debug("uploadSecrets() END")

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

		log := recv.log.New("ContainerName", container.OriginalName)

		containerSecretString, exists := containerToSecrets[container.OriginalName]
		if !exists {
			log.Warn("Secrets config exists but not for this a container in this region")
			continue
		}

		if len(containerSecretString) == 0 {
			log.Warn("After cleaning the env file it's empty")
			continue
		}

		containerSecrets, err := secrets.Encrypt([]byte(containerSecretString), symmetricKey)
		if err != nil {
			log.Crit("Secrets encryption failed", "Error", err)
			return
		}

		checksumArray := md5.Sum(containerSecrets)
		checksum := hex.EncodeToString(checksumArray[:])

		dstEnvFileS3Key := fmt.Sprintf("%s/%s.env-file",
			recv.s3KeyRoot(s3KeyOptDeployment), checksum)
		log.Info("Set destination env file", "S3Key", dstEnvFileS3Key)

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
			log.Crit("PutObject",
				"Error", err,
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
