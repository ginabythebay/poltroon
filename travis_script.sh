#!/bin/bash

# This script is an attempt to use libalpm, which we build in the
# travis_before_install.sh script.

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
export LDFLAGS="$LDFLAGS -L${libdir}"

# let me see what it set up here
tree pacman/lib/libalpm
env

go get -t -v ./...
go install ./...
go test -v ./...
