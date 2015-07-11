# Build the Dockerfile
# Return the name of the image
# e.g. IMAGE=$()
set -e # Exit on errors

docker_image_name="stackato/config-redis" 
git_branch="$(git rev-parse --abbrev-ref HEAD)"
git_short_hash="$(git rev-parse --short HEAD)"

docker_tag=${docker_image_name}:${git_branch}-${git_short_hash}

if docker build --rm=true --tag="${docker_tag}" . 1>&2; then
        echo ${docker_tag}
fi

