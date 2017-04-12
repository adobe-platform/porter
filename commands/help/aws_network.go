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
package help

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/adobe-platform/porter/aws/util"
	"github.com/adobe-platform/porter/logger"
	"github.com/phylake/go-cli"
)

type IpList struct {
	SyncToken  string `json:"syncToken"`
	CreateDate string `json:"createDate"`
	Prefixes   []struct {
		IPPrefix string `json:"ip_prefix"`
		Region   string `json:"region"`
		Service  string `json:"service"`
	} `json:"prefixes"`
	Ipv6Prefixes []struct {
		Ipv6Prefix string `json:"ipv6_prefix"`
		Region     string `json:"region"`
		Service    string `json:"service"`
	} `json:"ipv6_prefixes"`
}

type AwsNetworkCmd struct{}

func (recv *AwsNetworkCmd) Name() string {
	return "aws-network"
}

func (recv *AwsNetworkCmd) ShortHelp() string {
	return "Get AWS network CIDRs by region"
}

func (recv *AwsNetworkCmd) LongHelp() string {
	return `NAME
    aws-network -- Get AWS network CIDRs by region

SYNOPSIS
    aws-network -r <region> [-s <service>]

DESCRIPTION
    Download and parse https://ip-ranges.amazonaws.com/ip-ranges.json

    Print to stdout all IPv4 CIDrs matching
    service == <service> && region == <region>

    See https://aws.amazon.com/blogs/aws/aws-ip-ranges-json/ for more

OPTIONS
    -r  AWS region

    -s  Service (defaults to AMAZON if undefined)`
}

func (recv *AwsNetworkCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *AwsNetworkCmd) Execute(args []string) bool {
	if len(args) > 0 {

		var region, service string

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)
		flagSet.StringVar(&region, "r", "", "")
		flagSet.StringVar(&service, "s", "", "")
		flagSet.Usage = func() {
			fmt.Println(recv.LongHelp())
		}
		flagSet.Parse(args)

		if !util.ValidRegion(region) {
			return false
		}

		if service == "" {
			service = "AMAZON"
		}

		log := logger.CLI()

		res, err := http.Get("https://ip-ranges.amazonaws.com/ip-ranges.json")
		if err != nil {
			log.Error("http.Get", "Error", err)
			os.Exit(1)
		}
		defer res.Body.Close()

		ipList := IpList{}

		err = json.NewDecoder(res.Body).Decode(&ipList)
		if err != nil {
			log.Error("json.Unmarshal", "Error", err)
			os.Exit(1)
		}

		for _, prefix := range ipList.Prefixes {
			if prefix.Region == region && prefix.Service == service {

				fmt.Println(prefix.IPPrefix)
			}
		}

		return true
	}

	return false
}
