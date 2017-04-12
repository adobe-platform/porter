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
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/files"
	"github.com/adobe-platform/porter/logger"
	"github.com/phylake/go-cli"
)

type RsyslogCmd struct{}

func (recv *RsyslogCmd) Name() string {
	return "rsyslog"
}

func (recv *RsyslogCmd) ShortHelp() string {
	return "Manipulate rsyslog configuration"
}

func (recv *RsyslogCmd) LongHelp() string {
	return fmt.Sprintf(`NAME
    rsyslog -- Manipulate rsyslog configuration

SYNOPSIS
    rsyslog --init

DESCRIPTION
    rsyslog configures the rsyslog daemon with %s and
    %s

    porter's rsyslog config allows logging on TCP and UDP port 514

    Services should log on daemon.* in order to route logs to
    /var/log/porter.log which is monitored by splunkd for forwarding`, constants.RsyslogConfigPath, constants.RsyslogPorterConfigPath)
}

func (recv *RsyslogCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *RsyslogCmd) Execute(args []string) bool {
	if len(args) == 1 && args[0] == "--init" {
		initRsyslog()
		return true
	}

	return false
}

func initRsyslog() {
	log := logger.Host("cmd", "rsyslog")
	var err error

	log.Info("writing configuration at " + constants.RsyslogConfigPath)
	err = ioutil.WriteFile(constants.RsyslogConfigPath, []byte(files.RsyslogConf), constants.RsyslogConfigPerms)
	if err != nil {
		log.Error("WriteFile", "Path", constants.RsyslogConfigPath, "Error", err)
		return
	}

	log.Info("writing configuration at " + constants.RsyslogPorterConfigPath)
	err = ioutil.WriteFile(constants.RsyslogPorterConfigPath, []byte(files.RsyslogPorterConf), constants.RsyslogConfigPerms)
	if err != nil {
		log.Error("WriteFile", "Path", constants.RsyslogPorterConfigPath, "Error", err)
		return
	}

	log.Info("restarting rsyslog with new config")
	err = exec.Command("service", "rsyslog", "restart").Run()
	if err != nil {
		log.Error("failed to restart rsyslog", "Error", err)
		return
	}
}
