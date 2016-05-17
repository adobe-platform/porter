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
package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/adobe-platform/porter/commands"
	"github.com/phylake/go-cli"
)

func main() {

	http.DefaultClient = &http.Client{
		Timeout: 20 * time.Minute,
	}

	var err error
	cliDriver := cli.New(flag.ExitOnError)

	if err = cliDriver.RegisterRoot(commands.GetRootCommand()); err != nil {
		panic(err)
	}

	if err = cliDriver.ParseInput(); err != nil {
		panic(err)
	}
}
