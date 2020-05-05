#!/bin/bash -e

CORETH_VER="v0.1.0"     # Must match coreth version in go.mod
CORETH_PATH=$GOPATH/pkg/mod/github.com/ava-labs/coreth@$CORETH_VER

SALTICIDAE_VER="v0.3.0" # Must match salticidae version in go.mod
SALTICIDAE_PATH=$GOPATH/pkg/mod/github.com/ava-labs/salticidae@$SALTICIDAE_VER

# Fetch Gecko dependencies, including salticidae-go and coreth
echo "Fetching dependencies..."
go mod download

# Make sure specified versions of salticidae and coreth exist
if [ ! -d $CORETH_PATH ]; then
    echo "couldn't find coreth version ${CORETH_VER}"
    echo "build failed"
    exit 1
elif [ ! -d $SALTICIDAE_PATH ]; then
    echo "couldn't find salticidae version ${SALTICIDAE_VER}"
    echo "build failed"
    exit 1
fi

# Build salticidae
echo "Building salticidae..."
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    chmod -R u+w $SALTICIDAE_PATH
    cd $SALTICIDAE_PATH
    cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$SALTICIDAE_PATH/build" .
    make -j4
    make install
    cd -
    export CGO_CFLAGS="-I$SALTICIDAE_PATH/build/include" # So Go compiler can find salticidae
    export CGO_LDFLAGS="-L$SALTICIDAE_PATH/build/lib/ -lsalticidae -luv -lssl -lcrypto -lstdc++"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew install Determinant/salticidae/salticidae
    export CGO_CFLAGS="-I/usr/local/opt/openssl/include"
    export CGO_LDFLAGS="-L/usr/local/opt/openssl/lib/ -lsalticidae -luv -lssl -lcrypto"
else 
    echo "Your operating system is not supported"
    exit 1
fi

# Build the binaries
GECKO_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd ) # Directory above this script
BUILD_DIR="${GECKO_PATH}/build" # Where binaries go

echo "Building Gecko binary..."
go build -o "$BUILD_DIR/ava" "$GECKO_PATH/main/"*.go

echo "Building throughput test binary..."
go build -o "$BUILD_DIR/xputtest" "$GECKO_PATH/xputtest/"*.go

echo "Building EVM plugin binary..."
go build -o "$BUILD_DIR/plugins/evm" "$CORETH_PATH/plugin/"*.go
