#!/usr/bin/env bash

set -e

echo "building image"
docker build -f Dockerfile-build -t local-btrfs-build .
echo "compiling"
docker run -it --rm -v `pwd`:/go/src/github.com/danielpanteleit/local-btrfs:ro -v `pwd`/bin:/go/src/github.com/danielpanteleit/local-btrfs/bin local-btrfs-build
