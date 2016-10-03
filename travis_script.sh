#!/bin/bash

# This script is an attempt to use libalpm, which we build in the
# travis_before_install.sh script.

set -ev

headerdir="$(pwd)/pacman/lib/libalpm/"
libdir="$(pwd)/pacman/lib/libalpm/.libs"
export CGO_CFLAGS="$CGO_CFLAGS -I${headerdir}"
export LDFLAGS="$LDFLAGS -L${libdir}"

# let me see what it up here
env

go install ./...
go test -v ./...
