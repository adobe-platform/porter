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
	"strings"
	"text/template"
	"time"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/files"
	"github.com/adobe-platform/porter/logger"
	"github.com/phylake/go-cli"
)

type HAProxyCmd struct{}

func (recv *HAProxyCmd) Name() string {
	return "haproxy"
}

func (recv *HAProxyCmd) ShortHelp() string {
	return "Manipulate haproxy configuration"
}

func (recv *HAProxyCmd) LongHelp() string {
	return `NAME
    haproxy -- Manipulate haproxy configuration

SYNOPSIS
    haproxy -sn <service name> -hm <health check method> -hp <health check path>

DESCRIPTION
    haproxy creates and rewrites /etc/haproxy/haproxy.cfg to work with a primary
    traffic-serving docker container. Containers can EXPOSE any port they want
    because this command inspects the published ports of the container and works
    with .porter/config to determine from which port the container wishes to
    receive internet traffic.

    This command additionally expects on STDIN a host port to route internet
    traffic to.`
}

func (recv *HAProxyCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *HAProxyCmd) Execute(args []string) bool {
	if len(args) > 0 {
		var (
			servicePort       int
			serviceName       string
			healthCheckMethod string
			healthCheckPath   string
		)

		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&serviceName, "sn", "", "")
		flagSet.StringVar(&healthCheckMethod, "hm", "", "")
		flagSet.StringVar(&healthCheckPath, "hp", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		stat, err := os.Stdin.Stat()
		if err != nil {
			return false
		}

		// nothing on STDIN
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return false
		}

		servicePortBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return false
		}

		servicePortString := strings.TrimSpace(string(servicePortBytes))
		servicePort, err = strconv.Atoi(servicePortString)
		if err != nil {
			return false
		}

		var ipBlacklistPath string
		_, err = os.Stat(constants.HAProxyIpBlacklistPath)
		if err == nil {
			ipBlacklistPath = constants.HAProxyIpBlacklistPath
		}

		context := haProxyConfigContext{
			BackendName:       serviceName,
			FrontEndPorts:     constants.InetBindPorts,
			ContainerPort:     uint16(servicePort),
			HealthCheckMethod: healthCheckMethod,
			HealthCheckPath:   healthCheckPath,
			StatsUsername:     constants.HAProxyStatsUsername,
			StatsPassword:     constants.HAProxyStatsPassword,
			StatsUri:          constants.HAProxyStatsUri,
			IpBlacklistPath:   ipBlacklistPath,
		}

		if !rewriteConfig(context) {
			os.Exit(1)
		}
		return true
	}

	return false
}

type haProxyConfigContext struct {
	BackendName       string
	FrontEndPorts     []uint16
	ContainerPort     uint16
	HealthCheckMethod string
	HealthCheckPath   string
	StatsUsername     string
	StatsPassword     string
	StatsUri          string
	IpBlacklistPath   string
}

func rewriteConfig(context haProxyConfigContext) (success bool) {
	log := logger.Host("cmd", "haproxy")

	successfulHealthCheck := false
	methodPath := context.HealthCheckMethod + " " + context.HealthCheckPath

	sleepDuration := 2 * time.Second
	n := int(constants.StackCreationTimeout().Seconds() / sleepDuration.Seconds())
	for i := 0; i < n; i++ {
		time.Sleep(sleepDuration)

		healthURL := fmt.Sprintf("http://localhost:%d%s", context.ContainerPort, context.HealthCheckPath)

		req, err := http.NewRequest(context.HealthCheckMethod, healthURL, nil)
		if err != nil {
			log.Warn("http.NewRequest", "Error", err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			log.Warn(methodPath, "Error", err)
			continue
		}

		if resp.StatusCode != 200 {
			log.Warn(methodPath, "StatusCode", resp.StatusCode)
			continue
		}

		log.Info("successful health check on container. rewriting haproxy config")
		successfulHealthCheck = true
		break

	}

	if !successfulHealthCheck {
		log.Error("never received a 200 response for " + methodPath)
		return
	}

	tmpl, err := template.New("").Parse(files.HaproxyCfg)
	if err != nil {
		log.Error("template parsing failed", "Error", err)
		return
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, context)
	if err != nil {
		log.Error("template execution failed", "Error", err)
		return
	}

	err = ioutil.WriteFile(constants.HAProxyConfigPath, buf.Bytes(), constants.HAProxyConfigPerms)
	if err != nil {
		log.Error("WriteFile failed", "Path", constants.HAProxyConfigPath, "Error", err)
		return
	}

	err = exec.Command("service", "haproxy", "reload").Run()
	if err != nil {
		log.Error("service haproxy reload", "Error", err)
		return
	}

	success = true
	return
}
