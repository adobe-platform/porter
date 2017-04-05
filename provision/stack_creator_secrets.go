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
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	dockerutil "github.com/adobe-platform/porter/docker/util"
	"github.com/adobe-platform/porter/secrets"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func (recv *stackCreator) getContainerSecrets() (containerSecrets map[string]string, success bool) {
	recv.log.Debug("getContainerSecrets() BEGIN")
	defer recv.log.Debug("getContainerSecrets() END")

	containerSecrets = make(map[string]string)

	for _, container := range recv.region.Containers {

		if container.SrcEnvFile == nil {
			recv.log.Debug("No src_env_file for " + container.OriginalName)
			continue
		}

		var envFile string
		if container.SrcEnvFile.ExecName != "" {

			envFile, success = recv.getExecContainerSecrets(container)
		} else if container.SrcEnvFile.S3Bucket != "" && container.SrcEnvFile.S3Key != "" {

			envFile, success = recv.getS3ContainerSecrets(container)
		} else {

			recv.log.Warn("src_env_file defined but missing both exec_* and s3_*")
			continue
		}

		if !success {
			return
		}

		envFile = dockerutil.CleanEnvFile(envFile)

		containerSecrets[container.Name] = envFile
	}

	for containerName := range containerSecrets {
		recv.log.Debug(fmt.Sprintf("Container [%s] has secrets", containerName))
	}

	success = true
	return
}

func (recv *stackCreator) getExecContainerSecrets(container *conf.Container) (containerSecrets string, success bool) {

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

func (recv *stackCreator) getS3ContainerSecrets(container *conf.Container) (containerSecrets string, success bool) {
	s3DstClient := s3.New(recv.roleSession)
	log := recv.log.New("ContainerName", container.OriginalName)

	recv.log.Debug("getS3ContainerSecrets() BEGIN")
	defer recv.log.Debug("getS3ContainerSecrets() END")

	roleArn, err := recv.environment.GetRoleARN(recv.region.Name)
	if err != nil {
		log.Crit("GetRoleARN", "Error", err)
		return
	}

	s3Location := fmt.Sprintf("s3://%s/%s",
		container.SrcEnvFile.S3Bucket, container.SrcEnvFile.S3Key)
	log.Info("Getting secrets from S3", "Location", s3Location)

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

func (recv *stackCreator) getHostSecrets() (hostSecrets []byte, success bool) {
	recv.log.Debug("getHostSecrets() BEGIN")
	defer recv.log.Debug("getHostSecrets() END")

	if recv.region.AutoScalingGroup.SecretsExecName == "" {
		success = true
		return
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	cmd := exec.Command(recv.region.AutoScalingGroup.SecretsExecName, recv.region.AutoScalingGroup.SecretsExecArgs...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		recv.log.Error("exec.Command", "Error", err, "Stderr", stderrBuf.String())
		return
	}

	hostSecrets = stdoutBuf.Bytes()
	success = true
	return
}

func (recv *stackCreator) getPemFile() (pemFile []byte, success bool) {
	recv.log.Debug("getPemFile() BEGIN")
	defer recv.log.Debug("getPemFile() END")

	if recv.environment.HAProxy.SSL.Pem == nil || recv.environment.HAProxy.SSL.Pem.SecretsExecName == "" {
		success = true
		return
	}

	execName := recv.environment.HAProxy.SSL.Pem.SecretsExecName
	execArgs := recv.environment.HAProxy.SSL.Pem.SecretsExecArgs

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	cmd := exec.Command(execName, execArgs...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		recv.log.Error("exec.Command", "Error", err, "Stderr", stderrBuf.String())
		return
	}

	pemFile = stdoutBuf.Bytes()
	success = true
	return
}

func (recv *stackCreator) uploadSecrets(checksum string) (success bool) {
	recv.log.Debug("uploadSecrets() BEGIN")
	defer recv.log.Debug("uploadSecrets() END")

	containerToSecrets, getContainerSecretsSuccess := recv.getContainerSecrets()
	if !getContainerSecretsSuccess {
		return
	}

	hostSecrets, getHostSecretsSuccess := recv.getHostSecrets()
	if !getHostSecretsSuccess {
		return
	}

	pemFile, getPemFileSuccess := recv.getPemFile()
	if !getPemFileSuccess {
		return
	}

	symmetricKey, err := secrets.GenerateKey()
	if err != nil {
		recv.log.Crit("secrets.GenerateKey", "Error", err)
		return
	}
	recv.secretsKey = hex.EncodeToString(symmetricKey)

	secretPayload := secrets.Payload{
		HostSecrets:        hostSecrets,
		ContainerSecrets:   containerToSecrets,
		DockerRegistry:     os.Getenv(constants.EnvDockerRegistry),
		DockerPullUsername: os.Getenv(constants.EnvDockerPullUsername),
		DockerPullPassword: os.Getenv(constants.EnvDockerPullPassword),
		PemFile:            pemFile,
	}

	var secretPayloadBuf bytes.Buffer

	err = gob.NewEncoder(&secretPayloadBuf).Encode(secretPayload)
	if err != nil {
		recv.log.Crit("gob.Marshal", "Error", err)
		return
	}

	recv.secretsLocation = fmt.Sprintf("%s/%s.secrets", recv.s3KeyRoot(s3KeyOptDeployment), checksum)

	secretPayloadBytesEnc, err := secrets.Encrypt(secretPayloadBuf.Bytes(), symmetricKey)
	if err != nil {
		recv.log.Crit("Secrets encryption failed", "Error", err)
		return
	}

	uploadInput := &s3manager.UploadInput{
		Bucket:       aws.String(recv.region.S3Bucket),
		Key:          aws.String(recv.secretsLocation),
		Body:         bytes.NewReader(secretPayloadBytesEnc),
		StorageClass: aws.String("STANDARD_IA"),
	}

	if recv.region.SSEKMSKeyId != nil {
		uploadInput.SSEKMSKeyId = recv.region.SSEKMSKeyId
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading secrets",
		"S3bucket", recv.region.S3Bucket,
		"S3key", recv.secretsLocation,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(uploadInput)
	if err != nil {
		recv.log.Error("Upload", "Error", err)
		return
	}

	success = true
	return
}
