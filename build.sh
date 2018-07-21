#!/bin/bash
set -e

cd "$(dirname "$0")"

gitRepo=$(git branch | grep \* | cut -d ' ' -f2-)
v=$(git describe)

if [ $gitRepo != "master" ]; then
    v=$v"_"$gitRepo
fi

go build -ldflags "-X main.version=$v -X main.Env=dev" iotit.go
