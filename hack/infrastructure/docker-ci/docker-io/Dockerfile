# VERSION:        0.1
# DOCKER-VERSION  0.6.6
# AUTHOR:         Daniel Mizyrycki <daniel@docker.com>
# DESCRIPTION:    docker.io testing
# COMMENTS:       Add docker.io tests on top of the docker-ci image
#     Configuration for docker-io.sh is obtained from docker-ci-data container
# TO_BUILD:       docker build -t docker-io .
# TO_RUN:         docker run -volumes-from docker-ci-data docker-io

FROM docker-ci
MAINTAINER Daniel Mizyrycki <daniel@docker.com>

# Add docker-io testing dependencies
RUN apt-get install -y --no-install-recommends gcc libmysqlclient-dev
RUN pip install mock

# Set San Francisco timezone
RUN echo "America/Los_Angeles" >/etc/timezone
RUN dpkg-reconfigure --frontend noninteractive tzdata

ADD docker-io.sh /usr/bin/docker-io.sh
RUN chmod +x /usr/bin/docker-io.sh

CMD "/usr/bin/docker-io.sh"
