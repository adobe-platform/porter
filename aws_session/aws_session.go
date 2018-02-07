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
package aws_session

import (
	"os"
	"sync"
	"time"

	"github.com/adobe-platform/porter/constants"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

var (
	regionToSession     map[string]*session.Session
	regionToSessionLock sync.RWMutex
)

func STS(region, roleARN string, duration time.Duration) *session.Session {
	// clamp duration to sts:AssumeRole session length bounds
	if duration < 900*time.Second {
		duration = 900 * time.Second
	}

	if duration > 1*time.Hour {
		duration = 1 * time.Hour
	}

	stsClient := sts.New(Get(region))
	tokenCredentials := stscreds.NewCredentialsWithClient(stsClient, roleARN, func(provider *stscreds.AssumeRoleProvider) {
		provider.Duration = duration
	})

	config := aws.NewConfig()
	config.WithRegion(region)
	config.WithCredentials(tokenCredentials)
	if os.Getenv(constants.EnvDebugAws) != "" {
		config.WithLogLevel(aws.LogDebug)
	}

	roleSession := session.New(config)
	return roleSession
}

func Get(region string) (regionSession *session.Session) {
	regionToSessionLock.RLock()

	if !regionInMap(region) {
		regionToSessionLock.RUnlock()

		regionToSessionLock.Lock()
		defer regionToSessionLock.Unlock()

		// Multiple readers could be at regionToSessionLock.Lock()
		// Check condition again
		if !regionInMap(region) {
			regionSession = addSession(region)
		} else {
			// readers 2-N that were stopped at regionToSessionLock.Lock()
			// need the session
			regionSession = regionToSession[region]
		}
	} else {
		regionSession = regionToSession[region]
		regionToSessionLock.RUnlock()
	}

	return
}

func regionInMap(region string) bool {
	if regionToSession == nil {
		return false
	}

	if _, exists := regionToSession[region]; !exists {
		return false
	}

	return true
}

func addSession(region string) *session.Session {

	if regionToSession == nil {
		regionToSession = make(map[string]*session.Session)
	}

	// must alter sts config as well
	config := aws.NewConfig()
	config.WithRegion(region)
	if os.Getenv(constants.EnvDebugAws) != "" {
		config.WithLogLevel(aws.LogDebug)
	}

	regionSession := session.New(config)
	regionToSession[region] = regionSession
	return regionSession
}
