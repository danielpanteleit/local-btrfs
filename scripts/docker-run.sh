#!/usr/bin/env bash

if [[ ! -e $DATA_VOLUME ]]; then
    echo Missing required environment variable: DATA_VOLUME
    exit 1
fi

CMD="docker run -d -v /run/docker/plugins/:/run/docker/plugins/ -v ${DATA_VOLUME}:${DATA_VOLUME} danielpanteleit/docker-local-btrfs-volume-plugin"

echo $CMD
exec $CMD
