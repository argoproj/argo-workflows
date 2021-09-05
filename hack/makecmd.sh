#!/bin/bash

build_tools_image() {
  docker build -t argo-wf-tools -f ./hack/Dockerfile-tools .
}

start_sync() {
  docker-sync start
}

start_sync_stack() {
  docker-sync-stack start
}

stop_sync() {
  docker-sync stop
}

prune_docker_images() {
  docker image prune -f
}

prune_docker_containers() {
  docker container prune -f
}

ensure_vendor() {
  go mod vendor
}

for arg in "$@"
do
  case $arg in
    tools-image)
      ensure_vendor
      build_tools_image
    ;;
    codegen)
      ensure_vendor
      build_tools_image
      start_sync
      start_sync_stack
      stop_sync
      prune_docker_containers
      prune_docker_images
    ;;
    prune)
      prune_docker_containers
      prune_docker_images
    ;;
  esac
done
