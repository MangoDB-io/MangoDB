# syntax=docker/dockerfile:1

FROM postgres:17.2 AS development

ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US
ENV LC_ALL=en_US.UTF-8
ENV LC_CTYPE=en_US.UTF-8

RUN --mount=type=cache,sharing=locked,target=/var/cache/apt <<EOF
set -ex

apt update
apt upgrade -y
apt install -y \
    build-essential \
    cmake \
    curl \
    git \
    libicu-dev \
    libkrb5-dev \
    pkg-config \
    postgresql-16-cron \
    postgresql-16-pgvector \
    postgresql-16-postgis-3 \
    postgresql-16-rum \
    postgresql-server-dev-16
EOF

# extra packages for development
RUN --mount=type=cache,sharing=locked,target=/var/cache/apt <<EOF
set -ex

apt install -y \
    bpftrace \
    gdb \
    postgresql-16-pgtap
EOF

RUN --mount=target=/src,rw <<EOF
set -ex

cd /src

mkdir -p /tmp/install_setup
cp build/postgres-documentdb/documentdb/scripts/* /tmp/install_setup/
cp build/postgres-documentdb/10-preload.sh build/postgres-documentdb/20-install.sql /docker-entrypoint-initdb.d/

cp build/postgres-documentdb/90-install-development.sql /docker-entrypoint-initdb.d/

export CLEANUP_SETUP=1
export INSTALL_DEPENDENCIES_ROOT=/tmp/install_setup

env MAKE_PROGRAM=cmake /tmp/install_setup/install_setup_libbson.sh
/tmp/install_setup/install_setup_pcre2.sh
/tmp/install_setup/install_setup_intel_decimal_math_lib.sh

cd build/postgres-documentdb/documentdb
make -k -j $(nproc) DEBUG=yes
make install

rm -fr /tmp/install_setup /var/lib/apt/lists/*
EOF

WORKDIR /

LABEL org.opencontainers.image.title="PostgreSQL+DocumentDB (development image)"
LABEL org.opencontainers.image.description="PostgreSQL with DocumentDB extension (development image)"
LABEL org.opencontainers.image.source="https://github.com/FerretDB/FerretDB"
LABEL org.opencontainers.image.url="https://www.ferretdb.com/"
LABEL org.opencontainers.image.vendor="FerretDB Inc."
