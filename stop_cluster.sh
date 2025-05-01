#!/usr/bin/env bash

BASE_NAME="replica-node"
REPLICA_COUNT=6

for (( i=0; i < $REPLICA_COUNT; i++ ))
do
  NAME="${BASE_NAME}-${i}"

  echo "Stoping container ${NAME}..."

  docker container stop "${NAME}"
done
