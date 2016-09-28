Security
========

Porter is secure by default where possible. The defaults can hinder normal
development or rapid prototyping. This is a non-exhaustive list of security
touch points, most of which are configurable.

- Infrastructure
  - [SSH key](config-reference.md#key_pair_name)
  - [EC2 SSH ingress](cfn-customization.md#ssh)
  - [EC2 IAM role permissions](cfn-customization.md#additional-ec2-permissions)
  - [EC2 egress traffic](config-reference.md#security_group_egress)
  - [SSL certs](config-reference.md#ssl_cert_arn)
  - [SSL with R53](config-reference.md#ssl_cert_arn)
- Container
  - Runtime
    - [Container runtime config (including secrets)](container-config.md)
    - [UID](config-reference.md#uid)
    - [Read-only FS](config-reference.md#read_only)
    - [The code that calls `docker run`](../../commands/host/docker.go)
    - [Daemon config](../../files/porter_bootstrap)
  - Build time
    - [Service payload](service-payload.md)

Container Security: CIS Docker Benchmark
----------------------------------------

Here's how porter measures up to the following recommendations from the
[CIS Docker 1.11.0 Benchmark](https://benchmarks.cisecurity.org/tools2/docker/CIS_Docker_1.11.0_Benchmark_v1.0.0.pdf)

The following only applies to users that _do not_ provide a custom AMI which may
alter host-level recommendations. Any recommendation in the CIS Docker 1.11.0
Benchmark that doesn't appear here isn't applicable to porter.

This audit was performed on the current version of Amazon Linux AMI that porter
uses by default. See [Versions](versions.md) for details

```
# ------------------------------------------------------------------------------
# Docker Bench for Security v1.1.0
#
# Docker, Inc. (c) 2015-
#
# Checks for dozens of common best-practices around deploying Docker containers in production.
# Inspired by the CIS Docker 1.11 Benchmark:
# https://benchmarks.cisecurity.org/downloads/show-single/index.cfm?file=docker16.110
# ------------------------------------------------------------------------------

Initializing Wed Aug 10 23:07:18 UTC 2016


[INFO] 1 - Host Configuration
[WARN] 1.1  - Create a separate partition for containers
[PASS] 1.2  - Use an updated Linux Kernel
[WARN] 1.4  - Remove all non-essential services from the host - Network
[WARN]      * Host listening on: 16 ports
[WARN] 1.5  - Keep Docker up to date
[WARN]       * Using 1.11.2, when 1.12.0 is current as of 2016-07-28
[INFO]       * Your operating system vendor may provide support and security maintenance for docker
[INFO] 1.6  - Only allow trusted users to control Docker daemon
[INFO]      * docker:x:497
[WARN] 1.7  - Failed to inspect: auditctl command not found.
[WARN] 1.8  - Failed to inspect: auditctl command not found.
[WARN] 1.9  - Failed to inspect: auditctl command not found.
[INFO] 1.10 - Audit Docker files and directories - docker.service
[INFO]      * File not found
[INFO] 1.11 - Audit Docker files and directories - docker.socket
[INFO]      * File not found
[INFO] 1.12 - Audit Docker files and directories - /etc/default/docker
[INFO]      * File not found
[INFO] 1.13 - Audit Docker files and directories - /etc/docker/daemon.json
[INFO]      * File not found
[INFO] 1.14 - Audit Docker files and directories - /usr/bin/docker-containerd
[INFO]      * File not found
[INFO] 1.15 - Audit Docker files and directories - /usr/bin/docker-runc
[INFO]      * File not found


[INFO] 2 - Docker Daemon Configuration
[PASS] 2.1  - Restrict network traffic between containers
[PASS] 2.2  - Set the logging level
[PASS] 2.3  - Allow Docker to make changes to iptables
[PASS] 2.4  - Do not use insecure registries
[PASS] 2.5  - Do not use the aufs storage driver
[INFO] 2.6  - Configure TLS authentication for Docker daemon
[INFO]      * Docker daemon not listening on TCP
[PASS] 2.7 - Set default ulimit as appropriate
[WARN] 2.8  - Enable user namespace support
[PASS] 2.9  - Confirm default cgroup usage
[PASS] 2.10 - Do not change base device size until needed
[WARN] 2.11 - Use authorization plugin
[WARN] 2.12 - Configure centralized and remote logging
[PASS] 2.13 - Disable operations on legacy registry (v1)


[INFO] 3 - Docker Daemon Configuration Files
[INFO] 3.1  - Verify that docker.service file ownership is set to root:root
[INFO]      * File not found
[INFO] 3.2  - Verify that docker.service file permissions are set to 644
[INFO]      * File not found
[INFO] 3.3  - Verify that docker.socket file ownership is set to root:root
[INFO]      * File not found
[INFO] 3.4  - Verify that docker.socket file permissions are set to 644
[INFO]      * File not found
[PASS] 3.5  - Verify that /etc/docker directory ownership is set to root:root
[PASS] 3.6  - Verify that /etc/docker directory permissions are set to 755
[INFO] 3.7  - Verify that registry certificate file ownership is set to root:root
[INFO]      * Directory not found
[INFO] 3.8  - Verify that registry certificate file permissions are set to 444
[INFO]      * Directory not found
[INFO] 3.9  - Verify that TLS CA certificate file ownership is set to root:root
[INFO]      * No TLS CA certificate found
[INFO] 3.10 - Verify that TLS CA certificate file permissions are set to 444
[INFO]      * No TLS CA certificate found
[INFO] 3.11 - Verify that Docker server certificate file ownership is set to root:root
[INFO]      * No TLS Server certificate found
[INFO] 3.12 - Verify that Docker server certificate file permissions are set to 444
[INFO]      * No TLS Server certificate found
[INFO] 3.13 - Verify that Docker server key file ownership is set to root:root
[INFO]      * No TLS Key found
[INFO] 3.14 - Verify that Docker server key file permissions are set to 400
[INFO]      * No TLS Key found
[PASS] 3.15 - Verify that Docker socket file ownership is set to root:docker
[PASS] 3.16 - Verify that Docker socket file permissions are set to 660
[INFO] 3.17 - Verify that daemon.json file ownership is set to root:root
[INFO]      * File not found
[INFO] 3.18 - Verify that daemon.json file permissions are set to 644
[INFO]      * File not found
[INFO] 3.19 - Verify that /etc/default/docker file ownership is set to root:root
[INFO]      * File not found
[INFO] 3.20 - Verify that /etc/default/docker file permissions are set to 644
[INFO]      * File not found


[INFO] 4 - Container Images and Build Files
[PASS] 4.1  - Create a user for the container
[WARN] 4.5  - Enable Content trust for Docker


[INFO] 5  - Container Runtime
[WARN] 5.1  - Verify AppArmor Profile, if applicable
[WARN]      * No AppArmorProfile Found: fervent_thompson
[WARN]      * No AppArmorProfile Found: sick_kowalevski
[WARN] 5.2  - Verify SELinux security options, if applicable
[WARN]      * No SecurityOptions Found: fervent_thompson
[WARN]      * No SecurityOptions Found: sick_kowalevski
[PASS] 5.3  - Restrict Linux Kernel Capabilities within containers
[PASS] 5.4  - Do not use privileged containers
[PASS] 5.5  - Do not mount sensitive host system directories on containers
[PASS] 5.6  - Do not run ssh within containers
[PASS] 5.7  - Do not map privileged ports within containers
[PASS] 5.9 - Do not share the host's network namespace
[WARN] 5.10 - Limit memory usage for container
[WARN]      * Container running without memory restrictions: fervent_thompson
[WARN]      * Container running without memory restrictions: sick_kowalevski
[WARN] 5.11 - Set container CPU priority appropriately
[WARN]      * Container running without CPU restrictions: fervent_thompson
[WARN]      * Container running without CPU restrictions: sick_kowalevski
[PASS] 5.12 - Mount container's root filesystem as read only
[WARN] 5.13 - Bind incoming container traffic to a specific host interface
[WARN]      * Port being bound to wildcard IP: 0.0.0.0 in fervent_thompson
[WARN]      * Port being bound to wildcard IP: 0.0.0.0 in fervent_thompson
[WARN]      * Port being bound to wildcard IP: 0.0.0.0 in sick_kowalevski
[WARN]      * Port being bound to wildcard IP: 0.0.0.0 in sick_kowalevski
[PASS] 5.14 - Set the 'on-failure' container restart policy to 5
[PASS] 5.15 - Do not share the host's process namespace
[PASS] 5.16 - Do not share the host's IPC namespace
[PASS] 5.17 - Do not directly expose host devices to containers
[PASS] 5.18 - Override default ulimit at runtime only if needed
[PASS] 5.19 - Do not set mount propagation mode to shared
[PASS] 5.20 - Do not share the host's UTS namespace
[PASS] 5.21 - Do not disable default seccomp profile
[PASS] 5.24 - Confirm cgroup usage
[WARN] 5.25 - Restrict container from acquiring additional privileges
[WARN]      * Privileges not restricted: fervent_thompson
[WARN]      * Privileges not restricted: sick_kowalevski


[INFO] 6  - Docker Security Operations
[INFO] 6.4 - Avoid image sprawl
[INFO]      * There are currently: 2 images
[INFO] 6.5 - Avoid container sprawl
[INFO]      * There are currently a total of 3 containers, with 3 of them currently running
```
