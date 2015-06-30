#!/usr/bin/env bash

set -e

TAG=christianbladescb/etcdbk

docker pull centurylink/golang-builder
docker run --rm \
  -v $(pwd):/src \
  -v /var/run/docker.sock:/var/run/docker.sock \
  centurylink/golang-builder \
  ${TAG}

if [[ -n "$1" && "$1" = "push" ]]; then
    docker push ${TAG}
fi
