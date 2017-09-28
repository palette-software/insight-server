#!/bin/bash
set -e

# Upload the RPM to the RPM repository
# by exportin it to SSHPASS, sshpass wont log the command line and the password
export SSHPASS=$DEPLOY_PASS
sshpass -e scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -r rpm-build/_build/* $DEPLOY_USER@$DEPLOY_HOST:$DEPLOY_PATH

# Update the RPM repository
export DEPLOY_CMD="createrepo ${DEPLOY_PATH}/"
sshpass -e ssh  -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_HOST $DEPLOY_CMD

# Disable upload to Github releases, because we only need RPM packages at the moment
#./release-to-github.sh
