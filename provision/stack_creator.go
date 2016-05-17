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
	"runtime"
	"strconv"
	"strings"

	"github.com/adobe-platform/porter/aws/cloudformation"
	"github.com/adobe-platform/porter/aws_session"
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

		serviceVersion string

		config      conf.Config
		environment conf.Environment
		region      conf.Region
		s3Key       string

		roleSession *session.Session

		// Stack creation is mostly the same between CreateStack and UpdateStack
		// The difference is in the API call to CloudFormation
		cfnAPI func(*cfnlib.CloudFormation, CfnApiInput) (string, bool)

		templateTransforms map[string][]MapResource
	}
)

func (recv *stackCreator) createUpdateStackForRegion(outChan chan CreateStackRegionOutput, errChan chan struct{}) {

	checksum, success := recv.uploadServicePayload()
	if !success {
		// uploadServicePayload logs errors. all we care about is success
		errChan <- struct{}{}
		return
	}

	if !recv.copyEnvFile(checksum) {
		// copyEnvFile logs errors. all we care about is success
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

	payloadBytes, err := ioutil.ReadFile(constants.PayloadPath)
	if err != nil {
		recv.log.Error("ReadFile payload", "Error", err)
		return
	}

	s3Client := s3.New(recv.roleSession)

	// TODO don't use a digest that requires everything to be in memory
	checksumRaw := md5.Sum(payloadBytes)
	checksum = hex.EncodeToString(checksumRaw[:])
	recv.s3Key = recv.config.ServiceName + "/" + checksum + ".tar"

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(recv.region.S3Bucket),
		Key:    aws.String(recv.s3Key),
	}

	headObjectOutput, err := s3Client.HeadObject(headObjectInput)
	if err == nil {
		if headObjectOutput.ContentLength != nil && *headObjectOutput.ContentLength > 0 {
			recv.log.Info("Service payload exists", "S3key", recv.s3Key)
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

	input := &s3manager.UploadInput{
		Bucket:          aws.String(recv.region.S3Bucket),
		Key:             aws.String(recv.s3Key),
		Body:            bytes.NewReader(payloadBytes),
		ContentEncoding: aws.String("gzip"),
	}

	s3Manager := s3manager.NewUploader(recv.roleSession)
	s3Manager.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

	recv.log.Info("Uploading service payload",
		"S3key", recv.s3Key,
		"Concurrency", s3Manager.Concurrency)

	_, err = s3Manager.Upload(input)
	if err != nil {
		recv.log.Error("Upload failure", "Error", err)
		return
	}

	success = true
	return
}

func (recv *stackCreator) copyEnvFile(checksum string) (success bool) {
	s3DstClient := s3.New(recv.roleSession)

	roleArn, err := recv.environment.GetRoleARN(recv.region.Name)
	if err != nil {
		recv.log.Error("GetRoleARN", "Error", err)
		return
	}

	containerNameToDstKey := make(map[string]string)

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
			recv.log.Error("GetObject",
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
			recv.log.Error("ioutil.ReadAll",
				"Error", err,
				"Container", container.Name,
				"SrcEnvFile.S3Bucket", container.SrcEnvFile.S3Bucket,
				"SrcEnvFile.S3Key", container.SrcEnvFile.S3Key,
			)
			return
		}

		etag, err := strconv.Unquote(*getObjectOutput.ETag)
		if err != nil {
			recv.log.Error("Unquote "+*getObjectOutput.ETag, "Error", err)
			return
		}

		dstEnvFileS3Key := recv.config.ServiceName + "/" + etag + ".env-file"
		recv.log.Info("Set destination env file", "S3Key", dstEnvFileS3Key)

		putObjectInput := &s3.PutObjectInput{
			Bucket:               aws.String(container.DstEnvFile.S3Bucket),
			Key:                  aws.String(dstEnvFileS3Key),
			Body:                 bytes.NewReader(getObjectBytes),
			SSEKMSKeyId:          aws.String(container.DstEnvFile.KMSARN),
			ServerSideEncryption: aws.String("aws:kms"),
		}

		_, err = s3DstClient.PutObject(putObjectInput)
		if err != nil {
			recv.log.Error("PutObject",
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

	params := CfnApiInput{
		Environment:   recv.environment.Name,
		Region:        recv.region.Name,
		TemplateBytes: templateBytes,
	}

	stackId, success = recv.cfnAPI(client, params)
	return
}

func (recv *stackCreator) createTemplate() (templateBytes []byte, success bool) {

	var err error
	template := cfn.NewTemplate()

	switch recv.region.PrimaryTopology() {
	case conf.Topology_Inet:
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
	case conf.Topology_Worker:
		recv.log.Error("not yet supporting worker topologies")
		return
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
