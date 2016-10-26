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
	"flag"
	"fmt"
	"os"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/logger"
	"github.com/phylake/go-cli"
)

type HookCmd struct{}

func (recv *HookCmd) Name() string {
	return "hook"
}

func (recv *HookCmd) ShortHelp() string {
	return "Build and run arbitrary docker files"
}

func (recv *HookCmd) LongHelp() string {
	return `NAME
    hook -- Build and run arbitrary docker files

SYNOPSIS
    hook -name <hook name> [-e <environment>]

DESCRIPTION
    hook builds and runs custom hooks defined in the hooks section of .porter/config
    to facilitate programmable deployment pipelines.  The six pre and post hooks
    for the pack, provision, promote, and prune build phases are automatically run
    and not accessible for manual invocation with this command.

OPTIONS
    -name
        The name of the hook in .porter/config

    -e  Environment from .porter/config`
}

func (recv *HookCmd) Execute(args []string) bool {
	if len(args) > 0 {

		var hookName, environment string
		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&hookName, "name", "", "")
		flagSet.StringVar(&environment, "e", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		log := logger.CLI("cmd", "hook")

		switch hookName {
		case constants.HookPrePack,
			constants.HookPostPack,
			constants.HookPreProvision,
			constants.HookPostProvision,
			constants.HookPreHotswap,
			constants.HookPostHotswap,
			constants.HookPrePromote,
			constants.HookPostPromote,
			constants.HookPrePrune,
			constants.HookPostPrune,
			constants.HookEC2Bootstrap:
			log.Warn(fmt.Sprintf("The hook %s is reserved because it is called automatically", hookName))
			log.Warn("Please remove the call to this hook. It will still be called by porter")
			log.Warn("In future versions of porter this will be an error")
			return true
		}

		if !hook.Execute(log, hookName, environment, nil, true) {
			os.Exit(1)
		}

		return true
	}

	return false
}

func (recv *HookCmd) SubCommands() []cli.Command {
	return nil
}
