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
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/adobe-platform/porter/daemon/flags"
)

func EnvHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	for _, env := range os.Environ() {
		w.Write([]byte(env + "\n"))
	}
}

func FlagHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	w.Write([]byte(fmt.Sprintf("environment %s\n", flags.Environment)))
}

func PanicHandler(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("panicking"))
	w.WriteHeader(500)

	go func() {

		panic("intentional panic called")
	}()
}
