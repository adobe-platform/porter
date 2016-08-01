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
package logger

import (
	"io"
	"log/syslog"
	"os"
	"os/exec"
	"strings"

	"github.com/adobe-platform/porter/constants"
	"github.com/inconshreveable/log15"
	"github.com/onsi/ginkgo"
)

var cliLog = log15.New("porter_version", constants.Version)
var hostLog = log15.New("porter_version", constants.Version)

func init() {
	out, err := exec.Command("uname").Output()
	if err == nil && strings.TrimSpace(string(out)) == "Linux" {
		initHostLog()
	}

	var logFmt log15.Format
	if os.Getenv(constants.EnvNoLogColor) == "" {
		logFmt = log15.TerminalFormat()
	} else {
		logFmt = log15.LogfmtFormat()
	}

	if os.Getenv("TEST") == "true" {

		handler := log15.StreamHandler(ginkgo.GinkgoWriter, logFmt)
		handler = log15.LvlFilterHandler(log15.LvlError, handler)
		handler = log15.CallerStackHandler("%+v", handler)

		log15.Root().SetHandler(handler)
	} else {
		// the default logging format is best for logging to stdout
		addStackTraceLogging(cliLog, os.Stdout, logFmt)
	}
}

func initHostLog() {

	writer, err := syslog.Dial("udp", "localhost:514", syslog.LOG_DAEMON|syslog.LOG_INFO, constants.ProgramName)
	if err != nil {
		panic(err)
	}

	addStackTraceLogging(hostLog, writer, log15.LogfmtFormat())
}

func CLI(kvps ...interface{}) log15.Logger {
	return cliLog.New(kvps...)
}

func Host(kvps ...interface{}) log15.Logger {
	return hostLog.New(kvps...)
}

func Daemon(kvps ...interface{}) log15.Logger {
	kvps = append(kvps, "service", "porterd")
	return Host(kvps...)
}

func addStackTraceLogging(log log15.Logger, writer io.Writer, logFmt log15.Format) {
	/*
		Log stack traces for LvlCrit, LvlError, and LvlWarn
		to help us debug issues in the wild

		const (
			LvlCrit Lvl = iota
			LvlError
			LvlWarn
			LvlInfo
			LvlDebug
		)
	*/
	stackHandler := log15.StreamHandler(writer, logFmt)
	stackHandler = log15.CallerStackHandler("%+v", stackHandler)
	// put filter last because it will be run first
	stackHandler = log15.FilterHandler(func(r *log15.Record) bool {
		return r.Lvl <= log15.LvlWarn
	}, stackHandler)

	infoHandler := log15.StreamHandler(writer, logFmt)
	if os.Getenv(constants.EnvLogDebug) == "" {
		infoHandler = log15.FilterHandler(func(r *log15.Record) bool {
			return r.Lvl <= log15.LvlInfo
		}, infoHandler)
	} else {
		infoHandler = log15.FilterHandler(func(r *log15.Record) bool {
			return r.Lvl <= log15.LvlDebug
		}, infoHandler)
	}

	log.SetHandler(log15.MultiHandler(stackHandler, infoHandler))
}
