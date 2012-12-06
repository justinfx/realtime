#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export GOPATH="$( dirname ${DIR} )"
export GOBIN="${GOPATH}/bin"
echo "Setting source location: $GOPATH"
echo "Installing to: $GOBIN"

go get -v github.com/justinfx/realtime/src/realtime


