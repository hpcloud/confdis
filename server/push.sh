#!/usr/bin/env bash
set -e # Exit on errors

# Tag your local image to DOCKER_REGISTERY/image and push it
: ${EMAIL?"Need to set EMAIL (e.g. export EMAIL=username@activestate.com)"}
: ${REGISTRY_USERNAME?"Need to set REGISTRY_USERNAME (e.g. export REGISTRY_USERNAME=your_username)"}
: ${REGISTRY_PASSWORD?"Need to set REGISTRY_PASSWORD (e.g. export REGISTRY_PASSWORD=your_password)"}

image="$1"           # Name of the image to push
docker_registry="$2" # Address of the docker registry

# Check if a docker registry is already in the image name
if echo ${image} | grep --quiet ${docker_registry}; then
        >&2 echo "Docker registry ${docker_registry} found in the image $image"
        exit 1
fi

docker tag ${image} ${docker_registry}/${image}
docker login --email="$EMAIL" --username="$REGISTRY_USERNAME" --password="$REGISTRY_PASSWORD" $docker_registry
docker push ${docker_registry}/${image}

