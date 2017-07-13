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
package hook

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/adobe-platform/porter/aws/elb"
	"github.com/adobe-platform/porter/aws_session"
	"github.com/adobe-platform/porter/conf"
	"github.com/adobe-platform/porter/constants"
	"github.com/adobe-platform/porter/logger"
	"github.com/adobe-platform/porter/provision_state"
	"gopkg.in/inconshreveable/log15.v2"
)

type (
	regionHookRunner struct {
		runOutput *chan bytes.Buffer

		serviceName string
		hookName    string

		commandSuccess bool
	}

	hookLinkedList struct {
		hook      conf.Hook
		hookIndex int

		next *hookLinkedList
	}
)

var (
	// Multi-region deployment means we need a globally unique id for git clones
	// and image names
	globalCounter *uint32 = new(uint32)
)

func Execute(log log15.Logger,
	hookName, environment string,
	provisionedRegions map[string]*provision_state.Region,
	commandSuccess bool) bool {

	return ExecuteWithRunCapture(log, hookName, environment, provisionedRegions,
		commandSuccess, nil)
}

func ExecuteWithRunCapture(log log15.Logger,
	hookName, environment string,
	provisionedRegions map[string]*provision_state.Region,
	commandSuccess bool, runOutput *chan bytes.Buffer) (success bool) {

	log = log.New("HookName", hookName)
	log.Info("Hook BEGIN")
	defer log.Info("Hook END")

	var (
		config        *conf.Config
		configSuccess bool
	)

	// the only hook that can modify the config and have porter still use it is
	// pre-pack. after porter reads it in pack, it's copied into the tmp
	// directory.
	if hookName == constants.HookPrePack {
		config, configSuccess = conf.GetConfig(log, false)
	} else {
		config, configSuccess = conf.GetAlteredConfig(log)
	}
	if !configSuccess {
		return
	}

	workingDir, err := os.Getwd()
	if err != nil {
		log.Error("Getwd", "Error", err)
		return
	}
	log.Debug("os.Getwd()", "Path", workingDir)

	var configHooks []conf.Hook

	configHooks, exists := config.Hooks[hookName]
	if !exists {
		success = true
		return
	}

	if environment == "" {

		runArgs := runArgsFactory(log, config, workingDir)

		hookRunner := &regionHookRunner{

			runOutput: runOutput,

			serviceName: config.ServiceName,
			hookName:    hookName,

			commandSuccess: commandSuccess,
		}

		success = hookRunner.runConfigHooks(log, os.Stdout, configHooks, runArgs)

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
			provisionedRegions = make(map[string]*provision_state.Region)

			for _, region := range env.Regions {
				provisionedRegions[region.Name] = &provision_state.Region{}
			}
		}

		successChan := make(chan bool)
		var regionLogMutex sync.Mutex

		for regionName, regionState := range provisionedRegions {

			elbDNS := ""

			roleARN, err := env.GetRoleARN(regionName)
			if err != nil {
				log.Error("GetRoleARN", "Error", err)
				return
			}

			roleSession := aws_session.STS(regionName, roleARN, 3600)

			if regionState.ProvisionedELBName != "" {
				elbClient := elb.New(roleSession)

				log.Info("DescribeLoadBalancers")
				output, err := elb.DescribeLoadBalancers(elbClient, regionState.ProvisionedELBName)
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
				"-e", "AWS_REGION="+regionName,
				// AWS_DEFAULT_REGION is also needed for AWS SDKs
				"-e", "AWS_DEFAULT_REGION="+regionName,
				"-e", "AWS_ACCESS_KEY_ID="+credValue.AccessKeyID,
				"-e", "AWS_SECRET_ACCESS_KEY="+credValue.SecretAccessKey,
				"-e", "AWS_SESSION_TOKEN="+credValue.SessionToken,
				"-e", "AWS_SECURITY_TOKEN="+credValue.SessionToken,
			)

			if elbDNS != "" {
				runArgs = append(runArgs,
					"-e", "AWS_ELASTICLOADBALANCING_LOADBALANCER_DNS="+elbDNS)
			}

			if regionState.StackId != "" {
				runArgs = append(runArgs,
					"-e", "AWS_CLOUDFORMATION_STACKID="+regionState.StackId)
			}

			hookRunner := &regionHookRunner{
				runOutput: runOutput,

				serviceName: config.ServiceName,
				hookName:    hookName,

				commandSuccess: commandSuccess,
			}

			go func(runner *regionHookRunner, log log15.Logger,
				hooks []conf.Hook, runArgs []string) {

				log = log.New()

				var regionLogOutput bytes.Buffer
				logger.SetHandler(log, &regionLogOutput)

				hooksResult := runner.runConfigHooks(log, &regionLogOutput, hooks, runArgs)

				regionLogMutex.Lock()
				regionLogOutput.WriteTo(os.Stdout)
				regionLogMutex.Unlock()

				successChan <- hooksResult
			}(hookRunner, log, configHooks, runArgs)
		}

		success = true

		for i := 0; i < len(provisionedRegions); i++ {

			runConfigHooksSuccess := <-successChan

			success = success && runConfigHooksSuccess
		}
	}

	return
}

func runArgsFactory(log log15.Logger, config *conf.Config, workingDir string) []string {
	var mountedVolume, volumeFlag string
	mountedVolume = "/repo_root"
	volumeFlag = os.Getenv(constants.EnvVolumeFlag)

	// Adding volume flag to either allow read only or to maintain same volume flags
	switch volumeFlag {
	case "z", "ro":
	default:
		log.Info("Setting the default volume flag as empty.")
		volumeFlag = ""
	}

	if volumeFlag != "" {
		mountedVolume = fmt.Sprintf("%s:%s", mountedVolume, workingDir)
	}

	runArgs := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:%s", workingDir, mountedVolume),
		"-e", "PORTER_SERVICE_NAME=" + config.ServiceName,
		"-e", "DOCKER_ENV_FILE=" + constants.EnvFile,
		"-e", "HAPROXY_STATS_USERNAME=" + config.HAProxyStatsUsername,
		"-e", "HAPROXY_STATS_PASSWORD=" + config.HAProxyStatsPassword,
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

func (recv *regionHookRunner) runConfigHooks(log log15.Logger,
	regionLogOutput io.Writer, hooks []conf.Hook,
	runArgs []string) (success bool) {

	successChan := make(chan bool)
	var (
		concurrentCount   int
		eligibleHookCount int

		head *hookLinkedList
		tail *hookLinkedList

		hookLogMutex sync.Mutex
	)

	// Only retained lexical scope is successChan, everything else is copied
	goGadgetHook := func(log log15.Logger, hookIndex int, hook conf.Hook, runArgs []string) {

		var hookLogOutput bytes.Buffer
		logger.SetHandler(log, &hookLogOutput)

		hookResult := recv.runConfigHook(log, &hookLogOutput, hookIndex, hook, runArgs)

		hookLogMutex.Lock()
		hookLogOutput.WriteTo(regionLogOutput)
		hookLogMutex.Unlock()

		successChan <- hookResult
	}

	for hookIndex, hook := range hooks {
		if recv.commandSuccess {
			if hook.RunCondition == constants.HRC_Fail {
				continue
			}
		} else {
			if hook.RunCondition == constants.HRC_Pass {
				continue
			}
		}
		eligibleHookCount++

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

	if recv.runOutput != nil {
		*recv.runOutput = make(chan bytes.Buffer, eligibleHookCount)
	}

	for node := head; node != nil; node = node.next {

		if node.hook.Concurrent {

			concurrentCount++
		} else {

			concurrentCount = 1
		}

		log := log.New(
			"HookIndex", node.hookIndex,
			"Concurrent", node.hook.Concurrent,
			"Repo", node.hook.Repo,
			"Ref", node.hook.Ref,
			"RunCondition", node.hook.RunCondition,
		)

		log.Debug("go go gadget hook")
		go goGadgetHook(log, node.hookIndex, node.hook, runArgs)

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

			concurrentCount = 0
		}
	}

	success = true
	return
}

func (recv *regionHookRunner) runConfigHook(log log15.Logger,
	hookLogOutput io.Writer, hookIndex int, hook conf.Hook,
	runArgs []string) (success bool) {

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

		repoDir := fmt.Sprintf("%s_clone_%d_%d", recv.hookName, hookIndex, hookCounter)
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
		cloneCmd.Stdout = hookLogOutput
		cloneCmd.Stderr = hookLogOutput
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

	imageName := fmt.Sprintf("%s-%s-%d-%d",
		recv.serviceName, recv.hookName, hookIndex, hookCounter)

	if !recv.buildAndRun(log, hookLogOutput, imageName, dockerFilePath, runArgs) {
		return
	}

	success = true
	return
}

func (recv *regionHookRunner) buildAndRun(log log15.Logger,
	hookLogOutput io.Writer, imageName, dockerFilePath string,
	runArgs []string) (success bool) {

	log = log.New("Dockerfile", dockerFilePath, "ImageName", imageName)

	log.Debug("buildAndRun() BEGIN")
	defer log.Debug("buildAndRun() END")

	var runOutput bytes.Buffer

	defer func() {

		if recv.runOutput != nil {
			*recv.runOutput <- runOutput
		}
	}()

	log.Info("You are now exiting porter and entering a porter deployment hook")
	log.Info("Deployment hooks are used to hook into porter's build lifecycle")
	log.Info("None of the code about to be run is porter code")
	log.Info("If you experience problems talk to the author of this Dockerfile")
	log.Info("You can read more about deployment hooks here http://bit.ly/2dKBwd0")

	dockerBuildCmd := exec.Command("docker", "build",
		"-t", imageName,
		"-f", dockerFilePath,
		path.Dir(dockerFilePath),
	)
	dockerBuildCmd.Stdout = hookLogOutput
	dockerBuildCmd.Stderr = hookLogOutput

	fmt.Fprintln(hookLogOutput, "Building deployment hook START")
	fmt.Fprintln(hookLogOutput, "==============================")
	err := dockerBuildCmd.Run()
	fmt.Fprintln(hookLogOutput, "============================")
	fmt.Fprintln(hookLogOutput, "Building deployment hook END")

	if err != nil {
		log.Error("docker build", "Error", err)
		fmt.Fprintln(hookLogOutput, "This is not a problem with porter but with the Dockerfile porter tried to build")
		fmt.Fprintln(hookLogOutput, "DO NOT contact Brandon Cook to help debug this issue")
		fmt.Fprintln(hookLogOutput, "DO NOT file an issue against porter")
		fmt.Fprintln(hookLogOutput, "DO contact the author of the Dockerfile or try to reproduce the problem by running docker build on this machine")
		return
	}

	runArgs = append(runArgs, imageName)

	log.Debug("docker run", "Args", runArgs)

	runCmd := exec.Command("docker", runArgs...)
	runCmd.Stdout = io.MultiWriter(hookLogOutput, &runOutput)
	runCmd.Stderr = hookLogOutput

	fmt.Fprintln(hookLogOutput, "Running deployment hook START")
	fmt.Fprintln(hookLogOutput, "=============================")
	err = runCmd.Run()
	fmt.Fprintln(hookLogOutput, "===========================")
	fmt.Fprintln(hookLogOutput, "Running deployment hook END")

	if err != nil {
		log.Error("docker run", "Error", err)
		fmt.Fprintln(hookLogOutput, "This is not a problem with porter but with the Dockerfile porter tried to run")
		fmt.Fprintln(hookLogOutput, "DO NOT contact Brandon Cook to help debug this issue")
		fmt.Fprintln(hookLogOutput, "DO NOT file an issue against porter")
		fmt.Fprintln(hookLogOutput, "DO contact the author of the Dockerfile")
		fmt.Fprintln(hookLogOutput, "Run `porter help debug` to see how to enable debug logging which will show you the arguments used in docker run")
		fmt.Fprintln(hookLogOutput, "Be aware that enabling debug logging will print sensitive data including, but not limited to, AWS credentials")
		return
	}

	success = true
	return
}
