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

const issueLongHelp = `Please report all bugs, enhancement requests, etc. at
https://github.com/adobe-platform/porter/issues

We would appreciate it if you take the time to see if your issue has already
been filed before creating a new one.`

var Issue cli.Command

func init() {
	Issue = &cmd.Default{
		NameStr:      "issue",
		ShortHelpStr: "How to report an issue",
		LongHelpStr:  issueLongHelp,
	}
}
