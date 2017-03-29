#!/bin/bash

set -eux

docker build -f Dockerfile-test -t local-btrfs-test .
docker run --privileged --rm -v `pwd`:/go/src/github.com/danielpanteleit/local-btrfs:ro local-btrfs-test
