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
package daemon

import (
	"net/http"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon/api"
	"github.com/adobe-platform/porter/daemon/config"
	"github.com/adobe-platform/porter/daemon/elb_registration"
	"github.com/adobe-platform/porter/daemon/wait_handle"
	"github.com/adobe-platform/porter/logger"
)

func Run() {
	config.Init()

	go wait_handle.Call()
	go elb_registration.Call()

	log := logger.Daemon()

	router := api.NewRouter()

	log.Info("porterd listing on port " + constants.PorterDaemonBindPort)
	if err := http.ListenAndServe(":"+constants.PorterDaemonBindPort, router); err != nil {
		log.Crit("Failed to start service", "Error", err)
	}
}
