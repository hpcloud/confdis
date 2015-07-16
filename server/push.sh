#!/usr/bin/env bash
set -e # Exit on errors

image="$1"           # Name of the image to push
docker_registry="$2" # Address of the docker registry

# Log in the docker registry
if ! echo | docker login $docker_registry; then
        : ${EMAIL?"Need to set EMAIL (e.g. export EMAIL=username@activestate.com)"}
        : ${REGISTRY_USERNAME?"Need to set REGISTRY_USERNAME (e.g. export REGISTRY_USERNAME=your_username)"}
        : ${REGISTRY_PASSWORD?"Need to set REGISTRY_PASSWORD (e.g. export REGISTRY_PASSWORD=your_password)"}

        docker login --email="$EMAIL" --username="$REGISTRY_USERNAME" --password="$REGISTRY_PASSWORD" $docker_registry
fi

# Tag the image with the docker registry if not already done
if ! echo ${image} | grep --quiet ${docker_registry}; then
        docker tag ${image} ${docker_registry}/${image}
fi

# Push the image to the docker registry
docker push ${docker_registry}/${image}

