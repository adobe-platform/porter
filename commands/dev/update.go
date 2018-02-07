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
package dev

import (
	"github.com/phylake/go-cli"
)

type UpdateCLICmd struct{}

func (recv *UpdateCLICmd) Name() string {
	return "update"
}

func (recv *UpdateCLICmd) ShortHelp() string {
	return "Update the CLI"
}

func (recv *UpdateCLICmd) LongHelp() string {
	return `Downloads the latest version of the CLI`
}

func (recv *UpdateCLICmd) SubCommands() []cli.Command {
	return nil
}

func (recv *UpdateCLICmd) Execute(args []string) bool {
	// TODO
	return false
}
