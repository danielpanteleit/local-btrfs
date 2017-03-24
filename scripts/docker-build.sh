#!/usr/bin/env bash

set -e

echo "building image"
LOCAL_BTRFS=$(docker build -q -f Dockerfile-build .)
echo "compiling"
docker run -it --rm -v `pwd`/bin:/go/src/local-btrfs/bin $LOCAL_BTRFS
echo "removing image"
docker rmi $LOCAL_BTRFS
