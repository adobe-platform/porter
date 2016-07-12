Container Security
==================

Here's how porter measures up to the following recommendations from the
[CIS Docker 1.11.0 Benchmark](https://benchmarks.cisecurity.org/tools2/docker/CIS_Docker_1.11.0_Benchmark_v1.0.0.pdf)

The following only applies to users that _do not_ provide a custom AMI which may
alter host-level recommendations. Any recommendation in the CIS Docker 1.11.0
Benchmark that doesn't appear here isn't applicable to porter.

This audit was performed on the current version of Amazon Linux AMI that porter
uses by default. See [Versions](versions.md) for details

### :white_check_mark: 2.1 Restrict network traffic between containers

N/A. Porter runs a single container

(for now: https://github.com/adobe-platform/porter/issues/8)

### :white_check_mark: 2.3 Allow Docker to make changes to iptables

The daemon is not run with `--iptables=false`

### :white_check_mark: 2.4 Do not use insecure registries

The daemon is not run with `--insecure-registry`

### :white_check_mark: 2.5 Do not use the aufs storage driver

The daemon is not run with an explicit storage driver

### :white_check_mark: 2.6 Configure TLS authentication for Docker daemon

The daemon is not exposed

### :white_check_mark: 2.7 Set default ulimit as appropriate

N/A. Porter sets ulimit explicity during `docker run`

### :white_check_mark: 2.8 Enable user namespace support

Superseded by 4.1

### :white_check_mark: 2.9 Confirm default cgroup usage

The daemon is not run with `--cgroup-parent`

### :white_check_mark: 2.10 Do not change base device size until needed

The daemon is not run with `--storage-opt`

### :x: 2.11 Use authorization plugin

This must be implemented by porter users

### :white_check_mark: 2.12 Configure centralized and remote logging

Porter runs with the syslog log driver and aggregates logs in `/var/log/porter.log`

### :x: 2.13 Disable operations on legacy registry (v1)

Tracked here https://github.com/adobe-platform/porter/issues/42

### :white_check_mark: 3.1 Verify that docker.service file ownership is set to root:root

The current version Amazon Linux AMI in use doesn't use systemd.

`/etc/init.d/docker` is set to root:root

### :x: 3.2 Verify that docker.service file permissions are set to 644 or more restrictive

The current version Amazon Linux AMI in use doesn't use systemd.

Tracked here https://github.com/adobe-platform/porter/issues/43

### :question: 3.3 Verify that docker.socket file ownership is set to root:root

Conflicts with 3.15

Tracked here https://github.com/adobe-platform/porter/issues/44

### :x: 3.4 Verify that docker.socket file permissions are set to 644 or more restrictive

Conflicts with 3.16

Tracked here https://github.com/adobe-platform/porter/issues/45

### :white_check_mark: 3.5 Verify that /etc/docker directory ownership is set to root:root

### :white_check_mark: 3.6 Verify that /etc/docker directory permissions are set to 755 or more restrictive

### :white_check_mark: 3.7 Verify that registry certificate file ownership is set to root:root

N/A. There is no certs.d directory

### :white_check_mark: 3.8 Verify that registry certificate file permissions are set to 444 or more restrictive

N/A. There is no certs.d directory

### :white_check_mark: 3.9 Verify that TLS CA certificate file ownership is set to root:root

N/A. The daemon is not run with `--tlscacert`

### :white_check_mark: 3.10 Verify that TLS CA certificate file permissions are set to 444 or more restrictive

N/A. The daemon is not run with `--tlscacert`

### :white_check_mark: 3.11 Verify that Docker server certificate file ownership is set to root:root

N/A. The daemon is not run with `--tlscacert`

### :white_check_mark: 3.12 Verify that Docker server certificate file permissions are set to 444 or more restrictive

N/A. The daemon is not run with `--tlscacert`

### :white_check_mark: 3.13 Verify that Docker server certificate key file ownership is set to root:root

N/A. The daemon is not run with `--tlskey`

### :white_check_mark: 3.14 Verify that Docker server certificate key file permissions are set to 400

N/A. The daemon is not run with `--tlskey`

### :white_check_mark: 3.15 Verify that Docker socket file ownership is set to root:docker

### :x: 3.16 Verify that Docker socket file permissions are set to 660 or more restrictive

Tracked here https://github.com/adobe-platform/porter/issues/45

### :white_check_mark: 3.17 Verify that daemon.json file ownership is set to root:root

N/A. `/etc/docker/daemon.json` doesn't exist.

### :white_check_mark: 3.18 Verify that daemon.json file permissions are set to 644 or more restrictive

N/A. `/etc/docker/daemon.json` doesn't exist.

### :white_check_mark: 3.19 Verify that /etc/default/docker file ownership is set to root:root

N/A. `/etc/default/docker` doesn't exist.

### :white_check_mark: 3.20 Verify that /etc/default/docker file permissions are set to 644 or more restrictive

N/A. `/etc/default/docker` doesn't exist.

### :white_check_mark: 4.1 Create a user for the container

A `porter-docker` user with uid 601 is provisioned with the CloudFormation stack.

For containers that need to run as root see the [uid](config-reference.md#uid) config
