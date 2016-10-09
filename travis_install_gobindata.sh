#!/bin/bash


# This script installed go-bindata so we can build on travis.

# TODO(gina) delete this and use apt once
# https://github.com/travis-ci/apt-package-whitelist/issues/3417 is
# fixed.

set -ev

export GOPATH=$TRAVIS_BUILD_DIR/gosupport
mkdir -p $GOPATH/src

cd $GOPATH/src

pkg="go-bindata"
version="3.0.5"
wget "https://github.com/jteeuwen/$pkg/archive/v$version.tar.gz"

tar xfz v$version.tar.gz
cd $pkg
go install ./...


