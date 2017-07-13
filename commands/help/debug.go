/*
 * (c) 2016-2017 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package help

import (
	"github.com/phylake/go-cli"
	"github.com/phylake/go-cli/cmd"
)

const debugLongHelp = `These are various environment variable porter looks for to enable debugging in
the field.

The DEBUG_ flags cause additional logs to be sent via stdout.

The value for these flags is a non-empty string unless otherwise noted.

DEBUG_CONFIG

    Print out configuration

    Example:
    DEBUG_CONFIG=1 porter create-stack -e dev

DEBUG_AWS

    Dump all AWS HTTP calls

LOG_DEBUG

    Turn on porter's debug logging

LOG_COLOR

    Turn on log colors

STACK_CREATION_TIMEOUT

    Override rollback time with a string parsed by time.ParseDuration()
    https://golang.org/pkg/time/#ParseDuration

    This is clamped to the min and max duration supported by sts:AssumeRole

    Example:
    export STACK_CREATION_TIMEOUT=1h
    porter create-stack -e dev

STACK_CREATION_POLL_INTERVAL

    Override the default polling for creation status interval during
    stack provisioning. The value is a string parsed by time.ParseDuration()
    The default value is 10 seconds.

    https://golang.org/pkg/time/#ParseDuration

STACK_CREATION_ON_FAILURE

    By default porter calls cloudformation:CreateStack with OnFailure = DELETE

    Set this to DO_NOTHING or ROLLBACK to change this behavior. ROLLBACK is
    often useful for examining CloudFormation events to see if resources failed
    to provision. DO_NOTHING can give you time to login to an EC2 host if the
    WaitCondition failed to complete

    http://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_CreateStack.html

DEV_MODE

    Grease the wheels on development

VOLUME_FLAG
    
    By default the repo's root is volume mapped to /repo_root with no volume mounting options.
    This environment variable now allows the volume mount to be either shared as:
        a. Read only when VOLUME_FLAG is set to 'ro'.
        b. SELinux write compatible with labels preserved when VOLUME_FLAG is set to 'z'`

var Debug cli.Command

func init() {
	Debug = &cmd.Default{
		NameStr:      "debug",
		ShortHelpStr: "Activate debugging info",
		LongHelpStr:  debugLongHelp,
	}
}
