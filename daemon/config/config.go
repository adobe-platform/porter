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
package config

import (
	"net/http"
	"os"
	"time"
)

const (
	defaultTimeoutSecs = 10
)

var (
	CtxTimeout time.Duration
)

// This IS NOT init() because it needs to be called explicity when the daemon
// starts, otherwise it applies to all of porter which is undesirable
func Init() {
	timeout, err := time.ParseDuration(os.Getenv("CONTEXT_TIMEOUT"))
	// timeout is a minimum of defaultTimeoutSecs seconds
	if err != nil || timeout <= defaultTimeoutSecs*time.Second {
		CtxTimeout = defaultTimeoutSecs * time.Second
	} else {
		CtxTimeout = timeout
	}

	// exceeding the http client timeout is a problem with us as a client, not
	// our client connecting too slowly
	http.DefaultClient = &http.Client{
		Timeout: CtxTimeout - (1 * time.Second),
	}
}
