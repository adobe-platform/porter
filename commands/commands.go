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
package commands

//-not using this anymore- go:generate go run gen/docs.go

import (
	"fmt"

	"github.com/adobe-platform/porter/commands/bootstrap"
	"github.com/adobe-platform/porter/commands/build"
	"github.com/adobe-platform/porter/commands/dev"
	"github.com/adobe-platform/porter/commands/help"
	"github.com/adobe-platform/porter/commands/host"
	"github.com/adobe-platform/porter/constants"
	"github.com/phylake/go-cli"
	"github.com/phylake/go-cli/cmd"
)

const porterDescription = `Usage: porter COMMAND [OPTIONS]

porter is a platform built on AWS APIs to enable continuous delivery of docker
containers to EC2 instances across AWS regions.`

func GetRootCommand() cli.Command {
	return &cmd.Root{
		Help: porterDescription,
		SubCommandList: []cli.Command{
			&cmd.Default{
				NameStr:      "bootstrap",
				ShortHelpStr: "Bootstrap AWS resources",
				LongHelpStr: `There are various statically defined resources that porter can be configured to
use. These commands help rapidly bootstrap AWS resources and prevent a lot of
error-prone clicking around the AWS console.`,
				SubCommandList: []cli.Command{
					&bootstrap.IamCmd{},
					&bootstrap.ElbCmd{},
					&bootstrap.S3Cmd{},
				},
			},
			// &dev.UpdateCLICmd{},
			&dev.CreateStackCmd{},
			&dev.SyncStackCmd{},
			&cmd.Default{
				NameStr:      "host",
				ShortHelpStr: "EC2 host commands",
				LongHelpStr: `Commands that run on an EC2 host.

These likely won't work locally or on a build box but they can be used to find
out more about host-level services.

These commands log to STDOUT and primarily run from files/cloud-init.yaml`,
				SubCommandList: []cli.Command{
					&host.HAProxyCmd{},
					&host.RsyslogCmd{},
					&host.DockerCmd{},
					&host.DaemonCmd{},
					&host.SecretsCmd{},
					&host.SvcPayloadCmd{},
					&host.SignalCmd{},
				},
			},
			&cmd.Default{
				NameStr:      "build",
				ShortHelpStr: "build machine commands",
				LongHelpStr:  `Commands that run on a build box to deploy services.`,
				SubCommandList: []cli.Command{
					&build.PackCmd{},
					&build.ProvisionStackCmd{},
					&build.PromoteCmd{},
					&build.PruneCmd{},
					&build.HookCmd{},
					// &build.HotSwapCmd{},
					&build.CleanCmd{},
					&build.NotifyCmd{},
					&cmd.Default{
						NameStr:      "skms",
						ShortHelpStr: "DEPRECATED",
						ExecuteFunc: func(args []string) bool {
							return true
						},
					},
				},
			},
			&cmd.Default{
				NameStr:      "help",
				ShortHelpStr: "General help",
				LongHelpStr:  "General help and documentation not tied to a specific command",
				SubCommandList: []cli.Command{
					help.Debug,
					help.Issue,
					&help.AwsNetworkCmd{},
				},
			},
			&cmd.Default{
				NameStr:      "version",
				ShortHelpStr: "Print the current version",
				ExecuteFunc: func(args []string) bool {
					fmt.Println(constants.Version)
					return true
				},
			},
		},
	}
}
