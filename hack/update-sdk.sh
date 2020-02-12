#!/usr/bin/env bash
set -eux -o pipefail

lang=$1
git_branch=$2
version=$3

git_remote=origin
git_repo=argo-workflows-${lang}-sdk
path=sdks/${git_repo}

# init submodule
git submodule update --init -f ${path}

# reset to latest on branch
cd ${path}
git fetch ${git_remote}
git checkout ${git_branch} && git reset --hard ${git_remote}/${git_branch} || git checkout -b ${git_branch} && git push -u ${git_remote} ${git_branch}
git clean -fxd
rm -Rf *
cd -

# generate code
java -jar dist/openapi-generator-cli.jar generate \
    -i api/argo-server/swagger.json \
    -g ${lang} \
    -p hideGenerationTimestamp=true \
    -o ${path} \
    --invoker-package io.argoproj.argo.client \
    --api-package io.argoproj.argo.client.api \
    --model-package io.argoproj.argo.client.model \
    --group-id argoproj-labs \
    --artifact-id ${git_repo} \
    --artifact-version ${version} \
    --git-user-id argoproj-labs \
    --git-repo-id ${git_repo}

# commit, tag and push the changes
cd ${path}

git diff --quiet || ( git add . && git commit -m "Updated to ${version}" )

# tag, but only for v* versions
if [[ ${version} == v* ]]; then
    git tag -f ${version}
fi

git push --follow-tags
