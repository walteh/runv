########################################################
# args
########################################################

ARG BUSYBOX_VERSION=1.36.1
ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.22
ARG DEBIAN_CODENAME=bookworm
ARG SOCAT_VERSION=1.8.0.3

ARG JQ_GIT_REF=master
ARG LSHW_GIT_REF=master
ARG E2FSPROGS_GIT_REF=master
ARG RUNC_GIT_REF=main

FROM alpine:${ALPINE_VERSION} AS alpine-ref
FROM debian:${DEBIAN_CODENAME}-slim AS debian-ref
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS alpine-golang-ref
FROM golang:${GO_VERSION}-${DEBIAN_CODENAME} AS debian-golang-ref

########################################################
# debian-based builder
########################################################

FROM debian-ref AS debian-base

ENV DEBIAN_FRONTEND=noninteractive \
	DEBCONF_NONINTERACTIVE_SEEN=true \
	LC_ALL=C.UTF-8 \
	LANG=C.UTF-8

RUN apt-get update \
	&& apt-get install -y \
		build-essential \
		autoconf \
		libtool \
		git \
		bison \
		flex \
		wget \
	&& rm -rf /var/lib/apt/lists/*

########################################################
# busybox builder
########################################################
FROM alpine-ref AS busybox-static-builder
RUN apk add --no-cache build-base wget tar linux-headers musl-dev musl-utils
WORKDIR /build
# Grab and unpack BusyBox source
RUN wget -qO- https://busybox.net/downloads/busybox-${BUSYBOX_VERSION}.tar.bz2 \
	| tar -xj
WORKDIR /build/busybox-${BUSYBOX_VERSION}

# 1  baseline defconfig
RUN make defconfig

# 2  apply overrides from repo
COPY busybox.fragment.config .config.fragment
RUN KCONFIG_ALLCONFIG=.config.fragment yes "" | make oldconfig

ARG TARGET=aarch64

RUN make -j$(nproc) CFLAGS="-static -fno-PIE" LDFLAGS="-static -no-pie"

RUN mkdir -p /rootfs/bin /rootfs/sbin /rootfs/usr/sbin /rootfs/usr/bin /rootfs/usr/local/bin
RUN cp busybox /rootfs/bin/ \
	&& chroot /rootfs /bin/busybox --install -s

RUN mkdir -p /rootfs/etc/network
RUN echo -e 'auto eth0\niface eth0 inet dhcp' > /rootfs/etc/network/interfaces


########################################################
# jq builder
########################################################

FROM debian-ref AS jq-static-builder

RUN git clone --depth 1 --branch ${JQ_GIT_REF} https://github.com/jqlang/jq.git /app
## have to skip building docs but I don't need them for my use case
RUN cd /app \
	&& git submodule init \
	&& git submodule update \
	&& autoreconf -i \
	&& ./configure --disable-valgrind --enable-all-static --disable-docs --disable-maintainer-mode --prefix=/usr/local \
	&& make -j$(nproc) LDFLAGS=-all-static \
	&& strip jq

RUN cd /app && make check || echo "check failed"

########################################################
# lshw builder
########################################################

FROM debian-ref AS lshw-static-builder

RUN git clone --depth 1 --branch ${LSHW_GIT_REF} https://github.com/lyonel/lshw.git /src
WORKDIR /src
# Build statically linked lshw
RUN make static
RUN strip src/lshw-static
RUN cp src/lshw-static /bin/lshw

########################################################
# socat builder
########################################################

FROM alpine/socat:${SOCAT_VERSION} AS socat-static-builder

########################################################
# e2fsprogs builder
########################################################

FROM alpine-ref AS e2fsprogs-static-builder
RUN apk add --no-cache build-base git autoconf automake linux-headers musl-dev
RUN git clone --depth 1 --branch ${E2FSPROGS_GIT_REF} https://git.kernel.org/pub/scm/fs/ext2/e2fsprogs.git /src
WORKDIR /src
# Pass STATIC LDFLAGS so no interpreter is needed
RUN CFLAGS='-O2' \
	LDFLAGS='-static -s' \
	./configure --enable-static --disable-shared
RUN make -j$(nproc)
RUN strip misc/mke2fs
RUN cp misc/mke2fs /bin/

########################################################
# runc builder
########################################################

FROM debian-golang-ref AS runc-static-builder
RUN apt-get update \
	&& apt-get install -y \
		build-essential \
		autoconf \
		libtool \
		git \
		bison \
		flex \
		wget \
		libseccomp-dev \
		libseccomp2 \
	&& rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch ${RUNC_GIT_REF} https://github.com/opencontainers/runc.git /src
WORKDIR /src
RUN make static
RUN strip runc
RUN cp runc /bin/runc

########################################################
# runm-linux-mounter builder
########################################################

FROM alpine-golang-ref AS runm-linux-mounter-builder

COPY --from=runm . /app/runm

WORKDIR /app/runm
RUN GOOS=linux GOARCH=arm64 go build -x -ldflags="-s -w -extldflags=-static" -trimpath -o /out/runm-linux-mounter ./cmd/runm-linux-mounter

########################################################
# runm-linux-init builder
########################################################

FROM alpine-golang-ref AS runm-linux-init-builder

COPY --from=runm . /app/runm

WORKDIR /app/runm
RUN GOOS=linux GOARCH=${ARCH} go build -x -ldflags="-s -w -extldflags=-static" -trimpath -o /out/runm-linux-init ./cmd/runm-linux-init


########################################################
# rootfs bins builder
########################################################

FROM scratch AS fs-builder


# our static binaries (as needed)
# COPY --from=socat-static-builder /usr/bin/socat /mbin/socat
# COPY --from=e2fsprogs-static-builder /bin/mke2fs /mbin/mke2fs
# COPY --from=lshw-static-builder /bin/lshw /mbin/lshw
COPY --from=runc-static-builder /bin/runc /mbin/runc
COPY --from=runm-linux-init-builder /out/runm-linux-init /mbin/runm-linux-init
COPY --from=runm-linux-mounter-builder /out/runm-linux-mounter /mbin/runm-linux-mounter

# busybox initalized filesystem
COPY --from=busybox-static-builder /rootfs/ /

# our raw files
COPY udhcpc.default /rootfs/etc/udhcpc/default.script
COPY resolv.conf /rootfs/etc/resolv.conf

########################################################
# cpio builder
########################################################

FROM alpine-ref AS cpio-builder
RUN apk add --no-cache cpio gzip
WORKDIR /work

COPY --from=fs-builder / /rootfs

RUN chroot /rootfs ln -s mbin/runm-linux-mounter /init

RUN mkdir -p /dist \
	&& (cd /rootfs && find . -print0 | cpio --null -ov --format=newc > /dist/initramfs.cpio) \
	&& gzip -1 /dist/initramfs.cpio


########################################################
# export stage
########################################################

FROM scratch AS export
COPY --from=cpio-builder /dist/initramfs.cpio.gz /initramfs.cpio.gz
