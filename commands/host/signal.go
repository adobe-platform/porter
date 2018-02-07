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
package host

import (
	"flag"
	"fmt"
	"os"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/phylake/go-cli"
)

type SignalCmd struct{}

func (recv *SignalCmd) Name() string {
	return "signal"
}

func (recv *SignalCmd) ShortHelp() string {
	return "EC2 host signals"
}

func (recv *SignalCmd) LongHelp() string {
	return `NAME
    signal -- EC2 host signals

SYNOPSIS
    signal --hotswap-complete -r <region>

DESCRIPTION
    signal communicates EC2 host signals to other components

OPTIONS
    --hotswap-complete
        Signal that a hot swap occurred

    -r  AWS region`
}

func (recv *SignalCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *SignalCmd) Execute(args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "--hotswap-complete":
			if len(args) == 1 {
				return false
			}

			var region string
			flagSet := flag.NewFlagSet("", flag.ExitOnError)
			flagSet.StringVar(&region, "r", "", "")
			flagSet.Usage = func() {
				fmt.Println(recv.LongHelp())
			}
			flagSet.Parse(args[1:])

			signalQueue(region)
		default:
			return false
		}
		return true
	}

	return false
}

func signalQueue(regionStr string) {

	log := logger.Host("cmd", "signal")

	sqsClient := sqs.New(aws_session.Get(regionStr))

	log.Info("signaling hotswap complete")

	smi := &sqs.SendMessageInput{
		QueueUrl:    aws.String(os.Getenv("SIGNAL_QUEUE_URL")),
		MessageBody: aws.String("success"),
	}

	smiRetryMsg := func(i int) { log.Warn("SendMessage retrying", "Count", i) }
	if !util.SuccessRetryer(7, smiRetryMsg, func() bool {
		_, err := sqsClient.SendMessage(smi)
		if err != nil {
			log.Error("SendMessage", "Error", err)
			return false
		}
		return true
	}) {
		os.Exit(1)
	}

	log.Info("signaled hotswap complete")
}
