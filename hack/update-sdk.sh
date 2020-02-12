#!/usr/bin/env bash
set -eux -o pipefail

lang=$1
git_branch=$2
version=$3

git_remote=origin
git_repo=argo-workflows-${lang}-sdk
path=sdks/${git_repo}

# init submodule
git submodule update --init ${path}

# reset to latest on branch
cd ${path}
git fetch ${git_remote}
git checkout ${git_branch} && git reset --hard ${git_remote}/${git_branch} || git checkout -b ${git_branch} && git push -u ${git_remote} ${git_branch}
git clean -fxd
cd -

# generate code
java -jar dist/swagger-codegen.jar generate -i api/argo-server/swagger.json -l ${lang} -DhideGenerationTimestamp=true -o ${path} --git-user-id argoproj-labs --git-repo-id ${git_repo}

# commit, tag and push the changes
cd ${path}

git add .
git diff --quiet || git commit -m "Updated to ${version}"

if [[ ${version} != ${git_branch} ]]; then
    git tag ${version}
fi

git push -u ${git_remote} ${git_branch} --follow-tags
