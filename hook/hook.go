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
package hook

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"

	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/provision_output"
	"github.com/inconshreveable/log15"
)

type (
	Opts struct {
		BuildStdout io.Writer
		BuildStderr io.Writer

		RunStdout io.Writer
		RunStderr io.Writer
	}
)

// Multi-region deployment means we need an additional unique id for git clones
// and image names
var globalCounter *uint32 = new(uint32)

func newOpts() *Opts {
	return &Opts{
		BuildStdout: os.Stdout,
		BuildStderr: os.Stderr,

		RunStdout: os.Stdout,
		RunStderr: os.Stderr,
	}
}

func Execute(log log15.Logger,
	hookName, environment string,
	provisionedRegions []provision_output.Region,
	fs ...func(*Opts)) (success bool) {

	var err error

	log = log.New("HookName", hookName)
	log.Info("Hook BEGIN")

	config, configSuccess := conf.GetConfig(log, false)
	if !configSuccess {
		return
	}

	opts := newOpts()
	for _, f := range fs {
		f(opts)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		log.Error("Getwd", "Error", err)
		return
	}
	log.Debug("os.Getwd()", "Path", workingDir)

	var configHooks []conf.Hook

	configHooks = config.Hooks[hookName]
	if configHooks == nil {
		log.Error("Hook is undefined")
		return
	}

	if environment == "" {

		runArgs := runArgsFactory(log, config, workingDir)

		if !runConfigHooks(log, config, hookName, configHooks, runArgs, opts) {
			return
		}

	} else {

		env, err := config.GetEnvironment(environment)
		if err != nil {
			log.Error("GetEnvironment", "Error", err)
			return
		}

		// This applies to pre-provision which doesn't have provisioning output
		// since the provision command hasn't run but should work multi-region
		// in the same way that post-provision and others behave
		if provisionedRegions == nil {
			provisionedRegions = make([]provision_output.Region, 0)

			for _, region := range env.Regions {
				pr := provision_output.Region{
					AWSRegion: region.Name,
				}
				provisionedRegions = append(provisionedRegions, pr)
			}
		}

		for _, pr := range provisionedRegions {

			elbDNS := ""

			region, err := env.GetRegion(pr.AWSRegion)
			if err != nil {
				log.Error("GetRegion", "Error", err)
				return
			}

			roleARN, err := env.GetRoleARN(region.Name)
			if err != nil {
				log.Error("GetRoleARN", "Error", err)
				return
			}

			roleSession := aws_session.STS(region.Name, roleARN, 3600)

			if pr.ProvisionedELBName != "" {
				elbClient := elb.New(roleSession)

				log.Info("DescribeLoadBalancers")
				output, err := elb.DescribeLoadBalancers(elbClient, pr.ProvisionedELBName)
				if err != nil {
					log.Error("DescribeLoadBalancers", "Error", err)
					return
				}
				if len(output) == 1 {
					elbDNS = *output[0].DNSName
				} else {
					log.Warn("DescribeLoadBalancers - no ELB found")
				}
			}

			credValue, err := roleSession.Config.Credentials.Get()
			if err != nil {
				log.Warn("Couldn't get AWS credential values. Hooks calling AWS APIs will fail")
			}

			runArgs := runArgsFactory(log, config, workingDir)

			runArgs = append(runArgs,
				"-e", "PORTER_ENVIRONMENT="+environment,
				"-e", "AWS_REGION="+region.Name,
				// AWS_DEFAULT_REGION is also needed for AWS SDKs
				"-e", "AWS_DEFAULT_REGION="+region.Name,
				"-e", "AWS_ACCESS_KEY_ID="+credValue.AccessKeyID,
				"-e", "AWS_SECRET_ACCESS_KEY="+credValue.SecretAccessKey,
				"-e", "AWS_SESSION_TOKEN="+credValue.SessionToken,
				"-e", "AWS_SECURITY_TOKEN="+credValue.SessionToken,
			)

			if elbDNS != "" {
				runArgs = append(runArgs,
					"-e", "AWS_ELASTICLOADBALANCING_LOADBALANCER_DNS="+elbDNS)
			}

			if pr.StackId != "" {
				runArgs = append(runArgs,
					"-e", "AWS_CLOUDFORMATION_STACKID="+pr.StackId)
			}

			if !runConfigHooks(log, config, hookName, configHooks, runArgs, opts) {
				return
			}
		}
	}

	log.Info("Hook END")

	success = true
	return
}

func runArgsFactory(log log15.Logger, config *conf.Config, workingDir string) []string {
	runArgs := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:%s", workingDir, "/repo_root"),
		"-e", "PORTER_SERVICE_NAME=" + config.ServiceName,
		"-e", "DOCKER_ENV_FILE=" + constants.EnvFile,
		"-e", "HAPROXY_STATS_USERNAME=" + constants.HAProxyStatsUsername,
		"-e", "HAPROXY_STATS_PASSWORD=" + constants.HAProxyStatsPassword,
		"-e", "HAPROXY_STATS_URL=" + constants.HAProxyStatsUrl,
	}

	revParseOutput, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err == nil {
		sha1 := strings.TrimSpace(string(revParseOutput))

		runArgs = append(runArgs, "-e", "PORTER_SERVICE_VERSION="+sha1)
	}

	var warnedDeprecation bool
	for _, kvp := range os.Environ() {
		if strings.HasPrefix(kvp, "PORTER_") {
			if !warnedDeprecation {
				warnedDeprecation = true
				log.Warn("Hook environments configured with PORTER_ is deprecated. In future releases and this will be an error http://bit.ly/2ar6fcQ")
			}
			log.Debug("Deprecated environment", "Env", kvp)
			runArgs = append(runArgs, "-e", strings.TrimPrefix(kvp, "PORTER_"))
		}
	}

	return runArgs
}

type hookLinkedList struct {
	hook      conf.Hook
	hookIndex int

	next *hookLinkedList
}

func runConfigHooks(log log15.Logger, config *conf.Config, hookName string,
	hooks []conf.Hook, runArgs []string, opts *Opts) (success bool) {

	successChan := make(chan bool)
	var (
		concurrentCount int
		configSuccess   bool

		head *hookLinkedList
		tail *hookLinkedList
	)

	// Only retained lexical scope is successChan, everything else is copied
	goGadgetHook := func(log log15.Logger, config conf.Config, hookName string,
		hookIndex int, hook conf.Hook, runArgs []string, opts *Opts) {

		successChan <- runConfigHook(log, config, hookName, hookIndex, hook, runArgs, opts)
	}

	for hookIndex, hook := range hooks {
		next := &hookLinkedList{
			hook:      hook,
			hookIndex: hookIndex,
		}

		if head == nil {
			head = next
		}

		if tail == nil {
			tail = head
		} else {
			tail.next = next
			tail = next
		}
	}

	for node := head; node != nil; node = node.next {

		if node.hook.Concurrent {

			concurrentCount++
		} else {

			concurrentCount = 1
		}

		log := log.New("HookIndex", node.hookIndex, "Concurrent", node.hook.Concurrent)
		log.Debug("go go gadget hook")
		go goGadgetHook(log, *config, hookName, node.hookIndex, node.hook, runArgs, opts)

		// block anytime we're not running consecutive concurrent hooks
		if node.next == nil || !node.next.hook.Concurrent {

			log.Debug("Waiting for hook(s) to finish", "concurrentCount", concurrentCount)
			for i := 0; i < concurrentCount; i++ {
				success = <-successChan
				if !success {
					return
				}
			}
			log.Debug("Hook(s) finished", "concurrentCount", concurrentCount)

			config, configSuccess = conf.GetConfig(log, false)
			if !configSuccess {
				return
			}

			concurrentCount = 0
		}
	}

	success = true
	return
}

func runConfigHook(log log15.Logger, config conf.Config, hookName string,
	hookIndex int, hook conf.Hook, runArgs []string, opts *Opts) (success bool) {
	log.Debug("runConfigHook() BEGIN")
	defer log.Debug("runConfigHook() END")

	for envKey, envValue := range hook.Environment {
		if envValue == "" {
			envValue = os.Getenv(envKey)
		}
		runArgs = append(runArgs, "-e", envKey+"="+envValue)
		log.Debug("Configured environment", "Key", envKey, "Value", envValue)
	}

	dockerFilePath := hook.Dockerfile
	hookCounter := atomic.AddUint32(globalCounter, 1)

	if hook.Repo != "" {

		repoDir := fmt.Sprintf("%s_clone_%d_%d", hookName, hookIndex, hookCounter)
		repoDir = path.Join(constants.TempDir, repoDir)

		defer exec.Command("rm", "-fr", repoDir).Run()

		log.Info("git clone",
			"Repo", hook.Repo,
			"Ref", hook.Ref,
			"Directory", repoDir,
		)

		cloneCmd := exec.Command(
			"git", "clone",
			"--branch", hook.Ref, // this works on tags as well
			"--depth", "1",
			hook.Repo, repoDir,
		)
		cloneCmd.Stderr = os.Stderr
		err := cloneCmd.Run()
		if err != nil {
			log.Error("git clone", "Error", err)
			return
		}

		// Validation ensures that local hooks have a dockerfile path
		// Plugins default to Dockerfile
		if dockerFilePath == "" {
			dockerFilePath = "Dockerfile"
		}

		dockerFilePath = path.Join(repoDir, dockerFilePath)
	}

	imageName := fmt.Sprintf("%s-%s-%d-%d", config.ServiceName, hookName, hookIndex, hookCounter)

	if !buildAndRun(log, imageName, dockerFilePath, runArgs, opts) {
		return
	}

	success = true
	return
}

func buildAndRun(log log15.Logger, imageName, dockerFilePath string,
	runArgs []string, opts *Opts) (success bool) {

	log = log.New("Dockerfile", dockerFilePath, "ImageName", imageName)

	log.Debug("buildAndRun() BEGIN")
	defer log.Debug("buildAndRun() END")
	log.Info("docker build")

	dockerBuildCmd := exec.Command("docker", "build",
		"-t", imageName,
		"-f", dockerFilePath,
		path.Dir(dockerFilePath),
	)
	dockerBuildCmd.Stdout = opts.BuildStdout
	dockerBuildCmd.Stderr = opts.BuildStderr

	err := dockerBuildCmd.Run()
	if err != nil {
		log.Error("docker build", "Dockerfile", dockerFilePath, "Error", err)
		return
	}

	runArgs = append(runArgs, imageName)

	runCmd := exec.Command("docker", runArgs...)
	runCmd.Stdout = opts.RunStdout
	runCmd.Stderr = opts.RunStderr

	log.Debug("docker run", "Args", runArgs)

	err = runCmd.Run()
	if err != nil {
		log.Error("docker run", "Error", err)
		return
	}

	success = true
	return
}
