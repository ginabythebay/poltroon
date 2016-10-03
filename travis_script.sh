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
libdir="$TRAVIS_BUILD_DIR/pacman/lib/libalpm/"
export CGO_CFLAGS="$CGO_CFLAGS -I${headerdir}"
export CGO_LDFLAGS="$CGO_LDFLAGS -L${libdir}"

# let me see what it set up here
ls -l pacman/lib/libalpm/*
ls -l pacman/lib/libalpm/.libs/*
env

go get -t -v ./...
go install ./...
go test -v ./...
