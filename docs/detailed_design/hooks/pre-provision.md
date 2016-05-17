pre-provision
=============

Use Cases
---------

- Packaging and deployment of static assets that are compatible with the
  currently deployed service
- Ensuring non-ephemeral infrastructure is provisioned (e.g. RDS, DynamoDB)

Lifecycle
---------

This hook is called from `porter build provision` **for each** region
**before any** regions are provisioned
(as opposed to before each region is provisioned)

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

### Upload assets to S3

`.porter/hooks/pre-provision`

```Dockerfile
FROM python:2.7.11

RUN pip install awscli

ADD pre-provision.cmd /
RUN chmod 544 /pre-provision.cmd

WORKDIR /repo_root

CMD /pre-provision.cmd
```

`.porter/hooks/pre-provision.cmd`

```bash
gzip /index.html

aws s3api put-object \
  --bucket some-bucket \
  --key "index.html" \
  --content-type 'text/html' \
  --content-encoding 'gzip' \
  --body /index.html.gz
```
