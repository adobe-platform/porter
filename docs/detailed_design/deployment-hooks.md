Deployment hooks
================

Deployment hooks enable services to program and gate their deployments. This is
often used for testing but can also be used for things like deploying static
assets to a CDN or building other types of automation into a deployment.

Hooks are Dockerfiles meaning hook authors can choose any language or runtime.

Hooks can be referenced in 2 ways:

1. By specifying the `dockerfile` for the hook
1. By specifying a git `repo` to be cloned, a `ref` to be checked out, and
   `dockerfile` to be built and run. These are called plugins.

All hooks are optional and will only be called if they exist. If they exist they
must exit with a code of 0 to continue the deployment. Any non-zero exit code is
considered a failure and will halt whatever command was called that caused the
hook to be called. Exceptions to the rule can be configured via [run conditions](#run-conditions).

There are currently 9 hooks porter defines. 8 of them are used during the 4
build phases (pre and post), the other is used to customize EC2 initialization.

The `pre` and `post` hooks are tied to the `porter build ...` command names, not
the underlying mechanisms of provisioning, promoting, etc.

The hooks
---------

- [pre-pack](hooks/pre-pack.md)
- [post-pack](hooks/post-pack.md)
- [pre-provision](hooks/pre-provision.md)
- [post-provision](hooks/post-provision.md)
- [pre-promote](hooks/pre-promote.md)
- [post-promote](hooks/post-promote.md)
- [pre-prune](hooks/pre-prune.md)
- [post-prune](hooks/post-prune.md)
- [ec2-bootstrap](hooks/ec2-bootstrap.md)

User-defined hooks
------------------

A hook named other than one of above is a user defined hook.  The internal
structure is the same as any other hook. A user-defined hook is never
automatically called by porter. It can only be called by the
`porter build hook` command.

**Example**

```yaml
hooks:
  alert_the_operator:
  - dockerfile: path/to/Dockerfile
    repo: https://github.com/person/repo.git
    ref: v1.0.0
```

This hook might be run by a build system as such:

```
porter build hook -name alert_the_operator -e Stage
```

Execution order
---------------

Multiple of each hook can be run. They are run in the order defined unless
[configured to run concurrently](config-reference.md#concurrent).

Hook environment
----------------

Each hook's Docker context is the directory containing the hook.

The repo's root is volume mapped to `/repo_root` so hooks can access and mutate
the contents of a repo.

Environment variables are injected as they are available.

See the [config reference](config-reference.md#hook-dockerfile) for more.

### Standard environment variables

These are available to all hooks and provided by porter.

```
PORTER_SERVICE_NAME
PORTER_SERVICE_VERSION (git sha)
DOCKER_ENV_FILE
HAPROXY_STATS_USERNAME
HAPROXY_STATS_PASSWORD
HAPROXY_STATS_URL
```

### Custom environment variables

You can whitelist what environment each hook receives with the same semantics as
[Docker Compose](https://docs.docker.com/compose/compose-file/#/environment)

This is a pre_pack hook with `FOO` set to `bar`, and `BAZ` set to the value of
`BAZ` when porter is run.

```yaml
hooks:
  pre_pack:
  - dockerfile: path/to/Dockerfile
    environment:
      FOO: bar
      BAZ:
```

Plugins
-------

Single hooks are hardly sufficient for non-trivial projects. Porter enables
hooks to be developed independently and referenced by porter projects in
`.porter/config`.

Plugins are porter hooks that live in a separate repo.

Plugins can run as part of more than one hook. By default porter looks for a
`dockerfile` value in the hook config. If this isn't found it defaults to
`Dockerfile`. Here is an example config for a made up plugin called
porter-contrib-foo that runs for both `pre_pack` and `post_pack`, and a
porter-contrib-bar that contains a single Dockerfile at its root (to illustrate
the behavior).

```yaml
hooks:
  pre_pack:
  - repo: git@github.com:adobe-platform/porter-contrib-foo.git
    ref: v1.0.0
    dockerfile: pre-pack
  post_pack:
  - repo: git@github.com:adobe-platform/porter-contrib-foo.git
    ref: v1.0.0
    dockerfile: post-pack
  - repo: git@github.com:adobe-platform/porter-contrib-bar.git
    ref: v1.0.0
    # This is the default if undefined
    # dockerfile: Dockerfile
```

`pre-pack` and `post-pack` are Dockerfiles found in the root directory of the
porter-contrib-foo repo. There's no restriction on the name or placement of
these Dockerfiles (i.e. `pre-pack` could also have been `some_dir/Dockerfile`)
meaning a single repo could contain many hooks.

pre hook vs. post hook
----------------------

When to use a pre hook vs a post hook is determined by what build command you
want the hook to operate as part of.

For example, `post-provision` and `pre-promote` aren't very different. The state
of the deployment hasn't changed and all of the same variables are available.
The only difference is that `post-provision` is called during `porter build
provision`, while `pre-promote` is called during `porter build promote`.

That difference is probably only meaningful if these commands are run on
different boxes, with different environment variables, or are otherwise
logically separate from the caller's perspective.

The recommendation is to use pre hooks and then add or change to post hooks as
hooks are better understood or requirements dictate.

Run conditions
--------------

For post-pack, post-provision, and post-promote, and post-prune you can set run
conditions so that these hooks may run regardless of failure in the command they
run after. The implicit run condition is `run_condition: pass` which means the
hook will only be run only if the build step passed.

Additional options are `run_condition: fail` and `run_condition: always` which
be run only on failure, and always, respectively.

Failure is defined as the underlying command failing meaning if a `pre-pack`
hooks fails, the packaging never takes place and `post-pack` is not called even
with a `run_condition: fail` or `run_condition: always`. If `pre-pack` succeeds
or is undefined and packaging fails, then all `post-pack` hooks with
`run_condition: fail` or `run_condition: always` are run.

The intent of `run_condition` is to allow scripts that may have altered some
state in a `pre-*` hook to be able to clean up if something goes wrong.
