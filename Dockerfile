ARG GO_VERSION=1.24
ARG VERSION
ARG COMMIT
ARG REPO

FROM golang:${GO_VERSION} AS build

WORKDIR /go/src/${REPO}
COPY . ./
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		libsqlite3-dev \
	&& go build -v --tags "libsqlite3 linux" .


FROM debian:bookworm

LABEL VERSION=${VERSION} \
	MAINTAINER="K.C. Wong <kcwong@adveca.com>" \
	COMMIT=${COMMIT}

RUN apt-get --allow-insecure-repositories update \
	&& apt-get install -y --no-install-recommends \
		sqlite3 \
	&& mkdir -p /srv/trip-accountant/bin \
	&& mkdir /srv/trip-accountant/data

EXPOSE 8081
ENV GIN_MODE=release

COPY --from=build /go/src/${REPO}/trip-accountant /srv/trip-accountant/bin/
COPY --from=build /go/src/${REPO}/entrypoint.sh /srv/trip-accountant/bin

WORKDIR /srv/trip-accountant
ENTRYPOINT ["/srv/trip-accountant/bin/entrypoint.sh"]
CMD ["/srv/trip-accountant/bin/trip-accountant"]
