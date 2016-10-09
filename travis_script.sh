#!/bin/bash

# This script is a workaround for building on travis, where this is no
# native libalpm (travis is ubuntu-based).  Here we build the libalpm
# portion of pacman and then munge our environment variables so that
# cgo can find it.

set -ev

cd ./pacman
./autogen.sh
./configure

cd lib/libalpm
make

cd $TRAVIS_BUILD_DIR

headerdir="$TRAVIS_BUILD_DIR/pacman/lib/libalpm"
libdir="$TRAVIS_BUILD_DIR/pacman/lib/libalpm/.libs"
export CGO_CFLAGS="$CGO_CFLAGS -I${headerdir}"
export CGO_LDFLAGS="$CGO_LDFLAGS -L${libdir}"

# Normally this would happen in the travis install step, but we
# skipped that before (it would have failed because libalpm wasn't
# available)
go get -t -v ./...

make all
