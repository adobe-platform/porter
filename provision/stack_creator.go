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
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/cfn"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cfnlib "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/inconshreveable/log15"
)

type (
	// A struct for manipulating a Cloudformation stack in a single region
	stackCreator struct {
		log  log15.Logger
		args StackArgs

		stackName string

		config            conf.Config
		environment       conf.Environment
		region            conf.Region
		servicePayloadKey string

		secretsKey      string
		secretsLocation string

		registryDeployment bool

		roleSession *session.Session

		// Stack creation is mostly the same between CreateStack and UpdateStack
		// The difference is in the API call to CloudFormation
		cfnAPI func(*cfnlib.CloudFormation, CfnApiInput) (string, bool)

		templateTransforms map[string][]MapResource
	}
)

const (
	s3KeyOptTemplate = 1 << iota
	s3KeyOptDeployment
)

func (recv *stackCreator) createUpdateStackForRegion(outChan chan CreateStackRegionOutput, errChan chan struct{}) {

	checksum, success := recv.uploadServicePayload()
	if !success {
		// uploadServicePayload logs errors. all we care about is success
		errChan <- struct{}{}
		return
	}

	if !recv.uploadSecrets(checksum) {
		// uploadSecrets logs errors. all we care about is success
		errChan <- struct{}{}
		return
	}

	stackId, success := recv.createStack()
	if !success {
		// createStack logs errors. all we care about is success
		errChan <- struct{}{}
		return
	}

	output := CreateStackRegionOutput{
		StackId: stackId,
		Region:  recv.region.Name,
	}

	outChan <- output
}

func (recv *stackCreator) uploadServicePayload() (checksum string, success bool) {

	defer exec.Command("rm", "-rf", constants.PayloadPath).Run()

	payloadBytes, err := ioutil.ReadFile(constants.PayloadPath)
	if err != nil {
		recv.log.Error("ReadFile payload", "Error", err)
		return
	}

	s3Client := s3.New(recv.roleSession)

	// TODO don't use a digest that requires everything to be in memory
	checksumArray := md5.Sum(payloadBytes)
	checksum = hex.EncodeToString(checksumArray[:])
	recv.servicePayloadKey = fmt.Sprintf("%s/%s.tar", recv.s3KeyRoot(s3KeyOptDeployment), checksum)

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(recv.region.S3Bucket),
		Key:    aws.String(recv.servicePayloadKey),
	}

	headObjectOutput, err := s3Client.HeadObject(headObjectInput)
	if err == nil {
		if headObjectOutput.ContentLength != nil && *headObjectOutput.ContentLength > 0 {
			recv.log.Info("Service payload exists", "S3key", recv.servicePayloadKey)
			success = true
			return
		}
	} else if !strings.Contains(err.Error(), "404") {
		recv.log.Error("HeadObject", "Error", err)
		if strings.Contains(err.Error(), "403") {
			recv.log.Error("s3:GetObject and s3:ListBucket are needed for this operation to work")
		}
		return
	}

	uploadInput := &s3manager.UploadInput{
		Bucket:          aws.String(recv.region.S3Bucket),
		Key:             aws.String(recv.servicePayloadKey),
		Body:            bytes.NewReader(payloadBytes),
		ContentEncoding: aws.String("gzip"),
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading service payload",
		"S3key", recv.servicePayloadKey,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(uploadInput)
	if err != nil {
		recv.log.Error("Upload failure", "Error", err)
		return
	}

	dockerRegistry := os.Getenv(constants.EnvDockerRegistry)
	dockerRepository := os.Getenv(constants.EnvDockerRepository)
	dockerPushUsername := os.Getenv(constants.EnvDockerPushUsername)
	dockerPushPassword := os.Getenv(constants.EnvDockerPushPassword)

	if dockerRegistry != "" && dockerRepository != "" {
		recv.registryDeployment = true

		if dockerPushUsername != "" && dockerPushPassword != "" {

			recv.log.Info("docker login")
			loginCmd := exec.Command("docker", "login",
				"-u", dockerPushUsername,
				"-p", dockerPushPassword,
				dockerRegistry)
			loginCmd.Stderr = os.Stderr
			err := loginCmd.Run()
			if err != nil {
				recv.log.Error("docker login", "Error", err)
				return
			}
		}

		containerCount := len(recv.region.Containers)
		pushSuccessChan := make(chan bool, containerCount)

		for _, container := range recv.region.Containers {

			go func(log log15.Logger, container conf.Container) {
				log = log.New("ImageTag", container.Name)

				log.Info("docker push")
				pushCmd := exec.Command("docker", "push", container.Name)
				pushCmd.Stderr = os.Stderr
				err := pushCmd.Run()
				if err != nil {
					log.Error("docker push", "Error", err)
					pushSuccessChan <- false
					return
				}
				pushSuccessChan <- true

			}(recv.log, *container)
		}

		for i := 0; i < containerCount; i++ {
			pushSuccess := <-pushSuccessChan
			if !pushSuccess {
				return
			}
		}
	}

	success = true
	return
}

func (recv *stackCreator) createStack() (stackId string, success bool) {

	client := cloudformation.New(recv.roleSession)

	templateBytes, creationSuccess := recv.createTemplate()
	if !creationSuccess {
		return
	}

	err := ioutil.WriteFile(constants.CloudFormationTemplatePath, templateBytes, 0644)
	if err != nil {
		errorMessage := fmt.Sprintf("Unable to write %s file", constants.CloudFormationTemplatePath)
		recv.log.Error(errorMessage, "Error", err)
		return
	}

	checksumArray := md5.Sum(templateBytes)
	checksum := hex.EncodeToString(checksumArray[:])
	templateS3Key := fmt.Sprintf("%s/%s", recv.s3KeyRoot(s3KeyOptTemplate), checksum)

	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String(recv.region.S3Bucket),
		Key:    aws.String(templateS3Key),
		Body:   bytes.NewReader(templateBytes),
	}

	if recv.region.SSEKMSKeyId != nil {
		uploadInput.SSEKMSKeyId = recv.region.SSEKMSKeyId
		uploadInput.ServerSideEncryption = aws.String("aws:kms")
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading CloudFormation template",
		"S3bucket", recv.region.S3Bucket,
		"S3key", templateS3Key,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(uploadInput)
	if err != nil {
		recv.log.Error("Upload failure", "Error", err)
		return
	}

	templateUrl := fmt.Sprintf("https://s3.amazonaws.com/%s/%s",
		recv.region.S3Bucket, templateS3Key)

	params := CfnApiInput{
		Environment: recv.environment.Name,
		Region:      recv.region.Name,
		SecretsKey:  recv.secretsKey,
		SecretsLoc:  recv.secretsLocation,
		TemplateUrl: templateUrl,
	}

	stackId, success = recv.cfnAPI(client, params)
	return
}

func (recv *stackCreator) createTemplate() (templateBytes []byte, success bool) {

	var err error
	template := cfn.NewTemplate()

	stackDefinitionPath, err := recv.environment.GetStackDefinitionPath(recv.region.Name)
	if err != nil {
		recv.log.Error("GetStackDefinitionPath", "Error", err)
		return
	}

	if stackDefinitionPath != "" {
		recv.log.Info("Using custom stack definition", "Path", stackDefinitionPath)

		stackFile, err := os.Open(stackDefinitionPath)
		if err != nil {
			recv.log.Error("os.Open",
				"Path", stackDefinitionPath,
				"Error", err)
			return
		}

		err = json.NewDecoder(stackFile).Decode(template)
		if err != nil {
			recv.log.Error("json.Decode",
				"Path", stackDefinitionPath,
				"Error", err)
			return
		}
	}

	template.ParseResources()

	if !recv.mutateTemplate(template) {
		return
	}

	// serialize expanded template
	templateBytes, err = json.Marshal(template)
	if err != nil {
		recv.log.Error("json.Marshal", "Error", err)
		return
	}

	success = true
	return
}

func (recv *stackCreator) mutateTemplate(template *cfn.Template) (success bool) {

	template.Description = fmt.Sprintf("%s (powered by porter %s)", recv.config.ServiceName, constants.Version)

	success = recv.ensureResources(template)
	if !success {
		return
	}

	success = recv.mapResources(template)
	if !success {
		return
	}

	success = true
	return
}

func (recv *stackCreator) s3KeyRoot(prefixOpt int) string {
	var prefix string
	if prefixOpt&s3KeyOptTemplate == s3KeyOptTemplate {
		prefix = "porter-template"
	} else if prefixOpt&s3KeyOptDeployment == s3KeyOptDeployment {
		prefix = "porter-deployment"
	} else {
		panic(fmt.Errorf("invalid option %d", prefixOpt))
	}

	return fmt.Sprintf("%s/%s/%s/%s",
		prefix, recv.config.ServiceName, recv.environment.Name, recv.config.ServiceVersion)
}
