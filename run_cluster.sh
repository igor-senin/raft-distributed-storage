#!/usr/bin/env bash

IMAGE="raft-replica-node"
BASE_NAME="replica-node"
NETWORK_NAME="clusternet"
INTERNAL_PORT=8786
HOST_BASE_PORT=8000
REPLICA_COUNT=6
IP_BASE="172.18.0"
OCTET_BASE=10
SUBNET="${IP_BASE}.0/16"

docker network inspect "$NETWORK_NAME" > /dev/null 2>&1 || \
  docker network create "$NETWORK_NAME" --subnet="${SUBNET}"

for (( i=0; i < $REPLICA_COUNT; i++ ))
do
  NAME="${BASE_NAME}-${i}"
  HOST_PORT=$(( HOST_BASE_PORT + i ))
  IP_LAST_OCTET=$(( OCTET_BASE + i ))
  IP="${IP_BASE}.${IP_LAST_OCTET}"

  echo "Starting ${NAME} on port ${HOST_PORT}..."

  docker container run \
    -d --rm \
    --name "${NAME}" \
    --network "${NETWORK_NAME}" \
    --ip="${IP}" \
    -p "${HOST_PORT}:${INTERNAL_PORT}" \
    "${IMAGE}"
done
