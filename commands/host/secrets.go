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

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/secrets"
	"github.com/phylake/go-cli"
)

type SecretsCmd struct{}

func (recv *SecretsCmd) Name() string {
	return "secrets"
}

func (recv *SecretsCmd) ShortHelp() string {
	return "get secrets payload"
}

func (recv *SecretsCmd) LongHelp() string {
	return `NAME
    secrets -- get secrets payload

SYNOPSIS
    secrets --get -e <environment> -r <region>

DESCRIPTION
    secrets gets any secrets provided to porter and prints them on STDOUT.

    This command needs the latest porter config that exists on the host

OPTIONS
    -e  Environment from .porter/config

    -r  AWS region`
}

func (recv *SecretsCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *SecretsCmd) Execute(args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "--get":

			var environmentFlag, regionFlag string
			flagSet := flag.NewFlagSet("", flag.ExitOnError)
			flagSet.StringVar(&environmentFlag, "e", "", "")
			flagSet.StringVar(&regionFlag, "r", "", "")
			flagSet.Usage = func() {
				fmt.Println(recv.LongHelp())
			}
			flagSet.Parse(args[1:])

			log := logger.Host("cmd", "secrets")

			config, getStdinConfigSucces := conf.GetStdinConfig(log)
			if !getStdinConfigSucces {
				os.Exit(1)
			}

			environment, err := config.GetEnvironment(environmentFlag)
			if err != nil {
				log.Crit("GetEnvironment", "Error", err)
				os.Exit(1)
			}

			region, err := environment.GetRegion(regionFlag)
			if err != nil {
				log.Crit("GetRegion", "Error", err)
				os.Exit(1)
			}

			secretsPayload, downloadSuccess := secrets.Download(log, region)
			if !downloadSuccess {
				os.Exit(1)
			}

			_, err = os.Stdout.Write(secretsPayload.HostSecrets)
			if err != nil {
				log.Crit("os.Stdout.Write", "Error", err)
				os.Exit(1)
			}
		default:
			return false
		}
		return true
	}

	return false
}
