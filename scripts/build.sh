#!/bin/bash -e

SALTICIDAE_GO_VER="v0.1.0" # Must match salticidae-go version in go.mod
export SALTICIDAE_GO_PATH=$GOPATH/pkg/mod/github.com/ava-labs/salticidae-go@$SALTICIDAE_GO_VER

# This script depends on CGO_CFLAGS and CGO_LDFLAGS, which are exported from the below env file.
# Those variables specify C dependencies of Gecko.
source ${SALTICIDAE_GO_PATH}/scripts/env.sh

CORETH_VER="v0.1.0" # Must match coreth version in go.mod
CORETH_PATH=$GOPATH/pkg/mod/github.com/ava-labs/coreth@$CORETH_VER

# Fetch Gecko dependencies, including salticidae-go and coreth
echo "Fetching dependencies..."
go mod download

# Build salticidae-go
echo "Building salticidae-go..."
bash $SALTICIDAE_GO_PATH/scripts/build.sh

# Build the binaries
# Directory above this build script
GECKO_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

BUILD_DIR="${GECKO_PATH}/build"

echo "Building Gecko binary..."
go build -o "$BUILD_DIR/ava" "$GECKO_PATH/main/"*.go

echo "Building throughput test binary..."
go build -o "$BUILD_DIR/xputtest" "$GECKO_PATH/xputtest/"*.go

echo "Building EVM plugin binary..."
go build -o "$BUILD_DIR/plugins/evm" "$CORETH_PATH/plugin/"*.go
