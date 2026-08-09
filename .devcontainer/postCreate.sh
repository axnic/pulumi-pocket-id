#!/usr/bin/env bash
# Runs once when the devcontainer is created (cwd is the workspace folder,
# /workspace, per the Dev Containers spec). Gets the always-on Pocket-ID
# instance (docker-compose.yml, shared network namespace with this
# container) to a fully usable state with zero manual steps - unlike
# Garage, Pocket-ID's STATIC_API_KEY env var grants instant admin access,
# so there's no bootstrap step to run beyond installing the toolchain.
set -euo pipefail

# .config/mise.toml pulls in the community vfox-pulumi plugin. mise
# re-resolves that plugin reference on every invocation once a directory
# with the config in scope is used - including a first `mise install` -
# so it has to be installed once, from outside the repo, before anything
# else here touches mise.
(cd /tmp && mise plugins install vfox-pulumi https://github.com/pulumi/vfox-pulumi)

mise trust
mise install
