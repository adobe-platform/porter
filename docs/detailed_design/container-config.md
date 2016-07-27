Container config
================

Porter follows [12factor](http://12factor.net/config) when it comes to
configuring services. Often config contains long-term secrets like API keys that
shouldn't be baked into a container.

Porter provides a mechanism to securely inject environment variable at runtime.

This is an outline of the mechanism and required setup

1. Create a secrets file
1. Upload that file to a S3 bucket
1. Configure your container's source S3 bucket and key to the bucket and key you just uploaded to
1. Configure the env-file's destination S3 bucket and what KMS ARN porter should use to encrypt the contents of env-file
1. During `porter build provision` porter copies the source env-file to the destination env-file
1. During EC2 host initialization porter calls `GetObject` on the destination env-file and runs the container with the environment variables it finds

Creating a secrets file
-----------------------

This guide assumes the existence of

1. A S3 bucket to place secrets in called `secrets-src-bucket`
1. A S3 bucket to copy secrets into called `secrets-dst-bucket`. The source and destination can be the same bucket.
1. A KMS key with the ARN `arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee`

Porter expects the format Docker prescribes for [`--env-file`](https://docs.docker.com/engine/reference/commandline/run/#set-environment-variables-e-env-env-file)
to have been encrypted with [SSE-KMS](http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingKMSEncryption.html)

Here is a sample env file

```
API_KEY=secret
SOME_OTHER_CONFIG=foo
```

We'll call this file `secrets.env-file` and upload it to the bucket

```
aws s3api put-object \
--bucket secrets-bucket \
--key secrets.env-file \
--body secrete.env-file \
--server-side-encryption aws:kms \
--ssekms-key-id arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
```

Configure the container
-----------------------

In `.porter/config` set the `src_env_file` and `dst_env_file` properties

```yaml
containers:
- topology: inet
  src_env_file:
    s3_bucket: secrets-src-bucket
    s3_key: secrets.env-file
  dst_env_file:
    s3_bucket: secrets-dst-bucket
    kms_arn: arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
```

Deployment mechanisms
---------------------

Given the configuration and static resources (S3 buckets, KMS key, etc.) porter
has what it needs to inject `API_KEY` and `SOME_OTHER_CONFIG` into the container
at runtime

### provision

During `porter build provision` porter calls GetObject on `src_env_file.s3_key`
to get the decrypted contents of the env-file.

The decrypted contents aren't interpreted or saved. These bytes are uploaded to
the `s3_bucket` in `dst_env_file` and encrypted with `kms_arn`.

### EC2 initialization

The destination key was packaged up with the service payload. During EC2
initialization a simple `GetObject` is called on the key.

The contents of the file are split on `\n` and each line becomes an argument to
`docker run` like this

```
docker run \
-e API_KEY=secret \
-e SOME_OTHER_CONFIG=foo \
some_image
```

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

To avoid this scenario (which also happens during a scale out) the ETag of
version A becomes the key for a copied env-file. The copy ensures that the
failed instance in step 4 comes back online with the correct version of the
secrets file.

Multi-region container config
-----------------------------

Since SSE-KMS ties a S3 bucket to a KMS key per region, we need a way to define
container-specific config per region.

Here is an example config that deploys to 2 regions, us-west-2 and us-east-1,
and keeps secrets in the us-east-1 bucket.

```yaml
_container_base_definition: &CONTAINER_BASE
  topology: inet
  src_env_file:
    s3_bucket: secrets-src-bucket-us-east-1
    s3_key: secrets.env-file

environments:
- name: stage

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
