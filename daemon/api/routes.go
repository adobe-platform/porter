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
package api

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon/middleware"
	"github.com/adobe-platform/porter/logger"
	"github.com/julienschmidt/httprouter"
)

func NewRouter() *httprouter.Router {
	router := httprouter.New()

	middlewares := []func(middleware.Handle) middleware.Handle{
		middleware.VersionHeader, middleware.Profile,
	}

	//
	// Health
	//
	createRoute(router.GET, constants.PorterDaemonHealthPath, healthHandler, middlewares...)

	//
	// Testing
	//
	router.HandlerFunc("POST", "/panic", PanicHandler)

	//
	// AWS info
	//
	createRoute(router.GET, "/aws/ec2/tags", EC2TagsHandler, middlewares...)
	createRoute(router.GET, "/aws/region", RegionHandler, middlewares...)

	//
	// Introspection and profiling
	//
	createRoute(router.GET, "/env", EnvHandler, middlewares...)
	createRoute(router.GET, "/flag", FlagHandler, middlewares...)

	addProfiling(router)

	return router
}

func createRoute(method func(path string, handle httprouter.Handle),
	path string,
	handler middleware.Handle,
	middlewares ...func(middleware.Handle) middleware.Handle) {

	for _, m := range middlewares {
		handler = m(handler)
	}

	log := logger.Daemon("package", "api")

	routeHandle := middleware.Context(path, log, handler)
	method(path, routeHandle)
}

func healthHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// default is a 200 response
}

func addProfiling(router *httprouter.Router) {
	router.HandlerFunc("GET", "/debug/pprof", pprof.Index)
	router.HandlerFunc("GET", "/debug/pprof/cmdline", pprof.Cmdline)
	router.HandlerFunc("GET", "/debug/pprof/profile", pprof.Profile)
	router.HandlerFunc("GET", "/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc("POST", "/debug/pprof/symbol", pprof.Symbol)
	router.Handler("GET", "/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handler("GET", "/debug/pprof/heap", pprof.Handler("heap"))
	router.Handler("GET", "/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handler("GET", "/debug/pprof/block", pprof.Handler("block"))
}
