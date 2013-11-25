# VERSION:        0.1
# DESCRIPTION:    Create data image sharing /data volume
# AUTHOR:         Daniel Mizyrycki <daniel@dotcloud.com>
# COMMENTS:
#   This image is used to store docker-ci credentials
#   /data volume is owned by sysadmin.
# USAGE:
#   # Build data image
#   docker build -t docker-ci-data -rm .
#
#   # Create data container
#   docker run -name docker-ci-data docker-ci-data true
#
#   # List data from it
#   docker run -volumes-from docker-ci-data busybox ls -al /data

DOCKER-VERSION 0.6.5

# Smallest base image, just to launch a container
FROM busybox
MAINTAINER	Daniel Mizyrycki <daniel@docker.com>

# Create a regular user
RUN echo 'sysadmin:x:1000:1000::/data:/bin/sh' >> /etc/passwd
RUN echo 'sysadmin:x:1000:' >> /etc/group

# Create directory for that user
RUN mkdir -p /data/docker-ci
RUN chown sysadmin.sysadmin /data/docker-ci

# Add credentials to /data.
ADD docker-io.key /data/docker-ci/docker-io.key

# Create /data volume
VOLUME /data
