#!/usr/bin/env bash

docker container run \
  -d --rm \
  --name replica-node-0 \
  -p 8786:8786/tcp \
  raft-replica-node:latest
