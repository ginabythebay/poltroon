# poltroon


[![Build Status](https://travis-ci.org/ginabythebay/poltroon.svg?branch=master)](https://travis-ci.org/ginabythebay/poltroon)

Foolishly updates existing packages from AUR (arch linux only)

1. Runs `pacman --query --foreign` to find already-installed packages
   are not in a sync database.  Uses the
   [aurweb RPC Interface](https://aur.archlinux.org/rpc.php) to see
   which of those packages have newer versions available.
2. Asks if the user wants to proceed.  Exits if they don't.
3. Starts a two-stage pipeline.
4. In the first stage, we run cower -d to download the package (default it two workers).
5. In the second state, we run makepkg -s to build the package files.
6. At the end, we print out the command the user can run to install the packages.

All the action happens in /tmp/poltroon/ with a sub-directory for each package and a logs directory within that that can be inspected.

Inspired by [cower](https://github.com/falconindy/cower), extending
the idea even further.  Currently depends on having cower installed, but I am hoping to remove that dependency soon.
