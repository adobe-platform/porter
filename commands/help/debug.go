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
package help

import (
	"github.com/phylake/go-cli"
	"github.com/phylake/go-cli/cmd"
)

const debugLongHelp = `These are various environment variable porter looks for to enable debugging in
the field.

Most of the DEBUG_ flags cause additional logs to be sent via stdout.

The value for these flags is a non-empty string unless otherwise noted.

DEBUG_CONFIG
    Print out configuration

    Example:
    DEBUG_CONFIG=1 porter create-stack -e dev

DEBUG_AWS
    Dump all AWS HTTP calls

LOG_DEBUG
    Turn on porter's debug logging

STACK_CREATION_TIMEOUT
    Override rollback time with a string parsed by
    https://golang.org/pkg/time/#ParseDuration

    This is clamped to the min and max duration supported by STS AssumeRole

    Example:
    export STACK_CREATION_TIMEOUT=1h
    porter create-stack -e dev

STACK_CREATION_POLL_INTERVAL
    Override the default polling for creation status interval during
    stack provisioning. The value is a string parsed by
    https://golang.org/pkg/time/#ParseDuration. The default value is 10 seconds.

NO_DOCKER_OVERRIDE
    Due to a regression in Docker, porter will attempt to download and use
    Docker client 1.7 during create-stack to work around the issue

    https://github.com/docker/docker/issues/15785#issuecomment-153871706

NO_LOG_COLOR
    The default logger has color. This option turns it off.

DEV_MODE
    Grease the wheels on development`

var Debug cli.Command

func init() {
	Debug = &cmd.Default{
		NameStr:      "debug",
		ShortHelpStr: "Activate debugging info",
		LongHelpStr:  debugLongHelp,
	}
}
