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
	"github.com/adobe-platform/porter/promote"
	"github.com/adobe-platform/porter/provision_output"

	"github.com/phylake/go-cli"
)

type PromoteCmd struct{}

func (recv *PromoteCmd) Name() string {
	return "promote"
}

func (recv *PromoteCmd) ShortHelp() string {
	return "Promote provisioned instances"
}

func (recv *PromoteCmd) LongHelp() string {
	return `NAME
    promote -- Promote provisioned instances

SYNOPSIS
    promote [-provision-output <provision output file>]

DESCRIPTION
    Promote newly provisioned instances and remove old instances from the
    configured elb

OPTIONS
    -provision-output
    	The path to a provision output file. This is only used for testing.
    	DO NOT provide this if calling from a build machine.`
}

func (recv *PromoteCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *PromoteCmd) Execute(args []string) bool {
	var provisionOutputPath, elbType string

	if len(args) == 1 && args[0] == "--help" {
		return false
	}

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	flagSet.StringVar(&provisionOutputPath, "provision-output", "", "")
	flagSet.StringVar(&elbType, "elb", "", "")
	flagSet.Parse(args)

	if provisionOutputPath == "" {
		provisionOutputPath = constants.ProvisionOutputPath
	}

	log := logger.CLI("cmd", "build-promote")

	config, success := conf.GetAlteredConfig(log)
	if !success {
		os.Exit(1)
	}

	provisionedEnv := &provision_output.Environment{}
	provisionEnvBytes, err := ioutil.ReadFile(provisionOutputPath)
	if err != nil {
		log.Error("Unable to read provision output file", "Error", err)
		os.Exit(1)
	}

	err = json.Unmarshal(provisionEnvBytes, provisionedEnv)
	if err != nil {
		log.Error("json unmarshal error on provision output", "Error", err)
		os.Exit(1)
	}

	commandSuccess := hook.Execute(log, constants.HookPrePromote,
		provisionedEnv.Environment, provisionedEnv.Regions, true)

	if commandSuccess {
		commandSuccess = promote.Promote(log, config, provisionedEnv, elbType)
	}

	commandSuccess = hook.Execute(log, constants.HookPostPromote,
		provisionedEnv.Environment, provisionedEnv.Regions, commandSuccess)

	if !commandSuccess {
		os.Exit(1)
	}

	log.Info("Promote complete")
	return true
}
