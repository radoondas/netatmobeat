FROM golang:1.10.8
MAINTAINER Rado Ondas <ondas.radovan@gmail.com>

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         python-pip \
         virtualenv \
      && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip
RUN pip install --upgrade setuptools
RUN pip install --upgrade docker-compose==1.21.0
