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
venvroot=/usr/local/venv

podman run -i \
    --userns=keep-id \
    --user "$(id -u):$(id -g)" \
    --volume=.:/srv/source:z \
    --volume=rftest-ui-cache:${venvroot}:z \
    --replace --name=rftest-ui \
    --env=ssh_key \
    --env=venvroot \
    --env=LEADER_NODE \
    --env=IMAGE_URL \
    ghcr.io/marketsquare/robotframework-browser/rfbrowser-stable:19.11.0 \
    bash -l -s -- "${@}" <<'EOF'
set -e
echo "$ssh_key" > /tmp/idssh
if [ ! -x ${venvroot}/bin/robot ] ; then
    python3 -mvenv ${venvroot} --upgrade
    ${venvroot}/bin/pip3 install -q -r /srv/source/tests/pythonreq.txt
fi
cd /srv/source
mkdir -vp tests/outputs/
exec ${venvroot}/bin/robot \
    -v NODE_ADDR:${LEADER_NODE} \
    -v IMAGE_URL:${IMAGE_URL} \
    -v SSH_KEYFILE:/tmp/idssh \
    --name test-ui \
    --skiponfailure unstable \
    -d tests/outputs "${@}" tests/
EOF
