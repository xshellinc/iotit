#!/bin/bash

vFile=".version"
cd "$(dirname "$0")"

gitRepo=$(git branch | grep \* | cut -d ' ' -f2-)
gitCommit=$(git log --format="%H" -n 1 | tail)

if test -r "$vFile"; then
    v=$(head -n 1 "$vFile")
fi

if [ -z $v ]; then
    echo "Cannot read version from '.version' file"
    exit 2
fi

if [ $gitRepo != "master" ]; then
    v=$v"_"$gitRepo"_"$gitCommit
fi

go build -ldflags "-X main.Version=$v -X main.Env=dev" iotit.go
