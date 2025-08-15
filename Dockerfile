# syntax=docker/dockerfile:1.6

# -------- Builder stage --------
ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# Use China GOPROXY mirror for faster module downloads
ENV GOPROXY=https://goproxy.cn,direct \
	GOSUMDB=sum.golang.google.cn

# Cache go modules first
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

# Copy source
COPY . .

# Build arguments for versioning
ARG BUILD_TAG=dev
ARG BUILD_TIME=""
ARG GIT_COMMIT=unknown

# Build binary
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 \
	go build -ldflags "-w -s -X main.appVersion=v${BUILD_TAG} -X main.gitCommit=${GIT_COMMIT} -X 'main.buildTime=${BUILD_TIME}'" -o /out/fconv .

# -------- Package downloader stage --------
FROM debian:12-slim AS package-downloader

# Configure mirrors and download all required packages and their dependencies
RUN set -ex; \
	sed -i 's#deb.debian.org#mirrors.ustc.edu.cn#g' /etc/apt/sources.list.d/debian.sources \
	&& apt-get update \
	&& mkdir -p /tmp/build && DEBIAN_FRONTEND=noninteractive apt-get -o Dir::Cache::Archives=/tmp/build --download-only install -yq --no-install-recommends \
		libreoffice-writer libreoffice-calc \
		default-jre ure-java \
		fonts-dejavu \
	&& cp /etc/apt/sources.list.d/debian.sources /tmp/build/debian.sources

# -------- Runtime stage --------
FROM debian:12-slim

ARG GIT_COMMIT=

ENV DEBIAN_FRONTEND=noninteractive \
	GIN_MODE=release \
	FCONV_PORT=8080 \
	FCONV_ENABLE_SHA256=true \
	GIT_COMMIT=${GIT_COMMIT}

# Install packages from mounted .deb files
RUN --mount=from=package-downloader,source=/tmp/build,target=/tmp/build \
	set -ex; \
	(cd /tmp/build && dpkg -i *.deb && cp -f ./debian.sources /etc/apt/sources.list.d/debian.sources) \
	&& rm -rf /var/lib/apt/lists/* /usr/share/man/*

# Create non-root user
RUN groupadd -g 201 fconv \
	&& useradd -m -u 201 -g fconv fconv

WORKDIR /app

# Copy binary
COPY --from=builder /out/fconv /usr/local/bin/fconv

EXPOSE 8080
USER fconv

ENTRYPOINT ["fconv"]
