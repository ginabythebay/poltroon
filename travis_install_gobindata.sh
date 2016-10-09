#!/bin/bash


# This script installed go-bindata so we can build on travis.

# TODO(gina) delete this and use apt once
# https://github.com/travis-ci/apt-package-whitelist/issues/3417 is
# fixed.

set -ev

pkg="go-bindata"
pth="github.com/jteeuwen/$pkg"
version="3.0.7"

export GOPATH="$HOME/gosupport"
mkdir -p "$(dirname $GOPATH/src/$pth)"

cd "$(dirname $GOPATH/src/$pth)"

wget "https://$pth/archive/v$version.tar.gz"

tar xfz v$version.tar.gz
mv $pkg-$version $pkg
cd $pkg
go install ./...


