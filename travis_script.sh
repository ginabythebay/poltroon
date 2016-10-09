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

# TODO(gina) delete this and use apt once
# https://github.com/travis-ci/apt-package-whitelist/issues/3417 is
# fixed.
./travis_install_gobindata.sh
export PATH=$TRAVIS_BUILD_DIR/gosupport:$PATH

headerdir="$TRAVIS_BUILD_DIR/pacman/lib/libalpm"
libdir="$TRAVIS_BUILD_DIR/pacman/lib/libalpm/.libs"
export CGO_CFLAGS="$CGO_CFLAGS -I${headerdir}"
export CGO_LDFLAGS="$CGO_LDFLAGS -L${libdir}"

# Normally this would happen in the travis install step, but we
# skipped that before (it would have failed because libalpm wasn't
# available)
make data
go get -t -v ./...

make all
