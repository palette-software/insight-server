#!/bin/sh
VERSION="1.3.2"

SERVER_BINARY_SRC=../server/server
SERVER_BINARY=palette-insight-server

BIN_DIR=usr/local/bin
CONFIG_DIR=etc/palette-insight-server

# creat the temporary directories
mkdir -p $BIN_DIR
mkdir -p $CONFIG_DIR

# Copy the binary to its final destiantion
cp -v $SERVER_BINARY_SRC $BIN_DIR/$SERVER_BINARY
cp -v ../server/sample.config $CONFIG_DIR/server.config

rpmbuild -bb --buildroot $(pwd) palette-insight-server.spec

# Clean up the binary directory
rm -rfv $BIN_DIR/*
rm -v $CONFIG_DIR/server.config

