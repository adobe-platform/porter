Container config
================

Porter follows [12factor](http://12factor.net/config) when it comes to
configuring services. Often config contains long-term secrets like API keys that
shouldn't be baked into a container.

Secure secrets storage is outside of the scope of porter but secure secret
transport is something porter helps with by providing a mechanism to securely
inject environment variable at runtime.

Although porter doesn't deal with secrets storage directly, it is possible to
use S3 (optionally with [SSE-KMS](http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingKMSEncryption.html))
for long-term storage of secrets. SSE-KMS is transparent from a client
perspective (i.e. encrypted secrets are received as plain text over TLS and the
client doesn't know if they were previously encrypted). Keep in mind all
security aspects of S3 with SSE-KMS are outside of porter's control and have to
do with policies and permissions within an AWS account. You must evaluate S3's
suitability for long-term secrets storage.

The alternative to S3 with SSE-KMS is to handle your own secrets and pass them
to porter build commands at runtime.

Source 1: S3 w/ SS3-KMS
-----------------------

### Create a secrets file

Assuming the existence of

1. A S3 bucket to place secrets in called `secrets-src-bucket`
1. A S3 bucket to copy secrets into called `secrets-dst-bucket`. The source and destination can be the same bucket.
1. A KMS key with the ARN `arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee`

We need a secrets file that is the format Docker prescribes for [`--env-file`](https://docs.docker.com/engine/reference/commandline/run/#set-environment-variables-e-env-env-file)

Here is a sample env file

```
# comment on SUPER_SECRET_SECRET
SUPER_SECRET_SECRET=dont_tell_anyone
FOO=bar
```

We'll call this file `secrets.env-file`, upload it to the source bucket, and
encrypt it with a KMS key. Again, since SSE-KMS is transparent from the client
side, porter doesn't care if `--server-side-encryption` and `--ssekms-key-id`
were used in this operation. All that matters is that the role it assumes during
provisioning has the necessary permissions to decrypt if those options are used.

```
aws s3api put-object \
--bucket secrets-src-bucket \
--key secrets.env-file \
--body secrete.env-file \
--server-side-encryption aws:kms \
--ssekms-key-id arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
```

### Configure the container

In `.porter/config` set the `src_env_file` and `dst_env_file` properties

```yaml
environments:
- name: prod
  regions:
  - name: us-west-2
    containers:
    - topology: inet
      src_env_file:
        s3_bucket: secrets-src-bucket
        s3_key: secrets.env-file
      dst_env_file:
        s3_bucket: secrets-dst-bucket
        kms_arn: arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
```

Note: `dst_env_file.kms_arn` is optional but there for convenience so SSE-KMS
can be used throughout the entire flow.

### Aside: multi-region container config

SSE-KMS ties a S3 bucket to a KMS key per region so the following config is
common for multi-region deployments where the source for secrets is central and
they're copied to S3 buckets in the regions to which a service will be deployed.

Here is an example config that deploys to 2 regions (us-west-2 and us-east-1),
keeps secrets in the us-east-1 bucket, and encrypts the destination with SSE-KMS.

```yaml
_container_base_definition: &CONTAINER_BASE
  topology: inet
  src_env_file:
    s3_bucket: secrets-src-bucket-us-east-1
    s3_key: secrets.env-file

environments:
- name: prod

  regions:
  - name: us-east-1
    containers:
    - <<: *CONTAINER_BASE
      dst_env_file:
        s3_bucket: secrets-dst-bucket-us-east-1
        kms_arn: arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee

  - name: us-west-2
    containers:
    - <<: *CONTAINER_BASE
      dst_env_file:
        s3_bucket: secrets-dst-bucket-us-west-2
        kms_arn: arn:aws:kms:us-west-2:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeef
```

Source 2: DIY secrets
---------------------

In the above flow porter is using the information in `src_env_file` to retrieve
secrets and generate - for itself, internally - the format below:

```json
{
  "container_secrets": {
    "us-west-2": {
      "primary": {
        "SUPER_SECRET_SECRET": "dont_tell_anyone",
        "FOO": "bar"
      }
    }
  }
}
```

where

- `container_secrets` is a reserved key that must exist
- `us-west-2` is a region with one or more containers deployed to it
- `primary` is the default name given to the container in a single-container deployment
- `SUPER_SECRET_SECRET` is a secret key and `dont_tell_anyone` is a secret value

In the DIY flow you generate the same format and pass it to porter.

Here's a sample `.porter/config` to use with the DIY flow.

```yaml
environments:
- name: prod
  regions:
  - name: us-west-2
    containers:
    - topology: inet
      dst_env_file:
        s3_bucket: secrets-dst-bucket
        kms_arn: arn:aws:kms:us-west-2:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeef
```

In the DIY flow you can do the following to achieve the same thing but with your
own secret storage and retrieval.

Let's say you have a tool called `secrets-retriever` that get secrets from your
long-term secrets store, formats it to the above format, and prints it to stdout
like this.

```bash
$ secrets-retriever
{
  "container_secrets": {
    "us-west-2": {
      "primary": {
        "SUPER_SECRET_SECRET": "dont_tell_anyone",
        "FOO": "bar"
      }
    }
  }
}
```

You can pipe that output to `porter build provision` like this

```bash
secrets-retriever | porter build provision -e prod
```

Destination: S3 and EC2 initialization
--------------------------------------

Both sources of secrets lead here.

Now porter can help securely deliver secrets to a running container.

The tl;dr is (1) use a secondary symmetric key to encrypt secrets in transit, (2) don't store the lock and key together, and (3) don't persist anything to disk.

1. Porter running on a host you control ("the Build Box") generates a symmetric encryption key ("the Key") that is 256 bits in length and is generated from https://golang.org/pkg/crypto/rand/#Read.
1. Porter running on the Build Box encrypts the plain text secrets ("Plain Secrets") with AES-256 (https://golang.org/pkg/crypto/aes/#NewCipher) using the Key to create an encrypted byte array ("Encrypted Secrets")
1. Porter running on the Build Box provisions infrastructure by
  1. Uploading the Encrypted Secrets to a configurable S3 bucket (`dst_env_file.s3_bucket`). The S3 key is a MD5 digest of Encrypted Secrets. Together the bucket and key are the "S3 Location".
  1. Creating a CloudFormation Template ("the Template")
  1. Calling CloudFormation:CreateStack with
    1. The Template that has baked into it the S3 Location of Encrypted Secrets
    1. A CloudFormation parameter key “PorterSecretsKey” with parameter value = the Key
1. For each EC2 host that's provisioned, Porter running on the EC2 host
  1. Calls CloudFormation DescribeStack on its own CloudFormation template id (already known at runtime) to get the value of the parameter “PorterSecretsKey” which is the Key.
  1. Calls S3 GetObject on the S3 Location that was baked into the Template to get Encrypted Secrets
  1. Decrypts Encrypted Secrets with the Key to create Plain Secrets. None of Plain Secrets, Encrypted Secrets, or Key are persisted to disk.
  1. Starts the Docker container and injects Plain Secrets as environment variables

Where we finally end up is on a EC2 host running a simple docker command with the Plain Secrets, like this

```
docker run \
-e SUPER_SECRET_SECRET=dont_tell_anyone \
-e FOO=bar \
some_image
```

Summary
-------

Porter provides light integration with S3 to source secrets. It also provides a
way to plugin your own secrets.

In the S3 flow `src_env_file` is used.

The only configuration both flows require is `dst_env_file.s3_bucket`

`dst_env_file.kms_arn` is optionally configurable as an extra layer of protection.

Regardless of the use of SSE-KMS, porter does its own encryption of secrets in
transit, and separates the lock (encrypted payload in S3) from the key
(hex-encoded 256 bit key in a CloudFormation parameter). Under this scheme both
services would need to be compromised.

Obviously if the build box where secrets were first accessed, or the EC2 host
where porter and any other process have access to the lock and key, are
compromised then all bets are off.

Resiliency concerns
-------------------

It's important that a given deployment always use the same version of secrets it
was deployed with and that it's resilient to failure.

Consider the following scenario:

1. Secrets file `secrets.env-file`, version A is uploaded
1. EC2 instance initializes and pulls `secrets.env-file`
1. Secrets file `secrets.env-file`, version B is uploaded
1. An EC2 instance fails and the autoscaling group initializes another EC2 instance
1. EC2 instance initializes and pulls `secrets.env-file`
1. The EC2 instance now has version B of the env-file, not the one it was deployed with

To avoid this scenario (which also happens during a scale out) porter does a MD5
"fingerprint" of the encrypted secrets and copies them to a destination S3
bucket. The copy and fingerprint ensure that the failed instance in step 4
comes back online with the correct version of the secrets file.

Resources
---------

Some other options for secure secrets storage we know of are

- [Vault](https://www.vaultproject.io/)
- [Thycotic](https://thycotic.com/)
- [CyberArk](http://www.cyberark.com/)

For EC2 host hardening you might consider [CIS Amazon Linux](https://aws.amazon.com/marketplace/pp/B00UVT5UAK/ref=sp_mpg_product_title?ie=UTF8&sr=0-2)
