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
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/adobe-platform/porter/daemon/api"
	"github.com/adobe-platform/porter/daemon/flags"
	"github.com/adobe-platform/porter/logger"
)

var packageLogger = logger.New("main")

func main() {

	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, kvp := range os.Environ() {
		packageLogger.Debug("env", "kvp", kvp)
	}

	router := api.NewRouter()

	packageLogger.Info("f5waves listing on port " + flags.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", flags.Port), router); err != nil {
		packageLogger.Crit("Failed to start service", "Error", err)
	}

}
