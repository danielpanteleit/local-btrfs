#!/bin/bash

set -eux

docker build -f Dockerfile-test -t local-btrfs-test .
docker run --privileged --rm local-btrfs-test
