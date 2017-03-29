#!/bin/bash

set -eux

docker build -f Dockerfile-test -t local-btrfs-test .
docker run --privileged --rm -v `pwd`:/go/src/github.com/danielpanteleit/local-btrfs:ro -v `pwd`/bin:/go/src/github.com/danielpanteleit/local-btrfs/bin local-btrfs-test
