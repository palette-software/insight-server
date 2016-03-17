#!/bin/sh

PWD=$(pwd)

rpmbuild -bb\
  --buildroot $PWD\
  --define "_rpmdir $PWD/_build"\
  palette-insight-certs.spec

