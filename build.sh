#!/bin/bash

vFile=".version"
cd "$(dirname "$0")"

if test -r "$vFile"; then
    v=$(head -n 1 "$vFile")
fi

if [ -z $v ]; then
    echo "Cannot read version from '.version' file"
    exit 2
fi

go build -ldflags "-X main.Version=$v -X main.Env=dev" iotit.go
