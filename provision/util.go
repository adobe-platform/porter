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
package provision

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func GetStackName(serviceName, environment string, addTimestamp bool) (stackName string, err error) {
	if serviceName == "" {
		err = errors.New("service name is empty")
		return
	}

	whoAmIBytes, err := exec.Command("whoami").Output()
	if err != nil {
		return
	}
	whoAmI := strings.TrimSpace(string(whoAmIBytes))

	if addTimestamp {
		epoch := time.Now().Unix()

		stackName = fmt.Sprintf("%s-%s-%s-%d", serviceName, environment, whoAmI, epoch)
	} else {

		stackName = fmt.Sprintf("%s-%s-%s", serviceName, environment, whoAmI)
	}

	return
}
