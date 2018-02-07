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
package wait_handle

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon/flags"
	"github.com/adobe-platform/porter/daemon/identity"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type (
	waitConditionReq struct {
		Status   string `json:"Status"`
		Reason   string `json:"Reason"`
		UniqueID string `json:"UniqueId"`
		Data     string `json:"Data"`
	}

	awsErrorResp struct {
		XMLName xml.Name `xml:"Error"`
		Code    string   `xml:"Code"`
		Message string   `xml:"Message"`
	}
)

const (
	wcExpireError     = "Request has expired"
	fastSleepDuration = 2 * time.Second
)

func Call() {
	log := logger.Daemon(
		"package", "wait_handle",
		"AWS_STACKID", os.Getenv("AWS_STACKID"),
	)

	var (
		describeStackResourceOutput *cloudformation.DescribeStackResourceOutput
		err                         error
	)

	// Poll service health check until healthy and then call wait handle on its
	// behalf.
	//
	// NOTE: This polls the health check of the primary docker container via
	//       haproxy which ensures the haproxy configuration works with the
	//       container. This is both a stronger guarantee and easier to deal
	//       with than polling the published (-P) port of the primary container
	//       because haproxy also polls the health check to determine if a
	//       backend is up or down. If we polled the container and beat the
	//       poll that haproxy performs there's a window of time where we would
	//       think the service is alive but haproxy would return 503s.

	sleepDuration := fastSleepDuration
	consecutiveHealth := 0

	healthCheckClient := &http.Client{
		Timeout: constants.HC_Timeout * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	msg := flags.HealthCheckMethod + " " + flags.HealthCheckPath
	hcUrl := "http://localhost" + flags.HealthCheckPath

	for {
		// assume a worker or cron primary topology
		if flags.HealthCheckMethod == "" && flags.HealthCheckPath == "" {
			break
		}

		time.Sleep(sleepDuration)

		req, err := http.NewRequest(flags.HealthCheckMethod, hcUrl, nil)
		if err != nil {
			log.Warn("http.NewRequest", "Error", err)
			continue
		}

		resp, err := healthCheckClient.Do(req)

		if err != nil {
			log.Warn(msg, "Error", err)
			continue
		}

		if resp.StatusCode == 200 {

			consecutiveHealth++
			sleepDuration = constants.HC_Interval * time.Second

			log.Info(fmt.Sprintf("successful health check %d/%d",
				consecutiveHealth, constants.HC_HealthyThreshold))

		} else {

			consecutiveHealth = 0
			sleepDuration = fastSleepDuration
			log.Warn(msg, "StatusCode", resp.StatusCode)
		}

		if consecutiveHealth >= constants.HC_HealthyThreshold {
			log.Info("health threshold met. calling wait handle")
			break
		}
	}

	ii, err := identity.Get(log)
	if err != nil {
		// identity.Get logs errors
		return
	}

	cfnClient := cloudformation.New(aws_session.Get(ii.AwsCreds.Region))

	waitConditionHandle, exists := ii.Tags[constants.PorterWaitConditionHandleLogicalIdTag]
	if !exists {
		log.Error("missing EC2 tag " + constants.PorterWaitConditionHandleLogicalIdTag)
		return
	}

	dsri := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: aws.String(waitConditionHandle),
		StackName:         aws.String(os.Getenv("AWS_STACKID")),
	}

	retryMsg := func(i int) { log.Warn("DescribeStackResource retrying", "Count", i) }
	if !util.SuccessRetryer(9, retryMsg, func() bool {
		describeStackResourceOutput, err = cfnClient.DescribeStackResource(dsri)
		if err != nil {
			log.Error("DescribeStackResource", "Error", err)
			return false
		}
		return true
	}) {
		return
	}

	waitHandleURL := *describeStackResourceOutput.StackResourceDetail.PhysicalResourceId

	reqData := &waitConditionReq{
		Status:   "SUCCESS",
		Reason:   "Configuration Complete",
		UniqueID: ii.Instance.InstanceID,
		Data:     "Service has successfully started",
	}

	j, err := json.Marshal(reqData)
	if err != nil {
		log.Error("Failed to marshal request body", "Error", err)
		return
	}

	log.Info("SignalWaitCondition: About to signal WaitCondition", "URL", waitHandleURL, "Payload", string(j))
	req, err := http.NewRequest("PUT", waitHandleURL, bytes.NewReader(j))
	if err != nil {
		log.Error("Failed to construct request", "Error", err)
		return
	}
	// AWS provides us with a signed URL. Golang then takes this signed,
	// unicode encoded URL, and tries to do us a favor by modifying the
	// signed unicode characters into text. This in turn modifies the
	// signature of the URL and thus prohibits registration. So the
	// following song-and-dance is to set the entire URL as "Opaque",
	// which makes Golang ignore any encoding / decoding that it might
	// do in its default case.
	href, err := url.Parse(waitHandleURL)
	if err != nil {
		log.Error("Failed to parse signed URL", "Error", err)
		return
	}

	req.URL = &url.URL{
		Scheme: href.Scheme,
		Host:   href.Host,
		Opaque: strings.Replace(waitHandleURL, "https:", "", 1),
	}

	// Empty content-type is intentional. If you supply a content-type
	// then no error will be returned but registration will not succeed.
	// See example in docs at:
	// http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/quickref-waitcondition.html
	req.Header.Set("Content-Type", "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("Wait condition request", "Error", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", "Error", err)
		return
	}

	if resp.StatusCode == 200 {

		log.Info("Signal WaitCondition succeeded")
	} else {
		log.Error("Signal WaitCondition failed", "StatusCode", resp.StatusCode)
		errResp := new(awsErrorResp)
		if err = xml.Unmarshal(body, errResp); err != nil {
			log.Error("xml.Unmarshal", "Error", err)
			return
		}

		if errResp.Message == wcExpireError {
			log.Error("Wait condition URI has expired")
			return
		}
	}
}
