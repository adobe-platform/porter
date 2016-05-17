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
	"time"

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

	log = log.New("hook", hookName)
	log.Info("Hook BEGIN")

	config, configSuccess := conf.GetConfig(log)
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

	var configHooks []conf.Hook

	switch hookName {
	case constants.HookPrePack:
		configHooks = config.Hooks.PrePack
	case constants.HookPostPack:
		configHooks = config.Hooks.PostPack
	case constants.HookPreProvision:
		configHooks = config.Hooks.PreProvision
	case constants.HookPostProvision:
		configHooks = config.Hooks.PostProvision
	case constants.HookPrePromote:
		configHooks = config.Hooks.PrePromote
	case constants.HookPostPromote:
		configHooks = config.Hooks.PostPromote
	case constants.HookPrePrune:
		configHooks = config.Hooks.PrePrune
	case constants.HookPostPrune:
		configHooks = config.Hooks.PostPrune
	case constants.HookEC2Bootstrap:
		configHooks = config.Hooks.EC2Bootstrap
	default:
		log.Error("Invalid hook")
		return
	}

	if environment == "" {

		runArgs := runArgsFactory(config, workingDir)

		dockerFilePath := path.Join(constants.HookDir, hookName)
		_, err = os.Stat(dockerFilePath)
		if err == nil {

			imageName := fmt.Sprintf("%s-hook-%s", config.ServiceName, hookName)

			if !buildAndRun(log, imageName, dockerFilePath, runArgs, opts) {
				return
			}
		}

		if !runConfigHooks(log, config, configHooks, runArgs, opts) {
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

			log := log.New("Region", region.Name)

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

			runArgs := runArgsFactory(config, workingDir)

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

			dockerFilePath := path.Join(constants.HookDir, hookName)
			_, err = os.Stat(dockerFilePath)
			if err == nil {

				imageName := fmt.Sprintf("%s-hook-%s", config.ServiceName, hookName)

				if !buildAndRun(log, imageName, dockerFilePath, runArgs, opts) {
					return
				}
			}

			if !runConfigHooks(log, config, configHooks, runArgs, opts) {
				return
			}
		}
	}

	log.Info("Hook END")

	success = true
	return
}

func runArgsFactory(config *conf.Config, workingDir string) []string {
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

	for _, kvp := range os.Environ() {
		if strings.HasPrefix(kvp, "PORTER_") {
			runArgs = append(runArgs, "-e", strings.TrimPrefix(kvp, "PORTER_"))
		}
	}

	return runArgs
}

func runConfigHooks(log log15.Logger, config *conf.Config, hooks []conf.Hook,
	runArgs []string, opts *Opts) (success bool) {

	for _, hook := range hooks {

		if !runConfigHook(log, config, hook, runArgs, opts) {
			return
		}
	}

	success = true
	return
}

func runConfigHook(log log15.Logger, config *conf.Config, hook conf.Hook,
	runArgs []string, opts *Opts) (success bool) {

	repoDir := fmt.Sprintf("%d", time.Now().UnixNano())
	repoDir = path.Join(constants.TempDir, repoDir)
	defer func() {
		exec.Command("rm", "-fr", repoDir).Run()
	}()

	log.Info("Cloning",
		"Repo", hook.Repo,
		"Ref", hook.Ref,
		"Directory", repoDir,
	)

	cloneCmd := exec.Command(
		"git", "clone",
		"--branch", hook.Ref, // branch is a misnomer. this works on any ref
		"--depth", "1",
		hook.Repo, repoDir,
	)
	cloneCmd.Stderr = os.Stderr
	err := cloneCmd.Run()
	if err != nil {
		log.Error("git clone", "Error", err)
		return
	}

	dockerFilePath := hook.Dockerfile
	if dockerFilePath == "" {
		dockerFilePath = "Dockerfile"
	}

	hookName := path.Base(strings.TrimSuffix(hook.Repo, ".git"))
	dockerFilePath = path.Join(repoDir, dockerFilePath)
	imageName := fmt.Sprintf("%s-%s", config.ServiceName, hookName)

	if !buildAndRun(log, imageName, dockerFilePath, runArgs, opts) {
		return
	}

	success = true
	return
}

func buildAndRun(log log15.Logger, imageName, dockerFilePath string,
	runArgs []string, opts *Opts) (success bool) {

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

	err = runCmd.Run()
	if err != nil {
		log.Error("docker run", "Error", err)
		return
	}

	success = true
	return
}
