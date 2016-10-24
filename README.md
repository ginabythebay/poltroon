# poltroon


[![Build Status](https://travis-ci.org/ginabythebay/poltroon.svg?branch=master)](https://travis-ci.org/ginabythebay/poltroon)

Foolishly updates existing packages from AUR (arch linux only)

1. Runs `pacman --query --foreign` to find already-installed packages
   are not in a sync database.  Uses the
   [aurweb RPC Interface](https://aur.archlinux.org/rpc.php) to see
   which of those packages have newer versions available.
2. Asks if the user wants to proceed.  Exits if they don't.
3. Starts a two-stage pipeline.
4. In the first stage, we download the package and untar it. (default it two workers).
5. In the second state, we run makepkg -s to build the package files.
6. At the end, we print out the command the user can run to install the packages.

All the action happens in /tmp/poltroon/ with a sub-directory for each package and a logs directory within that that can be inspected.

Inspired by [cower](https://github.com/falconindy/cower), extending
the idea even further.

## Releasing

This is a way to release a new version

	github-release info --repo poltroon  # see current version
    VERSION="v0.2.0"  # or whatever
	git tag -a "$VERSION" -m "release $VERSION"
	git push --tags
	github-release info --repo poltroon
    github-release release --repo poltroon --tag "$VERSION" --pre-release

Also look into releasing binaries at some point.

## Future changes

I think this is good enough for me, for now.  Here are things I might
look at in the future.

* More testing.  If I mock out pacman, AUR, builds, I could have end to
  end testing.

* Simplify the pipeline.  Right now there is a two stage pipeline,
  which is more complication than it is worth.  Convert to a single
  state pipeline.

* Remove dependency on pacman (low priority).  We currently run
  `pacman --query --foreign` to find packages to consider updating.
  We could extend the alpm package to contain this functionality.
  Right now it doesn't seem worthwhile.  The current setup seems
  stable and fast.

* More automatic mode (medium priority).  Find a way to let the user
  look at all the PKGBUILD files beforehand, then run pacman
  automatically afterwards.  If we exec pacman after setting up the
  environment and the stdstreams, with sudo, it might do what I want?

* Look into running pacman with file:// urls.  This would make it cache old versions, which allows for automatic downgrades.  When I tried it, it complained about a lack of signatures, but there might be a flag that stops that.
