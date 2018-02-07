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
package build

import (
	"os"

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/hook"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision"

	"github.com/phylake/go-cli"
)

type PackCmd struct{}

func (recv *PackCmd) Name() string {
	return "pack"
}

func (recv *PackCmd) ShortHelp() string {
	return "Create a service payload"
}

func (recv *PackCmd) LongHelp() string {
	return ""
}

func (recv *PackCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *PackCmd) Execute(args []string) bool {

	if !doPack() {
		os.Exit(1)
	}

	return true
}

func doPack() (success bool) {
	log := logger.CLI("cmd", "pack")

	defer func() {

		postHookSuccess := hook.Execute(log, constants.HookPostPack, "", nil, success)

		success = success && postHookSuccess
	}()

	if !hook.Execute(log, constants.HookPrePack, "", nil, true) {
		return
	}

	config, getConfigSuccess := conf.GetConfig(log, true)
	if !getConfigSuccess {
		return
	}

	if os.Getenv(constants.EnvConfig) != "" {
		config.Print()
	}

	success = provision.Package(log, config)

	if !success {
		log.Error("Package failed")
		return
	}

	log.Info("Packaged service", "FilePath", constants.PayloadPath)

	success = true
	return
}
