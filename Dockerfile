FROM golang:alpine as builder
ENV USER=proxy APPNAME=proxy USER_ID=1000

RUN apk add make && adduser -D -H -u ${USER_ID} ${USER}

ADD go.mod /build/
RUN cd /build && go mod download

ARG VERSION=0.0.1

ADD . /build/
RUN cd /build && VERSION=${VERSION} BINARY=${APPNAME} make build

FROM scratch
ENV USER=proxy APPNAME=proxy APPDIR=/app
COPY --from=builder /build/${APPNAME} ${APPDIR}/
COPY --from=builder /etc/passwd /etc/passwd
WORKDIR ${APPDIR}
USER ${USER}
CMD ["/bin/sh","-c","/app/proxy"]
