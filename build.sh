#!/usr/bin/env bash

set -e

go build .
docker build . --tag raft-replica-node
