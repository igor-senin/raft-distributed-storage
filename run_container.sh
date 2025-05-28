#!/usr/bin/env bash

set -e

if [ "$#" -ne 2 ]; then
  echo "illegal number of arguments"
  exit 1
fi

IMAGE="raft-replica-node"
BASE_NAME="replica-node"
NETWORK_NAME="clusternet"
INTERNAL_PORT=8786
HOST_BASE_PORT=8000
IP_BASE="172.18.0"
OCTET_BASE=10
BASE_IP_ADDR="${IP_BASE}.${OCTET_BASE}"

REPLICA_INDEX=${1}
REPLICA_COUNT=${2}

NAME="${BASE_NAME}-${REPLICA_INDEX}"
HOST_PORT=$(( HOST_BASE_PORT + REPLICA_INDEX ))
IP_LAST_OCTET=$(( OCTET_BASE + REPLICA_INDEX ))
IP="${IP_BASE}.${IP_LAST_OCTET}"

docker container run \
-it --rm \
--name "${NAME}" \
--network "${NETWORK_NAME}" \
--mount type=bind,src="./${VOLUME_NAME}",dst="/opt/volume" \
--ip="${IP}" \
-p "${HOST_PORT}:${INTERNAL_PORT}" \
"${IMAGE}" \
  --clusterSize="${REPLICA_COUNT}" --idx="${REPLICA_INDEX}" --baseAddr="${BASE_IP_ADDR}"
