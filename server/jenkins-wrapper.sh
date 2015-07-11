#!/usr/bin/env bash
set -e # Exit on errors
IMAGE_NAME=$(./build.sh)

if [ "$PUSH_IMAGE" == "true" ]; then
        : ${DOCKER_REGISTRY?"Need to set DOCKER_REGISTRY (e.g. export REGISTRY_USERNAME=address of your docker-registry)"}
        ./push.sh $IMAGE_NAME $DOCKER_REGISTRY
fi

