#!/bin/bash

VERSION=$1

if [[ -z $VERSION ]]; then
  echo "Please provide the version you want as first argument to this script"
  exit 1
fi

if ! which glide &>/dev/null; then
  echo "You're missing glide. Please any key to run:"
  echo "curl https://glide.sh/get | sh"
  echo "..or CTRL+C to cancel and install manually"
  read -p
  curl https://glide.sh/get | sh
fi

sed -i "s/v0.7.11/v${VERSION}/" glide.yaml
glide update

go build
