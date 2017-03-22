#!/bin/bash

set -eux

TEST_DIR=$(readlink -f $1)
MNT_POINT=$(stat -c %m $TEST_DIR)

REL_DIR=${TEST_DIR#$MNT_POINT/}
if [[ $REL_DIR == $TEST_DIR ]]; then
    echo "could not find relative path for $TEST_DIR using mountpoint $MNT_POINT"
    exit 1
fi

for vol in $(btrfs sub list $TEST_DIR | cut -f9 -d" " | grep ^$REL_DIR | sort -r); do
    btrfs sub del $MNT_POINT/$vol
done
