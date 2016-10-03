#!/bin/bash

# This script is an attempt to make libalpm available on ubuntu (what
# travis uses)

set -ev

# things we need for building libalpm
pkgs=asciidoc autopoint libarchive curl

sudo apt-get update -qq
sudo apt-get install -qq $pkgs
git clone git://projects.archlinux.org/pacman.git pacman

cd ./pacman
./autogen.sh
./configure
make
