ec2-bootstrap
=============

The ec2-bootstrap hook allows EC2 host-level customizations.

**Key point to understand**

> This hook allows services to customize _at build time_ what EC2 will run
> during initialization by _generating code_. The stdout of this hook is placed
> into the bootstrap scripts called by porter during EC2 initialization. See the
> [`porter_bootstrap`](../../../files/porter_bootstrap) for the exact location
> of code injection.

In general there are two ways to customize your EC2 host: (1) provide a custom
AMI and (2) use this hook.

Caveats
-------

1. There is a limit on the size of a CloudFormation template. For extensive
customizations or long scripts it's best to have them already available in a
custom AMI, or download and run them during EC2 initialization.
1. EC2 initialization is only done once, and not repeated even if an instance is
restarted.

Support
-------

Please do not open issues related to this hook.

Once you begin customizing the EC2 instance with your own scripts you have the
ability to completely break porter's battle-tested initialization.

Use Cases
---------

- Any EC2 customizations

Lifecycle
---------

This hook is called **for each** region when creating a CloudFormation template.

The image built is used once, its stdout captured and placed in [`porter_bootstrap`](../../../files/porter_bootstrap)
and it is thrown away.

The container doesn't not run on EC2.

The image itself is not part of the service payload.

This hook isn't tied to a `porter build` command and runs even during
`create-stack`.

Environment
-----------

[Standard](../deployment-hooks.md#standard-environment-variables)
and [Custom](../deployment-hooks.md#custom-environment-variables)
environment variables

```
PORTER_ENVIRONMENT
AWS_REGION
```

`AWS_DEFAULT_REGION` `AWS_ACCESS_KEY_ID` `AWS_SECRET_ACCESS_KEY`
`AWS_SESSION_TOKEN` `AWS_SECURITY_TOKEN` are available to enable AWS SDKs to
make API calls and are the credentials of the assumed role.

Examples
--------

This is the most complex and interesting hook available so it's worth providing
examples of how to use it effectively.

- [Install Datadog (BASH)](#install-datadog-bash)
- [Install Datadog (Python)](#install-datadog-python)
- [Externalize container config](#externalize-container-config)

### Install Datadog (BASH)

`.porter/hooks/ec2-bootstrap`

```Dockerfile
FROM ubuntu:14.04

ADD .porter/hooks/ec2-bootstrap.cmd /
RUN chmod 544 /ec2-bootstrap.cmd

CMD /ec2-bootstrap.cmd
```

`.porter/hooks/ec2-bootstrap.cmd`

This file is used to generate the code that will be run on EC2 during
initialization.

```bash
#!/bin/bash
# export the DD_API_KEY environment variable that's available in the hook
echo "export DD_API_KEY=$DD_API_KEY"

# create the installation script
# notice that the single quotes around DATADOG prevent parameter expansion
# http://tldp.org/LDP/abs/html/here-docs.html
cat <<'DATADOG'
echo "installing datadog"
bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/dd-agent/master/packaging/datadog-agent/source/install_agent.sh)"
DATADOG
```

With these files call porter like this (replace your_api_key with your api key
if you want the agent to actually run, of course):

```bash
PORTER_DD_API_KEY=your_api_key \
porter create-stack -e dev
```

The environment variable `PORTER_DD_API_KEY` becomes `DD_API_KEY` inside the
hook. The final **output** of this hook is:

```bash
export DD_API_KEY=your_api_key
echo "installing datadog"
bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/dd-agent/master/packaging/datadog-agent/source/install_agent.sh)"
```

And it is this **output** of the hook that will be injected into the
`/usr/bin/porter_bootstrap` script to be run when an EC2 instance initializes.

### Install Datadog (Python)

Shell expansion can be confusing. Here's the same example in Python.

`.porter/hooks/ec2-bootstrap`

```Dockerfile
FROM python:2.7.11

ADD ec2-bootstrap.py /

CMD ["python", "/ec2-bootstrap.py"]
```

`.porter/hooks/ec2-bootstrap.py`

```python
from __future__ import print_function
import sys
import os


def stderr(*objs):
    print(*objs, file=sys.stderr)

# export the DD_API_KEY environment variable that's available in the hook
print("export DD_API_KEY={}".format(os.environ['DD_API_KEY']))

# let our build system users know what's happening by communicating on stderr
stderr("EC2 host will install datadog")

# create the installation script
print('echo "installing datadog"')
print('bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/dd-agent/master/packaging/datadog-agent/source/install_agent.sh)"')
```

With these files call porter like this (replace your_api_key with your api key
if you want the agent to actually run, of course):

```bash
PORTER_DD_API_KEY=your_api_key \
porter create-stack -e dev
```

The environment variable `PORTER_DD_API_KEY` becomes `DD_API_KEY` inside the
hook. The final **output** of this hook is:

```bash
export DD_API_KEY=your_api_key
echo "installing datadog"
bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/dd-agent/master/packaging/datadog-agent/source/install_agent.sh)"
```

### Externalize container config

Porter is heavily influenced by 12factor which talks about [immutable artifacts](http://12factor.net/codebase)
and [how to configure](http://12factor.net/config) them.

The product of `porter build pack` is a service payload which can be used in
multiple environments. A question that comes up is how to follow 12factor using
porter. Docker as early as 1.7 has been able to read a `--env-file`. Porter
creates this env file and makes it available for services to customize _during
EC2 initialization_.

One of the standard environment variables is `DOCKER_ENV_FILE` which we'll use
here.

`.porter/hooks/ec2-bootstrap`

```Dockerfile
FROM golang:1.6

ADD ec2-bootstrap.go /

CMD ["go", "run", "/ec2-bootstrap.go"]
```

`.porter/hooks/ec2-bootstrap.go`

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	region := os.Getenv("AWS_REGION")
	environment := os.Getenv("PORTER_ENVIRONMENT")
	envFile := os.Getenv("DOCKER_ENV_FILE")

	if region == "" || environment == "" || envFile == "" {
		// we need all of these
		os.Exit(2)
	}

	var configValue string
	switch environment {
	case "stage":
		switch region {
		case "us-west-2":
			configValue = "stage-us-west-2-value"
		case "us-east-1":
			configValue = "stage-us-east-1-value"
		default:
			os.Exit(2)
		}
	case "prod":
		switch region {
		case "us-west-2":
			configValue = "prod-us-east-1-value"
		case "us-east-1":
			configValue = "prod-us-east-1-value"
		default:
			os.Exit(2)
		}
	default:
		os.Exit(2)
	}

	fmt.Fprintln(os.Stderr, "Hi build box user")
	fmt.Fprintln(os.Stderr, "The container will run with SOME_CONFIG="+configValue)

	// if environment == "stage" && region == "us-west-2"
	// append SOME_CONFIG=stage-us-west-2-value to the --env-file
	// so it's available to the container at runtime
	codeToRunOnEC2 := fmt.Sprintf("echo 'SOME_CONFIG=%s' >> %s", configValue, envFile)

	// prints to stdout
	fmt.Println(codeToRunOnEC2)
}
```
