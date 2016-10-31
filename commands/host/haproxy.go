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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/files"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/stdin"
	"github.com/inconshreveable/log15"
	"github.com/phylake/go-cli"
)

type (
	HAProxyCmd struct{}

	haProxyConfigContext struct {
		ServiceName     string
		FrontEndPorts   []uint16
		HAPStdin        HAPStdin
		StatsUsername   string
		StatsPassword   string
		StatsUri        string
		IpBlacklistPath string
	}

	HAPStdin struct {
		Containers []HAPContainer `json:"containers"`
	}

	HAPContainer struct {
		Id                string `json:"id"`
		HealthCheckMethod string `json:"healthCheckMethod"`
		HealthCheckPath   string `json:"healthCheckPath"`
		HostPort          uint16 `json:"hostPort"`
	}

	hostSignal struct {
		Containers []containerSignal `json:"containers"`
	}

	containerSignal struct {
		HostPort uint16 `json:"hostPort"`
	}
)

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
    haproxy -sn <service name>

DESCRIPTION
    haproxy creates and rewrites /etc/haproxy/haproxy.cfg to work with a primary
    traffic-serving docker container. Containers can EXPOSE any port they want
    because this command inspects the published ports of the container and works
    with .porter/config to determine from which port the container wishes to
    receive internet traffic.

    This command additionally expects on STDIN the following JSON describing
    how to configure HAProxy

    {
      "containers": [
        {
          "id": "abc123",
          "healthCheckMethod": "GET",
          "healthCheckPath": "/health",
          "hostPort": 12345
        }
      ]
    }`
}

func (recv *HAProxyCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *HAProxyCmd) Execute(args []string) bool {
	if len(args) > 0 {
		var (
			stdinStruct HAPStdin
			serviceName string
		)
		log := logger.Host("cmd", "haproxy")

		flagSet := flag.NewFlagSet("", flag.ExitOnError)
		flagSet.StringVar(&serviceName, "sn", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		stdinBytes, err := stdin.GetBytes()
		if err != nil {
			log.Error("stdin.GetBytes", "Error", err)
			return false
		}
		if len(stdinBytes) == 0 {
			log.Error("Nothing on stdin")
			return false
		}

		err = json.Unmarshal(stdinBytes, &stdinStruct)
		if err != nil {
			return false
		}

		var ipBlacklistPath string
		_, err = os.Stat(constants.HAProxyIpBlacklistPath)
		if err == nil {
			ipBlacklistPath = constants.HAProxyIpBlacklistPath
		}

		context := haProxyConfigContext{
			ServiceName:     serviceName,
			FrontEndPorts:   constants.InetBindPorts,
			HAPStdin:        stdinStruct,
			StatsUsername:   constants.HAProxyStatsUsername,
			StatsPassword:   constants.HAProxyStatsPassword,
			StatsUri:        constants.HAProxyStatsUri,
			IpBlacklistPath: ipBlacklistPath,
		}

		if !hotswap(log, context) {
			os.Exit(1)
		}
		return true
	}

	return false
}

func hotswap(log log15.Logger, context haProxyConfigContext) (success bool) {

	if !healthCheckContainers(log, context.HAPStdin) {
		return
	}

	if !writeNewConfig(log, context) {
		return
	}

	if !reloadHaproxy(log) {
		return
	}

	if !signalHost(log, context) {
		return
	}

	success = true
	return
}

func writeNewConfig(log log15.Logger, context haProxyConfigContext) (success bool) {

	log.Info("writing new config")

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

	success = true
	return
}

func reloadHaproxy(log log15.Logger) (success bool) {

	pidBytes, err := ioutil.ReadFile("/var/run/haproxy.pid")
	if err != nil {
		log.Error("Couldn't read HAProxy pid file")
		return
	}
	pid := strings.TrimSpace(string(pidBytes))

	log.Info("reloading config")

	t0 := time.Now()
	err = exec.Command("service", "haproxy", "reload").Run()
	if err != nil {
		log.Error("service haproxy reload", "Error", err)
		return
	}

	// observing 60+-5s for pid to go away
	// wait 3 mins
	var i int
	for ; i < 90; i++ {

		log.Info("waiting for reload to complete")
		time.Sleep(2 * time.Second)

		_, err = os.Stat("/proc/" + pid)
		if err != nil {
			break
		}
	}

	if i == 90 {

		log.Error("previous haproxy pid is still around after 3 minutes")
		return
	} else {

		log.Info("previous haproxy pid is gone", "seconds", time.Now().Sub(t0).Seconds())
	}

	success = true
	return
}

func signalHost(log log15.Logger, context haProxyConfigContext) (success bool) {

	err := exec.Command("which", "porter_hotswap_signal").Run()
	if err != nil {
		success = true
		return
	}

	hSignal := hostSignal{}

	for _, container := range context.HAPStdin.Containers {
		if container.HostPort != 0 {

			cSignal := containerSignal{
				HostPort: container.HostPort,
			}

			hSignal.Containers = append(hSignal.Containers, cSignal)
		}
	}

	signalBytes, err := json.Marshal(hSignal)
	if err != nil {
		log.Error("json.Marshal", "Error", err)
		return
	}
	signalStr := string(signalBytes)
	log.Info("calling porter_hotswap_signal", "stdin", signalStr)

	cmd := exec.Command("porter_hotswap_signal")
	cmd.Stdin = strings.NewReader(signalStr)

	cmdComplete := make(chan struct{})
	go func(cmd *exec.Cmd) {

		err = cmd.Run()
		if err != nil {
			log.Error("porter_hotswap_signal", "Error", err)
		}

		cmdComplete <- struct{}{}
	}(cmd)

	select {
	case <-cmdComplete:
	case <-time.After(60 * time.Second):
		log.Error("porter_hotswap_signal timed out after 60 seconds")
	}

	success = true
	return
}

func healthCheckContainers(log log15.Logger, stdin HAPStdin) (success bool) {

	successChan := make(chan bool)
	for _, container := range stdin.Containers {

		go func(container HAPContainer) {

			successChan <- healthCheckContainer(log, container)

		}(container)
	}

	for i := 0; i < len(stdin.Containers); i++ {
		chanSuccess := <-successChan
		if !chanSuccess {
			return
		}
	}

	success = true
	return
}

func healthCheckContainer(log log15.Logger, container HAPContainer) (success bool) {
	log = log.New("ContainerId", container.Id)
	methodPath := container.HealthCheckMethod + " " + container.HealthCheckPath
	healthURL := fmt.Sprintf("http://127.0.0.1:%d%s", container.HostPort, container.HealthCheckPath)

	sleepDuration := 2 * time.Second
	n := int(constants.StackCreationTimeout().Seconds() / sleepDuration.Seconds())
	for i := 0; i < n; i++ {
		time.Sleep(sleepDuration)

		req, err := http.NewRequest(container.HealthCheckMethod, healthURL, nil)
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
		success = true
		break
	}

	if !success {
		log.Error("never received a 200 response for " + methodPath)
	}

	return
}
