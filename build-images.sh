#!/bin/bash

# Terminate on error
set -e

# Prepare variables for later use
images=()
# The image will be pushed to GitHub container registry
repobase="${REPOBASE:-ghcr.io/nethserver}"
# Configure the image name
reponame="nethsec-controller"
tag=${IMAGE_TAG:-latest}

# Create a new empty container image
container=$(buildah from scratch)

# Reuse existing nodebuilder-nethsec-controller container, to speed up builds
if ! buildah containers --format "{{.ContainerName}}" | grep -q nodebuilder-nethsec-controller; then
    echo "Pulling NodeJS runtime..."
    buildah from --name nodebuilder-nethsec-controller -v "${PWD}:/usr/src:Z" docker.io/library/node:18.13.0-alpine
fi

echo "Build static UI files with node..."
buildah run \
    --workingdir "/usr/src/ui" \
    --env "NODE_OPTIONS=--openssl-legacy-provider" \
    nodebuilder-nethsec-controller \
    sh -c "yarn install --frozen-lockfile && yarn build"

# Add imageroot directory to the container image
buildah add "${container}" imageroot /imageroot
buildah add "${container}" ui/dist /ui
# Setup the entrypoint, ask to reserve one TCP port with the label and set a rootless container
buildah config --entrypoint=/ \
    --label="org.nethserver.authorizations=traefik@any:routeadm node:fwadm" \
    --label="org.nethserver.tcp-ports-demand=5" \
    --label="org.nethserver.rootfull=1" \
    --label="org.nethserver.images=ghcr.io/nethserver/nethsec-vpn:$tag ghcr.io/nethserver/nethsec-api:$tag ghcr.io/nethserver/nethsec-ui:$tag ghcr.io/nethserver/nethsec-proxy:$tag docker.io/grafana/promtail:2.7.1" \
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
