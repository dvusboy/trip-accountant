ARG GO_VERSION=1.24
ARG REPO=dvusboy/trip-accountant
ARG PREFIX=/srv/trip-accountant
ARG VERSION=0.0.0
ARG COMMIT=unknown

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
	&& mkdir -p ${PREFIX}/bin \
	&& mkdir ${PREFIX}/data

EXPOSE 8081
ENV GIN_MODE=release

COPY --from=build /go/src/${REPO}/trip-accountant ${PREFIX}/bin/
COPY --from=build /go/src/${REPO}/entrypoint.sh ${PREFIX}/bin

WORKDIR ${PREFIX}
ENTRYPOINT ["${PREFIX}/bin/entrypoint.sh"]
CMD ["${PREFIX}/bin/trip-accountant"]
