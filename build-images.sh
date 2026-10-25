#!/bin/bash

# Terminate on error
set -e

# Prepare variables for later use
images=()
# The image will be pushed to GitHub container registry
repobase="${REPOBASE:-ghcr.io/nethserver}"
# Configure the image name
reponame="nethsecurity-controller"
promtail_image="docker.io/grafana/promtail:3.6.11"
loki_image="docker.io/grafana/loki:2.9.17"
prometheus_image="docker.io/prom/prometheus:v3.15.0"
grafana_image="docker.io/grafana/grafana:13.2.4"
timescale_image="docker.io/timescale/timescaledb:2.31.0-pg16"

# Build controller service images
for service in vpn api ui proxy; do
    echo "Build nethsecurity-${service} container"
    buildah build --layers --target dist \
        --file "controller/${service}/Containerfile" \
        --tag "${repobase}/nethsecurity-${service}" \
        "controller/${service}"
    images+=("${repobase}/nethsecurity-${service}")
done

# Create a new empty container image
container=$(buildah from scratch)

# Reuse existing nodebuilder-nethsecurity-controller container, to speed up builds
if ! buildah containers --format "{{.ContainerName}}" | grep -q nodebuilder-nethsecurity-controller; then
    echo "Pulling NodeJS runtime..."
    buildah from --name nodebuilder-nethsecurity-controller -v "${PWD}:/usr/src:Z" docker.io/library/node:24.21.0-slim
fi

echo "Build static UI files with node..."
buildah run \
    --workingdir "/usr/src/ui" \
    --env "NODE_OPTIONS=--openssl-legacy-provider" \
    nodebuilder-nethsecurity-controller \
    sh -c "corepack enable && yarn install --frozen-lockfile && yarn build"

# Add imageroot directory to the container image
buildah add "${container}" imageroot /imageroot
buildah add "${container}" ui/dist /ui
# Setup the entrypoint, ask to reserve one TCP port with the label and set a rootless container
buildah config --entrypoint=/ \
    --label="org.nethserver.authorizations=traefik@any:routeadm node:tunadm,portsadm" \
    --label="org.nethserver.min-core=3.20.1" \
    --label="org.nethserver.tcp-ports-demand=11" \
    --label="org.nethserver.images=\
        ghcr.io/nethserver/nethsecurity-vpn:${IMAGETAG:-latest} \
        ghcr.io/nethserver/nethsecurity-api:${IMAGETAG:-latest} \
        ghcr.io/nethserver/nethsecurity-ui:${IMAGETAG:-latest} \
        ghcr.io/nethserver/nethsecurity-proxy:${IMAGETAG:-latest} \
        $promtail_image \
        $loki_image \
        $prometheus_image \
        $grafana_image \
        $timescale_image" \
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
