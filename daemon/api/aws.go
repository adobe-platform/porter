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
package api

import (
	"encoding/json"
	"net/http"

	. "github.com/adobe-platform/porter/daemon/http"

	"github.com/adobe-platform/porter/daemon/identity"
	"github.com/adobe-platform/porter/daemon/middleware"
	"golang.org/x/net/context"
)

func EC2TagsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log := middleware.GetRequestLog(ctx)

	ii, err := identity.Get(log)
	if err != nil {
		S500(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ii.Tags); err != nil {
		log.Error("json.NewEncoder(w).Encode", "Error", err)
		S500(w)
	}
}

func RegionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	log := middleware.GetRequestLog(ctx)

	ii, err := identity.Get(log)
	if err != nil {
		S500(w)
		return
	}

	w.Write([]byte(ii.AwsCreds.Region))
}
