/*
 * (c) 2016-2018 Adobe. All rights reserved.
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
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/phylake/go-cli"
)

type SvcPayloadCmd struct{}

func (recv *SvcPayloadCmd) Name() string {
	return "svc-payload"
}

func (recv *SvcPayloadCmd) ShortHelp() string {
	return "download/verify service payload"
}

func (recv *SvcPayloadCmd) LongHelp() string {
	return `NAME
    svc-payload -- download/verify service payload

SYNOPSIS
    svc-payload --get -b <bucket> -k <key> -s <sum> -l <path> -r <region>

DESCRIPTION
    svc-payload downloads and verifies the integrity of the service payload

OPTIONS
    -b  S3 Bucket

    -k  S3 Key

    -s  SHA256 to verify

    -l  Location on the filesystem to download to

    -r  AWS region`
}

func (recv *SvcPayloadCmd) SubCommands() []cli.Command {
	return nil
}

func (recv *SvcPayloadCmd) Execute(args []string) bool {
	if len(args) > 0 {
		switch args[0] {
		case "--get":

			log := logger.Host("cmd", "svc-payload")

			var err error
			var bucketFlag, locationFlag, keyFlag, sumFlag, regionFlag string
			flagSet := flag.NewFlagSet("", flag.ExitOnError)
			flagSet.StringVar(&bucketFlag, "b", "", "")
			flagSet.StringVar(&keyFlag, "k", "", "")
			flagSet.StringVar(&sumFlag, "s", "", "")
			flagSet.StringVar(&regionFlag, "r", "", "")
			flagSet.StringVar(&locationFlag, "l", "", "")
			flagSet.Usage = func() {
				fmt.Println(recv.LongHelp())
			}
			flagSet.Parse(args[1:])

			if len(sumFlag) != 64 {
				return false
			}

			expectedChecksum, err := hex.DecodeString(sumFlag)
			if err != nil {
				log.Crit("hex.Decode", "Error", err)
				os.Exit(1)
			}

			exec.Command("mkdir", "-p", filepath.Dir(locationFlag)).Run()

			if _, err = os.Stat(locationFlag); err == nil {
				log.Info("service payload exists")
				os.Exit(0)
			}

			log.Info("downloading/verifying service payload",
				"Location", locationFlag)

			s3Client := s3manager.NewDownloader(aws_session.Get(regionFlag))
			s3Client.Concurrency = runtime.GOMAXPROCS(-1) // read, don't set, the value

			getObjectInput := &s3.GetObjectInput{
				Bucket: aws.String(bucketFlag),
				Key:    aws.String(keyFlag),
			}

			var payloadFile *os.File

			retryMsg := func(i int) { log.Warn("Service payload download retrying", "Count", i) }
			if !util.SuccessRetryer(7, retryMsg, func() bool {

				// a partial failure in download could leave us with partial
				// content in this file
				//
				// create it every time
				payloadFile, err = os.Create(locationFlag)

				// an error here is likely permissions related and not worth
				// retrying
				if err != nil {
					log.Crit("os.Create", "Error", err)
					os.Exit(1)
				}

				_, err = s3Client.Download(payloadFile, getObjectInput)
				if err != nil {

					payloadFile.Close()

					log.Error("S3 download", "Error", err)
					return false
				}

				return true
			}) {
				log.Crit("Failed to download service payload")
				os.Exit(1)
			}
			defer payloadFile.Close()

			_, err = payloadFile.Seek(0, 0)
			if err != nil {
				log.Crit("file.Seek", "Error", err)
				os.Exit(1)
			}

			payloadBytes, err := ioutil.ReadAll(payloadFile)
			if err != nil {
				log.Crit("ioutil.ReadAll", "Error", err)
				os.Exit(1)
			}

			actualChecksumArray := sha256.Sum256(payloadBytes)
			actualChecksum := actualChecksumArray[:]

			if !bytes.Equal(actualChecksum, expectedChecksum) {
				log.Crit("Checksums don't match",
					"Expected", sumFlag,
					"Actual", hex.EncodeToString(actualChecksum))
				os.Exit(1)
			}

		default:
			return false
		}
		return true
	}

	return false
}
