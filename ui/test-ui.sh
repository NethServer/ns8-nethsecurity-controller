#!/bin/bash

#
# Copyright (C) 2026 Nethesis S.r.l.
# SPDX-License-Identifier: GPL-3.0-or-later
#

#
# Hint: access the test logs on HTTP port 8000 with this command:
#
#     python -mhttp.server -d tests/outputs/ 8000 &
#

set -e -a

SSH_KEYFILE=${SSH_KEYFILE:-$HOME/.ssh/id_rsa}

LEADER_NODE="${1:?missing LEADER_NODE argument}"
IMAGE_URL="${2:?missing IMAGE_URL argument}"
shift 2

ssh_key="$(< $SSH_KEYFILE)"

mkdir -vp tests/outputs/
trap 'podman cp --overwrite rftest-ui:/home/pwuser/outputs/ tests/' EXIT
podman run -i \
    --network=host \
    --volume=.:/home/pwuser/source:z \
    --replace --name=rftest-ui \
    --env=ssh_key \
    --env=LEADER_NODE \
    --env=IMAGE_URL \
    ghcr.io/marketsquare/robotframework-browser/rfbrowser-stable:19.11.0 \
    bash -l -s -- "${@}" <<'EOF'
set -e
echo "$ssh_key" > /tmp/idssh
mkdir -vp ~/outputs
cd /home/pwuser/source # ui/ is the base dir
pip3 install -q -r tests/pythonreq.txt
exec robot \
    -v NODE_ADDR:${LEADER_NODE} \
    -v IMAGE_URL:${IMAGE_URL} \
    -v SSH_KEYFILE:/tmp/idssh \
    --name test-ui \
    --skiponfailure unstable \
    -d /home/pwuser/outputs "${@}" tests/
EOF
