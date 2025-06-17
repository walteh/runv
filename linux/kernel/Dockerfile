########################################################
# args
########################################################

ARG LINUX_KERNEL_VERSION=6.15-rc7

########################################################
# builder
########################################################

FROM debian:bookworm-slim AS builder

RUN apt-get update && apt-get install -y \
	git \
	make \
	gcc \
	flex \
	bison \
	libssl-dev \
	libelf-dev \
	bc \
	kmod \
	cpio \
	&& rm -rf /var/lib/apt/lists/*

ARG LINUX_KERNEL_VERSION

ENV SRC_DIR=/usr/src \
	DIST_DIR=/dist \
	LINUX_DIR=/usr/src/linux \
	LINUX_REPO_URL=git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git

RUN mkdir -p ${SRC_DIR} ${DIST_DIR} \
	&& git clone --depth 1 --branch v${LINUX_KERNEL_VERSION} ${LINUX_REPO_URL} ${LINUX_DIR}

WORKDIR ${LINUX_DIR}

ARG TARGETARCH

COPY base.config ./
COPY ${TARGETARCH}.fragment.config ./

RUN scripts/kconfig/merge_config.sh -m base.config ${TARGETARCH}.fragment.config

RUN make LOCALVERSION= olddefconfig

RUN <<EOF
if [ "${TARGETARCH}" = "arm64" ]; then
	make LOCALVERSION= -j$(nproc) Image
else
	make LOCALVERSION= -j$(nproc) bzImage
fi
EOF

RUN <<EOF
if [ "${TARGETARCH}" = "arm64" ]; then
	cp arch/arm64/boot/Image /boot/kernel
	cp .config /boot/config
else
	cp arch/x86/boot/bzImage /boot/kernel
	cp .config /boot/config
fi
EOF

########################################################
# export stage
########################################################

FROM scratch AS export
COPY --from=builder /boot /
