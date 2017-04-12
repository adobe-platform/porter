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
package host

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"text/template"
	"time"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/daemon"
	"github.com/adobe-platform/porter/daemon/flags"
	"github.com/adobe-platform/porter/logger"
	"github.com/phylake/go-cli"
)

type DaemonCmd struct{}

func (recv *DaemonCmd) Name() string {
	return "daemon"
}

func (recv *DaemonCmd) ShortHelp() string {
	return "Install porterd"
}

func (recv *DaemonCmd) LongHelp() string {
	return `NAME
    daemon -- Install porterd

SYNOPSIS
    daemon --init -e <environment> -sn <service name> -hm <health check method> -hp <health check port>
    daemon --run -e <environment> -sn <service name> -hm <health check method> -hp <health check port>

DESCRIPTION
    daemon is a host-level HTTP service

    The daemon is started automatically.

    see the readme at
    https://github.com/adobe-platform/porter/tree/master/daemon for more

OPTIONS
	--init
		Write out the init script so porterd can be managed by PID 1

	--run
		In the init script, run the daemon`
}

func (recv *DaemonCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *DaemonCmd) Execute(args []string) bool {
	if len(args) > 1 {

		switch args[0] {
		case "--init":

			var (
				environment       string
				serviceName       string
				healthCheckMethod string
				healthCheckPath   string
				elbs              string
			)

			flagSet := flag.NewFlagSet("", flag.ExitOnError)
			flagSet.StringVar(&environment, "e", "", "")
			flagSet.StringVar(&serviceName, "sn", "", "")
			flagSet.StringVar(&healthCheckMethod, "hm", "", "")
			flagSet.StringVar(&healthCheckPath, "hp", "", "")
			flagSet.StringVar(&elbs, "elbs", "", "")
			flagSet.Usage = func() {
				fmt.Println(recv.LongHelp())
			}
			flagSet.Parse(args[1:])

			context := initConfigContext{
				AwsStackId:        os.Getenv("AWS_STACKID"),
				Environment:       environment,
				ServiceName:       serviceName,
				HealthCheckMethod: strconv.Quote(healthCheckMethod),
				HealthCheckPath:   strconv.Quote(healthCheckPath),
				Elbs:              elbs,
			}

			installDaemon(context)
			return true

		case "--run":

			flagSet := flag.NewFlagSet("", flag.ContinueOnError)
			flagSet.StringVar(&flags.Environment, "e", "", "")
			flagSet.StringVar(&flags.ServiceName, "sn", "", "")
			flagSet.StringVar(&flags.HealthCheckMethod, "hm", "", "")
			flagSet.StringVar(&flags.HealthCheckPath, "hp", "", "")
			flagSet.Parse(args[1:])

			if flags.Environment == "" ||
				flags.ServiceName == "" {
				return false
			}

			daemon.Run()
			return true
		}
	}

	return false
}

type initConfigContext struct {
	Environment       string
	ServiceName       string
	HealthCheckMethod string
	HealthCheckPath   string
	Elbs              string
	AwsStackId        string
}

const porterdInitConfigTemplate = `description "porterd"
author      "Brandon Cook"

start on (local-filesystems and net-device-up)
stop on runlevel [!2345]

env ELBS={{ .Elbs }}
env AWS_STACKID={{ .AwsStackId }}
respawn
exec /usr/bin/porter host daemon --run -e {{ .Environment }} -sn {{ .ServiceName }} -hm {{ .HealthCheckMethod }} -hp {{ .HealthCheckPath }}
`

func installDaemon(context initConfigContext) {
	log := logger.Host("cmd", "daemon")
	var err error

	log.Info("installing porterd init script at " + constants.PorterDaemonInitPath)

	tmpl, err := template.New("").Parse(porterdInitConfigTemplate)
	if err != nil {
		log.Error("template parsing failed", "Error", err)
		os.Exit(1)
	}

	var porterdInitConfig bytes.Buffer

	err = tmpl.Execute(&porterdInitConfig, context)
	if err != nil {
		log.Error("template execution failed", "Error", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(constants.PorterDaemonInitPath, porterdInitConfig.Bytes(), constants.PorterDaemonInitPerms)
	if err != nil {
		log.Error("WriteFile", "Path", constants.PorterDaemonInitPath, "Error", err)
		os.Exit(1)
	}

	err = exec.Command("initctl", "reload-configuration").Run()
	if err != nil {
		log.Error("initctl reload-configuration", "Error", err)
		os.Exit(1)
	}

	err = exec.Command("initctl", "start", "porterd").Run()
	if err != nil {
		log.Error("initctl start porterd", "Error", err)
		os.Exit(1)
	}

	firstTime := true
	for {
		if firstTime {
			firstTime = false
			time.Sleep(10 * time.Millisecond)
		} else {
			time.Sleep(2 * time.Second)
		}

		healthURL := "http://127.0.0.1:" + constants.PorterDaemonBindPort + constants.PorterDaemonHealthPath
		errMsg := "GET " + healthURL

		resp, err := http.Get(healthURL)
		if err != nil {
			log.Error(errMsg, "Error", err)
			continue
		}

		if resp.StatusCode != 200 {
			log.Error(errMsg, "StatusCode", resp.StatusCode)
			continue
		}

		log.Info("successful porterd health check")
		break

	}
}
