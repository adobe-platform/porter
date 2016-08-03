pre-pack
========

Use Cases
---------

- Unit tests
- Linting
- Static analysis
- Config generation

Lifecycle
---------

This hook runs only once during `porter build pack`.

Environment
-----------

[Standard](../deployment-hooks.md#standard-environment-variables)
and [Custom](../deployment-hooks.md#custom-environment-variables)
environment variables

Examples
--------

### Run unit tests

```Dockerfile
FROM golang:1.6

# switch to the volume-mapped /repo_root
WORKDIR /repo_root

# run tests
CMD go test
```

### Generate configuration

`.porter/config`

```yaml
hooks:
  pre_pack:
  - dockerfile: .porter/hooks/pre-pack
    environment:
      DYNAMIC_ELB_NAME:

environments:
- name: Prod

  regions:
  - name: us-west-2

    azs:
    - name: us-west-2a
    - name: us-west-2b
    - name: us-west-2c

    containers:
    - topology: inet

    elb: $DYNAMIC_ELB_NAME
```

`.porter/hooks/pre-pack`

```Dockerfile
FROM ubuntu:16.04

ADD pre-pack.cmd /
RUN chmod 544 /pre-pack.cmd

WORKDIR /repo_root/.porter

CMD /pre-pack.cmd
```

`.porter/hooks/pre-pack.cmd`

```bash
#!/bin/bash -e

render_template() {
  eval "echo \"$(cat $1)\""
}

CONF=$(render_template config)
echo "$CONF" > config
```

With the above config running

```bash
export DYNAMIC_ELB_NAME=some-elb-name
porter build pack
```

Will result in the hook being run with `DYNAMIC_ELB_NAME=some-elb-name` and it
writing the config which can be passed along to remaining `porter build`
commands.

This could have been done in any language. Here I use simple
[bash templating](http://pempek.net/articles/2013/07/08/bash-sh-as-template-engine/)
