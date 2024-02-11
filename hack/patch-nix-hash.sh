#!/usr/bin/env bash
# This script automatically updates the Nix hash.
# Needed because dependabot automatically updates Go deps.

# this is reliant on the vendorHash being on line 195
# see https://stackoverflow.com/questions/5694228/sed-in-place-flag-that-works-both-on-mac-bsd-and-linux
sed -i.bak '195s/vendorHash = \"\([^\"]*\)\"/vendorHash = ""/g' ./dev/nix/flake.nix
NIX_HASH=$(nix --log-format raw build ./dev/nix 2>&1 | grep  "got: " | awk '{ split($0,a,"got:    "); print a[2] }')
sed -i.bak '195s|vendorHash = \"\([^\"]*\)\"|vendorHash = "'$NIX_HASH'"|g' ./dev/nix/flake.nix
echo "Changed Nix hash to : $NIX_HASH"
rm -rf result
rm ./dev/nix/flake.nix.bak
