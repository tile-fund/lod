# To build, run in root of LOD source tree:
#
#	$ git clone git@github.com:dechristopher/lod.git or git clone https://github.com/dechristopher/lod.git
#	$ cd lod
#	$ docker build -t lod .
#
# Example use with http-based config:
#
#	$ docker run -p 1337:1337 lod --conf http://my-domain.com/config
#
# Example use with local config:
#
#	$ mkdir lod-config
#	$ cp your_config.toml lod-config/config.toml
#	$ docker run -v /path/to/lod-config:/opt/lod_config -p 1337:1337 lod --conf /opt/lod_config/config.toml
#
# You can create your own Dockerfile that adds a `config.toml` from the context into the config directory, like so:
#
#   # FROM dechristopher/lod:0.8.0
#   # COPY /path/to/your_config.toml /opt/lod_cfg/config.toml
#   # CMD [ "/opt/lod", "--conf", "/opt/lod_cfg/config.toml" ]
#

# ---- Build Stage ----
FROM golang:1.20.4-alpine3.16 as builder

LABEL maintainer="Andrew DeChristopher"

ARG VERSION="DEVELOPMENT"
ENV VERSION="${VERSION}"

WORKDIR /build

# We need libgeos to suport geospatial operations when running tile intersections
# for some of the administrative endpoint functions
RUN apk update \
	&& apk add musl-dev \
	&& apk add gcc \
	&& apk add geos \
	&& apk add geos-dev

ENV USER=lod
ENV UID=10001

RUN adduser \
	--disabled-password \
	--gecos "" \
	--home "/nonexistent" \
	--shell "/sbin/nologin" \
	--no-create-home \
	--uid "${UID}" \
	"${USER}"

# Set up source for compilation
RUN mkdir -p /go/src/github.com/dechristopher/lod
COPY . /go/src/github.com/dechristopher/lod

# Build binary
RUN cd /go/src/github.com/dechristopher/lod/cmd/lod \
	&& go build -v -ldflags "-w -X 'github.com/dechristopher/lod/config.Version=${VERSION}'" -gcflags "-N -l" -o /opt/lod \
	&& chmod a+x /opt/lod

# ---- Run Stage ----
FROM alpine:3.21

LABEL maintainer="Andrew DeChristopher"

RUN apk update \
	&& apk add ca-certificates \
	&& apk add geos \
	&& apk add geos-dev \
	&& rm -rf /var/cache/apk/*

RUN mkdir /opt/lod_config

# Import the user and group files from the builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Use an unprivileged user
USER lod:lod

# Copy binary from builder
COPY --from=builder /opt/lod /opt/
WORKDIR /opt

ENTRYPOINT ["/opt/lod"]
