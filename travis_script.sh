#!/bin/bash

# This script is a workaround for building on travis, where this is no
# native libalpm (travis is ubuntu-based).  Here we build the libalpm
# portion of pacman and then munge our environment variables so that
# cgo can find it.

set -ev

# TODO(gina) delete this and use apt once
# https://github.com/travis-ci/apt-package-whitelist/issues/3417 is
# fixed.
./travis_install_gobindata.sh
export PATH="$HOME/gosupport/bin:$PATH"

make data

make all
