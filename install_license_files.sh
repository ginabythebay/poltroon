#!/bin/bash
set -e

rm -rf data/
mkdir -p data
for f in $(find . -name "LICENSE"); do
    dest="data/$f"
    mkdir -p "$(dirname $dest)"
    cp "$f" "$dest"
done
