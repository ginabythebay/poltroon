#!/bin/bash

# This script is an attempt to make libalpm available on ubuntu (what
# travis uses)

set -ev

git clone git://projects.archlinux.org/pacman.git pacman

cd ./pacman
./autogen.sh
./configure

cd lib/libalpm
make
