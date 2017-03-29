#!/bin/bash

set -eux

dd if=/dev/zero of=/btrfs.img bs=1M count=0 seek=100
mkfs.btrfs /btrfs.img
mount -o loop /btrfs.img /btrfs

PKGS=$(find . -type f -name '*.go' | sed -r 's|/[^/]+$||' |sort |uniq | grep -v "^./vendor")
go test -v $PKGS
