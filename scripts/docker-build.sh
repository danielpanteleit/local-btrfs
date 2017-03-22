#!/usr/bin/env bash

set -e

echo "building image"
LOCAL_PERSIST=$(docker build -q -f Dockerfile-build .)
echo "compiling"
docker run -it --rm -v `pwd`/bin:/go/src/local-persist/bin $LOCAL_PERSIST
echo "removing image"
docker rmi $LOCAL_PERSIST
