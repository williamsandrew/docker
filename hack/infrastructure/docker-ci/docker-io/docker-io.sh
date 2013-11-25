#!/bin/sh

set -x

# docker-io repository key is stored in docker-io-data container with path:
DOCKER_IO_KEY_PATH='/data/docker-ci/docker-io.key'

# Setup the environment
DOCKER_IO_PATH=/data/docker-io
export PYTHONPATH=$DOCKER_IO_PATH/test

# Set ssh key for github retrieve
mkdir /root/.ssh
python -c "
import base64;
key = base64.b64decode(open(\"$DOCKER_IO_KEY_PATH\").read());
open('/root/.ssh/id_rsa','w').write(key)"
chmod 0600 /root/.ssh/id_rsa
echo StrictHostKeyChecking no > /root/.ssh/config

# Fetch latest docker-io master and dependencies
rm -rf $DOCKER_IO_PATH
git clone git@github.com:dotcloud/docker-io.git -b master $DOCKER_IO_PATH

cd $DOCKER_IO_PATH
pip install -r requirements.txt

python docker_io/manage.py test --settings=settings.tests
