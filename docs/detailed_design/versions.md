Versioning
==========

It's important that the software services deploy on remains stable over time and
that building on new software is intentional. `porter` ensures version locking
of all software except security updates on Amazon Linux AMI using
[`repo_upgrade: security`](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AmazonLinuxAMIBasics.html#security-updates)
(the default) and [`repo_releasever: 2015.09`](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AmazonLinuxAMIBasics.html#RepoConfig)

Checkout `/var/log/cloud-init-output.log` to see what security patches have been
applied to the host.

Versions
--------

Here is the version list of software actively used by porter.
Packages such as cfn-hup, yum, and cloud-init are excluded.

Refer to the [Amazon Linux AMI release notes](https://aws.amazon.com/amazon-linux-ami/2015.09-release-notes/)
for additional package information

| Software               | Version |
|------------------------|---------|
| Amazon Linux AMI       | 2015.09 |
| rsyslogd               | 5.8.10  |
| curl                   | 7.40.0  |
| wget                   | 1.16.1  |
| HA-Proxy               | 1.5.2   |
| porter (Go)            | 1.6.2   |
| porterd (Go)           | 1.6.2   |
