#!/bin/sh

PWD=$(pwd)

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

VERSION=$(cat ../assets/VERSION)

rpmbuild -bb\
  --buildroot $PWD\
  --define "version ${VERSION}"\
  --define "_rpmdir $PWD"\
  palette-insight-server.spec

# Clean up the binary directory
rm -rfv $BIN_DIR/*
rm -v $CONFIG_DIR/server.config

