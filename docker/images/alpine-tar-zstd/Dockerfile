FROM alpine:3.16.2

RUN apk update \
    && apk add --no-cache pigz tar zstd \
    && rm -rf /var/cache/apk/*
