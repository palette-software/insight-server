#!/usr/bin/env bash

# Stop on the first error
set -e

if [ "$#" -ne 1 ]; then
    echo "Error! This script takes exactly one argument! Exiting."
    exit 1
fi

LICENSE_KEY=$1

# Replace the license key in Insight Server's config file
sed -i 's/^license_key=.*$/license_key='${LICENSE_KEY}'/m' /etc/palette-insight-server/server.config

# Restart Insight Server
supervisorctl restart palette-insight-server
