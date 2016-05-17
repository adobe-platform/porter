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
package build

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/logger"

	"github.com/phylake/go-cli"
)

type (
	NotifyCmd struct{}
	SlackBody struct {
		Text string `json:"text"`
	}
)

func (recv *NotifyCmd) Name() string {
	return "notify"
}

func (recv *NotifyCmd) ShortHelp() string {
	return "CI Slack notifications"
}

func (recv *NotifyCmd) LongHelp() string {
	return `NAME:
    notify -- CI Slack notifications

SYNOPSIS
    notify -go-ci -phase <pack | provision | promote> -success=<t|f>

DESCRIPTION
    Post message to a configured incoming webhook.
    https://api.slack.com/incoming-webhooks

    Currently GO CI is the only supported CI system.`
}

func (recv *NotifyCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *NotifyCmd) Execute(args []string) bool {
	log := logger.CLI("cmd", "build-notify")

	if len(args) > 1 && args[0] == "-go-ci" {
		var buildPhase, webhookURL string
		var phaseSuccess bool

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)
		flagSet.StringVar(&buildPhase, "phase", "", "")
		flagSet.BoolVar(&phaseSuccess, "success", false, "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		err := flagSet.Parse(args[1:])
		if err != nil {
			log.Warn("flagSet.Parse", "Error", err)
			return true
		}

		config, success := conf.GetAlteredConfig(log)
		if !success {
			return true
		}

		switch buildPhase {
		case "pack":
			if phaseSuccess {
				webhookURL = config.Slack.PackSuccessHook
			} else {
				webhookURL = config.Slack.PackFailureHook
			}
		case "provision":
			if phaseSuccess {
				webhookURL = config.Slack.ProvisionSuccessHook
			} else {
				webhookURL = config.Slack.ProvisionFailureHook
			}
		case "promote":
			if phaseSuccess {
				webhookURL = config.Slack.PromoteSuccessHook
			} else {
				webhookURL = config.Slack.PromoteFailureHook
			}
		default:
			log.Warn("invalid -phase", "phase", buildPhase)
			return true
		}

		if webhookURL == "" {
			return true
		}

		slackPost(webhookURL, goCIMessage(buildPhase, phaseSuccess))
		return true
	}

	return false
}

func slackPost(slackURL string, message string) {

	log := logger.CLI()
	messageBody := &SlackBody{
		Text: message,
	}

	messageBodyBytes, err := json.Marshal(messageBody)
	if err != nil {
		log.Error("Slack message body json.Marshal failed", "Error", err)
		return
	}

	resp, err := http.PostForm(slackURL, url.Values{"payload": {string(messageBodyBytes)}})

	if err != nil || resp.StatusCode != 200 {
		log.Error("Post to slack failed", "Error", err, "StatusCode", resp.StatusCode)
		return
	}

	log.Info("Message posted to slack", "StatusCode", resp.StatusCode)

}

func goCIMessage(buildPhase string, phaseSuccess bool) string {

	pipelineUrl := fmt.Sprintf("%s/%s/%s",
		os.Getenv("GO_NOTIFICATION_URL"),
		"tab/pipeline/history",
		os.Getenv("GO_PIPELINE_NAME"))

	buildUrl := fmt.Sprintf("%s/%s/%s/%s", os.Getenv("GO_NOTIFICATION_URL"),
		"pipelines/value_stream_map",
		os.Getenv("GO_PIPELINE_NAME"),
		os.Getenv("GO_PIPELINE_COUNTER"))

	stageUrl := fmt.Sprintf("%s/%s/%s/%s/%s/%s", os.Getenv("GO_NOTIFICATION_URL"),
		"pipelines",
		os.Getenv("GO_PIPELINE_NAME"),
		os.Getenv("GO_PIPELINE_COUNTER"),
		os.Getenv("GO_STAGE_NAME"),
		os.Getenv("GO_STAGE_COUNTER"))

	jobUrl := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s%s",
		os.Getenv("GO_NOTIFICATION_URL"),
		"tab/build/detail",
		os.Getenv("GO_PIPELINE_NAME"),
		os.Getenv("GO_PIPELINE_COUNTER"),
		os.Getenv("GO_STAGE_NAME"),
		os.Getenv("GO_STAGE_COUNTER"),
		os.Getenv("GO_JOB_NAME"),
		"#tab-console")

	msg := fmt.Sprintf("<%s|%s> >> <%s|%s> >> <%s|%s/%s> >> <%s|%s>",
		pipelineUrl,
		os.Getenv("GO_PIPELINE_NAME"),
		buildUrl,
		os.Getenv("GO_PIPELINE_COUNTER"),
		stageUrl,
		os.Getenv("GO_STAGE_NAME"),
		os.Getenv("GO_STAGE_COUNTER"),
		jobUrl,
		os.Getenv("GO_JOB_NAME"))

	if phaseSuccess {
		msg = msg + " passed"
	} else {
		msg = "*FAILED:* " + msg
	}

	return msg
}
