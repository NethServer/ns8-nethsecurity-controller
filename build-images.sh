#!/bin/bash

# Terminate on error
set -e

# Prepare variables for later use
images=()
# The image will be pushed to GitHub container registry
repobase="${REPOBASE:-ghcr.io/nethserver}"
# Configure the image name
reponame="nethsecurity-controller"
controller_version=0.0.1
promtail_version=2.7.1
loki_version=2.9.4
prometheus_version=2.50.1
grafana_version=10.3.3
webssh_version=1.6.2

# Create a new empty container for webssh
echo "Build webssh container" # from https://github.com/huashengdun/webssh
mkdir -p webssh_tmp/build
pushd webssh_tmp
wget https://github.com/huashengdun/webssh/archive/refs/tags/v${webssh_version}.tar.gz -O webssh.tar.gz
tar xvzf webssh.tar.gz -C build --strip-components=1
pushd build
webssh=$(buildah from docker.io/python:3-alpine)
buildah add "${webssh}" . /code
buildah run \
    --workingdir '/code' \
    ${webssh} \
    sh -c "apk add --no-cache libc-dev libffi-dev gcc && \
    pip install -r requirements.txt --no-cache-dir && \
    apk del gcc libc-dev libffi-dev && \
    addgroup webssh && \
    adduser -Ss /bin/false -g webssh webssh && \
    chown -R webssh:webssh /code
    "
popd
popd
rm -rf webssh_tmp
# Commit the image
buildah config --workingdir '/code'  --entrypoint='["python", "run.py"]'  "${webssh}"
buildah commit "${webssh}" "${repobase}/webssh"

# Append the image URL to the images array
images+=("${repobase}/webssh")

# Create a new empty container image
container=$(buildah from scratch)

# Reuse existing nodebuilder-nethsecurity-controller container, to speed up builds
if ! buildah containers --format "{{.ContainerName}}" | grep -q nodebuilder-nethsecurity-controller; then
    echo "Pulling NodeJS runtime..."
    buildah from --name nodebuilder-nethsecurity-controller -v "${PWD}:/usr/src:Z" docker.io/library/node:18.13.0-alpine
fi

echo "Build static UI files with node..."
buildah run \
    --workingdir "/usr/src/ui" \
    --env "NODE_OPTIONS=--openssl-legacy-provider" \
    nodebuilder-nethsecurity-controller \
    sh -c "yarn install --frozen-lockfile && yarn build"

# Add imageroot directory to the container image
buildah add "${container}" imageroot /imageroot
buildah add "${container}" ui/dist /ui
# Setup the entrypoint, ask to reserve one TCP port with the label and set a rootless container
buildah config --entrypoint=/ \
    --label="org.nethserver.authorizations=traefik@any:routeadm node:tunadm" \
    --label="org.nethserver.tcp-ports-demand=10" \
    --label="org.nethserver.images=ghcr.io/nethserver/nethsecurity-vpn:$controller_version ghcr.io/nethserver/nethsecurity-api:$controller_version ghcr.io/nethserver/nethsecurity-ui:$controller_version ghcr.io/nethserver/nethsecurity-proxy:$controller_version docker.io/grafana/promtail:$promtail_version docker.io/grafana/loki:$loki_version docker.io/prom/prometheus:v$prometheus_version docker.io/grafana/grafana:$grafana_version ghcr.io/nethserver/webssh:${IMAGETAG:-latest}" \
    "${container}"
# Commit the image
buildah commit "${container}" "${repobase}/${reponame}"

# Append the image URL to the images array
images+=("${repobase}/${reponame}")

#
# NOTICE:
#
# It is possible to build and publish multiple images.
#
# 1. create another buildah container
# 2. add things to it and commit it
# 3. append the image url to the images array
#

#
# Setup CI when pushing to Github.
# Warning! docker::// protocol expects lowercase letters (,,)
if [[ -n "${CI}" ]]; then
    # Set output value for Github Actions
    printf "::set-output name=images::%s\n" "${images[*],,}"
else
    # Just print info for manual push
    printf "Publish the images with:\n\n"
    for image in "${images[@],,}"; do printf "  buildah push %s docker://%s:%s\n" "${image}" "${image}" "${IMAGETAG:-latest}" ; done
    printf "\n"
fi
