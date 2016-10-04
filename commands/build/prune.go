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
	"io/ioutil"
	"os"

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision_output"
	"github.com/adobe-platform/porter/prune"
	"github.com/phylake/go-cli"
)

type PruneCmd struct{}

func (recv *PruneCmd) Name() string {
	return "prune"
}

func (recv *PruneCmd) ShortHelp() string {
	return "Delete extra CloudFormation stacks"
}

func (recv *PruneCmd) LongHelp() string {
	return `NAME
    prune -- Delete extra CloudFormation stacks

SYNOPSIS
    prune [--keep <stacks to keep>]

DESCRIPTION
    Delete extra CloudFormation stacks. Instances attached to the configured
    ELB are not candidates for deletion.

OPTIONS
    --keep
        The number of stacks to keep with the status CREATE_COMPLETE,
        UPDATE_COMPLETE, or ROLLBACK_COMPLETE not including the stack with
        instances attached to the configured ELB.

        The default is 0 meaning only the stack with instances attached to the
        configured ELB will be kept.

        Eligible stacks will be sorted by creation time with the oldest being
        deleted first.`
}

func (recv *PruneCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *PruneCmd) Execute(args []string) bool {

	if len(args) == 1 && args[0] == "--help" {
		return false
	}

	var elbTag string
	var keepCount int

	if len(args) > 0 {

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)

		flagSet.IntVar(&keepCount, "keep", 0, "")
		flagSet.StringVar(&elbTag, "elb", "", "")
		flagSet.Parse(args)

		if keepCount < 0 {
			return false
		}
	}

	log := logger.CLI("cmd", "build-prune")

	config, success := conf.GetAlteredConfig(log)
	if !success {
		os.Exit(1)
	}

	provisionedEnv := &provision_output.Environment{}
	provisionEnvBytes, err := ioutil.ReadFile(constants.ProvisionOutputPath)
	if err != nil {
		log.Error("ioutil.ReadFile", "Error", err)
		os.Exit(1)
	}

	err = json.Unmarshal(provisionEnvBytes, provisionedEnv)
	if err != nil {
		log.Error("json.Unmarshal", "Error", err)
		os.Exit(1)
	}

	env, err := config.GetEnvironment(provisionedEnv.Environment)
	if err != nil {
		log.Error("GetEnvironment", "Error", err)
		os.Exit(1)
	}

	if !hook.Execute(log, constants.HookPrePrune,
		provisionedEnv.Environment, provisionedEnv.Regions, true) {
		os.Exit(1)
	}

	commandSuccess := prune.Do(log, config, env, keepCount, true, elbTag)

	hookSuccess := hook.Execute(log, constants.HookPostPrune,
		provisionedEnv.Environment, provisionedEnv.Regions, commandSuccess)

	if !commandSuccess || !hookSuccess {
		os.Exit(1)
	}

	log.Info("Prune complete")
	return true
}
