#!/bin/bash

build_tools_image() {
  docker build -t argo-wf-tools -f ./hack/Dockerfile-tools .
}

run_mounted_command() {
  docker run \
    -it \
    --mount type=bind,source="$(pwd)",target=/go/src/github.com/argoproj/argo-workflows \
    argo-wf-tools \
    "$@"
}

prune_docker_images() {
  docker image prune -f
}

prune_docker_containers() {
  docker container prune -f
}

for arg in "$@"
do
  case $arg in
    tools-image)
      build_tools_image
    ;;
    run-cmd)
      build_tools_image
      run_mounted_command make codegen
      prune_docker_containers
      prune_docker_images
    ;;
    prune)
      prune_docker_containers
      prune_docker_images
    ;;
  esac
done
