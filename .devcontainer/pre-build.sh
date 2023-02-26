#!/usr/bin/env sh
set -eux

# install kit
curl -q https://raw.githubusercontent.com/kitproj/kit/main/install.sh | sh

pwd
ls

# do time consuming tasks, e.g. download deps and initial build
CI=1 kit pre-test