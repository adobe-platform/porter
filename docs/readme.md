porter
======

`porter` is a platform built on AWS APIs to enable continuous delivery (CD) of
docker containers to EC2 instances across AWS regions.

- [Features](#features)
- [Project fit](#project-fit)
- [Documentation](#documentation)
- [Getting started](#getting-started)

Features
--------

- Interactive command line interface (CLI) packaged as a statically linked binary
- Multi-region, multi-AZ, blue-green deployments.
- Works in EC2-Classic, Default VPC, and Custom VPCs
- Highly customizable
  - Programmable deployment pipeline and EC2 host-level customizations
  - Customizable AWS infrastructure with CloudFormation
- Rapidly bootstrap an empty AWS account
- Easily integrates with any CI/CD software
- Easy to adopt and unobtrusive
- Allows developers to provision a single-region stack using the same
  CloudFormation template used in other environments to verify feature and bug
  fixes _without_ needing to integrate in a shared environment (e.g. QA2)
- Configurable deployment blackout windows during which build commands
  will fail
- Secrets management with KMS and S3 integration

Project fit
-----------

Porter is a great fit for a lot of different projects, and not a great fit for
others. Here's some examples of each

### Good fit

- Typical single-tenant N-tier services
- Projects practicing continuous delivery
- Dockerized, [12factor](http://12factor.net/) services
- Teams wanting to use AWS that don't have the resources to build a
  deployment pipeline but find CodePipeline, CodeDeploy, ECS, and
  ElasticBeanstalk too constraining.
- Mid-large companies / operations teams
  - Possibly with a multiple AWS accounts
  - Looking to provide a PaaS (`porter + CI/CD + automated onboarding == PaaS`)
  - Wanting to prevent "CI snowflakes" (service-specific one-off logic baked
    into CI jobs). Porter's CI integration surface area is intentionally very
    small.
- Multi-region services
- Client applications (SPA, web-deployed clients) with a backend service
  - Have OAuth2 flows? `porter create-stack` can help
- Projects looking to vastly simplify host provisioning (burn your cookbooks)

### Poor fit

- Mid-large projects consisting of many microservices, or projects needing the
  cost efficiency of clustering technologies (e.g. DCOS, Kubernetes)
- Sophisticated Docker users. Porter uses basic features of Docker 1.7
- Projects that manually release software. Porter was born and bred to support a
  fully automated CD environment. That said it is possible to add manual
  verification steps to a porter deployment.
- Dockerized services with simple deployment requirements - use ECS
- Dockerized services with no need of host-level software - use ECS
- Entirely stateless services (not even in-memory caching) - use Lambda
- Projects needing cloud-agnostic or alternative infrastructure provisioning - use [Terraform](https://www.terraform.io)
- Projects need cloud-agnosticism. Porter is tied to AWS.

Documentation
-------------

Lots of useful documentation is built into porter itself.

Run `porter help`, and most porter commands with no arguments for details on how
to call them.

- [Config reference](config-reference.md)
- [Deployment hooks](detailed_design/deployment-hooks.md)
- [CloudFormation customizations](detailed_design/cfn-customization.md)
- [Container config (including secrets)](detailed_design/container-config.md)
- [CI/CD integration](detailed_design/ci-cd-integration.md)
- [Deployment flow](https://www.lucidchart.com/documents/view/95a3fdca-ff76-40c5-98fd-6b3071ba86bc)
- [Service interface](detailed_design/platform-service.md)
- [Porter components](detailed_design/components.md)
- [Porter's dependencies](detailed_design/versions.md)
- [FAQ](faq.md)

Getting started
---------------

### Prerequisites

To use this platform effectively the following is required

- Comfortable using a command line interface (CLI)
- Basic understanding of Docker such as what a `Dockerfile` contains, and the
  `docker build` and `docker run` commands
- Basic understanding of HTTP, DNS, and SSH

To customize this platform or the infrastructure created, an excellent
understanding of CloudFormation templates is required.

### Downloading and installing

Download, rename, and chmod the latest release for your platform from
[the releases page](https://github.com/adobe-platform/porter/releases)

porter has two runtime dependencies: git and docker.

A working docker installation looks like this

```bash
[bcook:~]$ docker version
Client:
 Version:      1.10.2
 API version:  1.22
 Go version:   go1.5.3
 Git commit:   c3959b1
 Built:        Mon Feb 22 22:37:33 2016
 OS/Arch:      darwin/amd64

Server:
 Version:      1.10.2
 API version:  1.22
 Go version:   go1.5.3
 Git commit:   c3959b1
 Built:        Mon Feb 22 22:37:33 2016
 OS/Arch:      linux/amd64
```
