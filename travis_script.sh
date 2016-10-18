#!/bin/bash

# This script is a workaround for building on travis, where this is no
# native libalpm (travis is ubuntu-based).  Here we build the libalpm
# portion of pacman and then munge our environment variables so that
# cgo can find it.

set -ev

make data

make all
