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
package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/adobe-platform/porter/constants"
	"gopkg.in/inconshreveable/log15.v2"
)

// Do our best to add the porter temporary directory to .gitignore
func GitIgnoreTempDir(log log15.Logger) {
	var err error

	_, err = os.Stat(".git")
	if err != nil {
		// no point if there's no repo
		return
	}

	addTempDir(log, ".gitignore")
}

// Do our best to add the porter temporary directory to .dockerignore
func DockerIgnoreTempDir(log log15.Logger) {
	addTempDir(log, ".dockerignore")
}

func addTempDir(log log15.Logger, filePath string) {
	var err error

	_, err = os.Stat(constants.PorterDir)
	if err != nil {
		// must not be in a porter project
		return
	}

	var file *os.File

	_, err = os.Stat(filePath)
	if err == nil {

		file, err = os.OpenFile(filePath, os.O_RDWR, 0666)
		if err != nil {
			return
		}
	} else {

		log.Info(fmt.Sprintf("Creating %s file", filePath))
		file, err = os.Create(filePath)
		if err != nil {
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), constants.TempDir) {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		return
	}

	_, err = file.WriteString(constants.TempDir + "\n")
	if err != nil {
		return
	}

	log.Info(fmt.Sprintf("Appended %s to %s", constants.TempDir, filePath))
}
