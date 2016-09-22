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
package provision

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/inconshreveable/log15"
)

// Package creates the service payload to deliver to S3
func Package(log log15.Logger, config *conf.Config) (success bool) {

	// clean up old artifacts before building
	exec.Command("rm", "-rf", constants.PayloadWorkingDir).Run()

	// clean up artifacts after building
	defer exec.Command("rm", "-rf", constants.PayloadWorkingDir).Run()

	exec.Command("mkdir", "-p", constants.PayloadWorkingDir).Run()

	revParseOutput, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		log.Error("git rev-parse", "Error", err)
		return
	}

	now := time.Now().Unix()
	config.ServiceVersion = strings.TrimSpace(string(revParseOutput))

	builtContainers := make(map[string]interface{})

	dockerRegistry := os.Getenv(constants.EnvDockerRegistry)
	dockerRepository := os.Getenv(constants.EnvDockerRepository)

	// This is in a loop but assumes we're building a single container
	// TODO support multiple containers
	for _, environment := range config.Environments {

		for _, region := range environment.Regions {

			for _, container := range region.Containers {

				container.OriginalName = container.Name

				// Alter the name in the config so we know which image names are part
				// of the service payload. This is important for hotswap to know which
				// of the available images on the host are the ones to be swapped in.
				if dockerRegistry == "" && dockerRepository == "" {

					container.Name = fmt.Sprintf("s3/s3/porter-%s-%d-%s",
						config.ServiceVersion, now, container.Name)
				} else {

					container.Name = fmt.Sprintf("%s/%s/porter-%s-%d-%s",
						dockerRegistry, dockerRepository,
						config.ServiceVersion, now, container.Name)
				}

				if _, exists := builtContainers[container.OriginalName]; !exists {

					imagePath := fmt.Sprintf("%s/%s.docker", constants.PayloadWorkingDir, container.Name)

					_, err := os.Stat(container.Dockerfile)
					if err != nil {
						log.Error("Dockerfile stat", "Error", err)
						return
					}

					haveBuilder := true
					_, err = os.Stat(container.DockerfileBuild)
					if err != nil {
						haveBuilder = false
					}

					if haveBuilder {
						var err error

						buildBuilderCmd := exec.Command("docker", "build", "-t", container.Name+"-builder", "-f", container.DockerfileBuild, ".")
						buildBuilderCmd.Stdout = os.Stdout
						buildBuilderCmd.Stderr = os.Stderr
						err = buildBuilderCmd.Run()
						if err != nil {
							log.Error("build Dockerfile.build", "Error", err)
							return
						}

						runCmd := exec.Command("docker", "run", "--rm", container.Name+"-builder")

						runCmdStdoutPipe, err := runCmd.StdoutPipe()
						if err != nil {
							log.Error("couldn't create StdoutPipe", "Error", err)
							return
						}

						buildCmd := exec.Command("docker", "build",
							"-t", container.Name,
							"-f", container.Dockerfile,
							"-")
						buildCmd.Stdin = runCmdStdoutPipe
						buildCmd.Stdout = os.Stdout
						buildCmd.Stderr = os.Stderr

						err = runCmd.Start()
						if err != nil {
							log.Error("docker run", "Error", err)
							return
						}

						err = buildCmd.Start()
						if err != nil {
							log.Error("build Dockerfile", "Error", err)
							return
						}

						runCmd.Wait()
						buildCmd.Wait()
					} else {
						buildCmd := exec.Command("docker", "build",
							"-t", container.Name,
							"-f", container.Dockerfile,
							".")
						buildCmd.Stdout = os.Stdout
						buildCmd.Stderr = os.Stderr
						err := buildCmd.Run()
						if err != nil {
							log.Error("build Dockerfile", "Error", err)
							return
						}
					}

					if dockerRegistry == "" {
						log.Info(fmt.Sprintf("saving docker image to %s", imagePath))

						exec.Command("mkdir", "-p", path.Dir(imagePath)).Run()

						saveCmd := exec.Command("docker", "save", "-o", imagePath, container.Name)
						saveCmd.Stdout = os.Stdout
						saveCmd.Stderr = os.Stderr
						err = saveCmd.Run()
						if err != nil {
							log.Error("docker save", "Error", err)
							return
						}
					}

					builtContainers[container.OriginalName] = nil
				}
			}
		}
	}

	if !copyPathBasedFiles(log, config) {
		return
	}

	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return
	}

	// for later build stages
	err = ioutil.WriteFile(constants.AlteredConfigPath, configBytes, 0644)
	if err != nil {
		log.Error("WriteFile", "Path", constants.AlteredConfigPath)
		return
	}

	// for the service payload about to be created
	err = ioutil.WriteFile(constants.PackPayloadConfigPath, configBytes, 0644)
	if err != nil {
		log.Error("WriteFile", "Path", constants.PackPayloadConfigPath)
		return
	}

	log.Info(fmt.Sprintf("creating service payload at %s", constants.PayloadPath))

	tarCmd := exec.Command("tar", "-C", constants.PayloadWorkingDir, "-czf", constants.PayloadPath, ".")
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	err = tarCmd.Run()
	if err != nil {
		log.Error("tar", "Error", err)
		return
	}

	success = true
	return
}

// Ensure the files that are specified with paths in the config are part of the
// temp directory which is passed between the pack and provision stages in GoCD.
// If we fetched materials in every stage then the referenced files would always
// be there, and this function wouldn't be strictly necessary.
func copyPathBasedFiles(log log15.Logger, config *conf.Config) bool {
	for _, environment := range config.Environments {
		if digest, success := digestAndCopy(log, environment.StackDefinitionPath); success {
			environment.StackDefinitionPath = digest
		} else {
			return false
		}

		for _, region := range environment.Regions {
			if digest, success := digestAndCopy(log, region.StackDefinitionPath); success {
				region.StackDefinitionPath = digest
			} else {
				return false
			}
		}
	}

	return true
}

func digestAndCopy(log log15.Logger, filePath string) (string, bool) {
	if filePath == "" {
		return "", true
	}

	log = log.New("Filepath", filePath)

	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Error("ioutil.ReadFile", "Error", err)
		return "", false
	}
	digestArray := md5.Sum(fileBytes)
	digest := hex.EncodeToString(digestArray[:])

	newFilePath := constants.TempDir + "/" + digest
	err = ioutil.WriteFile(newFilePath, fileBytes, 0644)
	if err != nil {
		log.Error("ioutil.ReadFile", "Error", err)
		return "", false
	}

	return newFilePath, true
}
