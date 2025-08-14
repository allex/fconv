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

# -------- Runtime stage --------
FROM debian:12-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install runtime deps: LibreOffice (headless capable), fonts, certs
RUN set -ex; \
	sed -i 's#deb.debian.org#mirrors.ustc.edu.cn#g' /etc/apt/sources.list.d/debian.sources \
	&& apt-get update \
	&& apt-get install -yq --no-install-recommends \
		libreoffice-writer libreoffice-calc \
		default-jre ure-java \
		fonts-dejavu \
	&& apt-get autoremove -yq \
	&& rm -rf /var/lib/apt/lists/* /usr/share/man/*

# Create non-root user
RUN groupadd -g 201 fconv \
  && useradd -m -u 201 -g fconv fconv

WORKDIR /app

# Copy binary
COPY --from=builder /out/fconv /usr/local/bin/fconv

# Default env
ENV GIN_MODE=release \
	FCONV_LISTEN_ADDR=:8080 \
	FCONV_ENABLE_SHA256=true

EXPOSE 8080
USER fconv

ENTRYPOINT ["fconv"]
