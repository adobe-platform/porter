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
package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/adobe-platform/porter/constants"
	"github.com/inconshreveable/log15"
)

// download docker client 1.7.1 because of this regression
// https://github.com/docker/docker/issues/15785#issuecomment-153871706
func OverrideDockerClient(log log15.Logger) (success bool) {

	defer func() {
		dockerVersion, _ := exec.Command("docker", "version").Output()
		fmt.Println("docker version")
		fmt.Println(string(dockerVersion))
	}()

	if os.Getenv(constants.EnvNoDockerOverride) != "" {
		return true
	}

	exec.Command("mkdir", "-p", constants.TempDir).Run()

	unameOutput, err := exec.Command("uname").Output()
	if err != nil {
		log.Error("uname", "Error", err)
		return
	}

	// only do override for mac
	if !strings.HasPrefix(string(unameOutput), "Darwin") {
		success = true
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Error("os.Getwd", "Error", err)
		return
	}

	path := fmt.Sprintf("%s/%s:%s", wd, constants.TempDir, os.Getenv("PATH"))
	err = os.Setenv("PATH", path)
	if err != nil {
		log.Error("os.Setenv", "Error", err)
		return
	}

	dockerBinFilePath := constants.TempDir + "/docker"

	_, err = os.Stat(dockerBinFilePath)
	if err == nil {
		success = true
		return
	}

	dockerBinFile, err := os.Create(dockerBinFilePath)
	if err != nil {
		log.Error("os.Create", "Error", err)
		return
	}
	defer dockerBinFile.Close()

	resp, err := http.Get(constants.DockerBinaryDarwinURL)
	if err != nil {
		log.Error("http.Get", "Error", err)
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(dockerBinFile, resp.Body)
	if err != nil {
		log.Error("io.Copy", "Error", err)
		return
	}

	err = exec.Command("chmod", "u+x", dockerBinFilePath).Run()
	if err != nil {
		log.Error("chmod", "Error", err)
		return
	}

	success = true
	return
}
