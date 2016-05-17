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
package http

import (
	"net/http"
)

func S400(w http.ResponseWriter) {
	http.Error(w, http.StatusText(400), 400)
}

func S401(w http.ResponseWriter) {
	http.Error(w, http.StatusText(401), 401)
}

func S404(w http.ResponseWriter) {
	http.Error(w, http.StatusText(404), 404)
}

func S408(w http.ResponseWriter) {
	http.Error(w, http.StatusText(408), 408)
}

func S500(w http.ResponseWriter) {
	http.Error(w, http.StatusText(500), 500)
}
