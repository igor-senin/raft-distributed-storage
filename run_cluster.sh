#!/usr/bin/env bash

set -e

IMAGE="raft-replica-node"
BASE_NAME="replica-node"
NETWORK_NAME="clusternet"
INTERNAL_PORT=8786
HOST_BASE_PORT=8000
REPLICA_COUNT=${1:-6}
IP_BASE="172.18.0"
OCTET_BASE=10
BASE_IP_ADDR="${IP_BASE}.${OCTET_BASE}"
SUBNET="${IP_BASE}.0/16"

docker network inspect "$NETWORK_NAME" > /dev/null 2>&1 || \
  docker network create "$NETWORK_NAME" --subnet="${SUBNET}"

for (( i=0; i < $REPLICA_COUNT; i++ ))
do
  NAME="${BASE_NAME}-${i}"
  HOST_PORT=$(( HOST_BASE_PORT + i ))
  IP_LAST_OCTET=$(( OCTET_BASE + i ))
  IP="${IP_BASE}.${IP_LAST_OCTET}"

  # Создаём директории для маунта контейнеров.
  VOLUME_NAME="volume-${NAME}"
  mkdir ${VOLUME_NAME}

  case "$i" in
    "0") IS_LEADER="--leader" ;;
      *) IS_LEADER="" ;;
  esac

  echo "Starting ${NAME} on port ${HOST_PORT}..."

  docker container run \
    -d --rm \
    --name "${NAME}" \
    --network "${NETWORK_NAME}" \
    --mount type=bind,src="./${VOLUME_NAME}",dst="/opt/volume" \
    --ip="${IP}" \
    -p "${HOST_PORT}:${INTERNAL_PORT}" \
    "${IMAGE}" \
      --clusterSize="${REPLICA_COUNT}" --idx="${i}" --baseAddr="${BASE_IP_ADDR}" ${IS_LEADER}
done
