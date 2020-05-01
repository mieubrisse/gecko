#!/bin/bash -e

CORETH_VER="v0.1.0" # Must match coreth version in go.mod
SALTICIDAE_GO_VER="v0.2.0" # Must match salticidae-go version in go.mod

# Fetch Gecko dependencies, including salticidae-go and coreth
echo "Fetching dependencies..."
go mod download

export SALTICIDAE_GO_PATH=$GOPATH/pkg/mod/github.com/ava-labs/salticidae-go@$SALTICIDAE_GO_VER
if [ ! -f $SALTICIDAE_GO_PATH ]; then
    echo "couldn't find salticidae-go version ${SALTICIDAE_GO_VER}"
    echo "build failed"
    exit 1
fi
CORETH_PATH=$GOPATH/pkg/mod/github.com/ava-labs/coreth@$CORETH_VER
if [ ! -f $CORETH_PATH ]; then
    echo "couldn't find salticidae-go version ${CORETH_VER}"
    echo "build failed"
    exit 1
fi

# This script (specifically, building Gecko) depends on CGO_CFLAGS and CGO_LDFLAGS, 
# which are exported from the below env file.
# Those variables specify locations of C dependencies of Gecko (ie salticidae)
source ${SALTICIDAE_GO_PATH}/scripts/env.sh

# Build salticidae-go
echo "Building salticidae-go..."
chmod -R 744 $SALTICIDAE_GO_PATH
bash $SALTICIDAE_GO_PATH/scripts/build.sh

# Build the binaries
GECKO_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd ) # Directory above this script

BUILD_DIR="${GECKO_PATH}/build" # Where binaries go

echo "Building Gecko binary..."
go build -o "$BUILD_DIR/ava" "$GECKO_PATH/main/"*.go

echo "Building throughput test binary..."
go build -o "$BUILD_DIR/xputtest" "$GECKO_PATH/xputtest/"*.go

echo "Building EVM plugin binary..."
CORETH_PATH=$GOPATH/pkg/mod/github.com/ava-labs/coreth@$CORETH_VER
go build -o "$BUILD_DIR/plugins/evm" "$CORETH_PATH/plugin/"*.go
