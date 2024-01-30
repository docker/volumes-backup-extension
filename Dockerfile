FROM golang:1.21-alpine AS builder
ENV CGO_ENABLED=0
WORKDIR /backend
COPY vm/go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY vm/. .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o bin/service

FROM --platform=$BUILDPLATFORM node:17.7-alpine3.14@sha256:539e64749f7dc6c578d744d879fd0ec37f3afe552ae4aca9744cc85121728c4c AS client-builder
WORKDIR /ui
# cache packages in layer
COPY ui/package.json /ui/package.json
COPY ui/package-lock.json /ui/package-lock.json
RUN --mount=type=cache,target=/usr/src/app/.npm \
    npm set cache /usr/src/app/.npm && \
    npm ci
# install
COPY ui /ui
RUN --mount=type=secret,id=BUGSNAG_API_KEY \
    REACT_APP_BUGSNAG_API_KEY=$(cat /run/secrets/BUGSNAG_API_KEY) \
    npm run build
RUN --mount=type=secret,id=REACT_APP_MUI_LICENSE_KEY \
    REACT_APP_MUI_LICENSE_KEY=$(cat /run/secrets/REACT_APP_MUI_LICENSE_KEY) \
    yarn build

FROM alpine:3.16@sha256:bc41182d7ef5ffc53a40b044e725193bc10142a1243f395ee852a8d9730fc2ad as base
ARG CLI_VERSION=20.10.17
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
RUN apk update \
    && apk add --no-cache ca-certificates curl \
    && rm -rf /var/cache/apk/*
RUN curl -fL "https://download.docker.com/linux/static/stable/$(uname -m)/docker-${CLI_VERSION}.tgz" | tar zxf - --strip-components 1 docker/docker \
    && chmod +x /docker

FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS docker-credentials-client-builder
ENV CGO_ENABLED=0
WORKDIR /output
RUN apk update \
    && apk add --no-cache build-base=0.5-r3 \
    && rm -rf /var/cache/apk/*
COPY client .
RUN make cross

FROM busybox:1.35.0@sha256:d7f4aada301c0f13d93ceed62fef318c195c38bf430fc8bfbdf1d850514422ff

ARG BUGSNAG_RELEASE_STAGE="local"
ARG BUGSNAG_APP_VERSION="latest"

ENV BUGSNAG_RELEASE_STAGE=$BUGSNAG_RELEASE_STAGE
ENV BUGSNAG_APP_VERSION=$BUGSNAG_APP_VERSION

LABEL org.opencontainers.image.title="Volumes Backup & Share" \
    org.opencontainers.image.description="Backup, clone, restore, and share Docker volumes effortlessly." \
    org.opencontainers.image.vendor="Docker Inc." \
    com.docker.desktop.extension.api.version=">= 0.2.3" \
    com.docker.extension.screenshots="[ \
    {\"alt\": \"Home page - list of volumes\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/1-table.png\"}, \
    {\"alt\": \"Import data into a new volume\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/2-import-new.png\"}, \
    {\"alt\": \"Export volume dialog\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/3-export.png\"}, \
    {\"alt\": \"Transfer volume to another host\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/4-transfer.png\"}, \
    {\"alt\": \"Clone volume dialog\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/5-clone.png\"}, \
    {\"alt\": \"Delete volume dialog\", \"url\": \"https://raw.githubusercontent.com/docker/volumes-backup-extension/main/docs/images/6-delete.png\"} \
    ]" \
    com.docker.extension.detailed-description="<p>With Volumes Backup & Share you can easily create copies of your volumes and also share them with others through SSH or pushing them to a registry.</p> \
    <h2 id="-features">✨ What can you do with this extension?</h2> \
    <ul> \
    <li>Export a volume:</li> \
    <ul><li>To a compressed file in your local filesystem</li> \
    <li>To an existing local image</li> \
    <li>To a new local image</li> \
    <li>To a new image in Docker Hub (or another registry)</li></ul> \
    <li>Import data into a new container or into an existing container:</li> \
    <ul><li>From a compressed file in your local filesystem</li> \
    <li>From an existing image</li> \
    <li>From an existing image in Docker Hub (or another registry)</li></ul> \
    <li>Transfer a volume via SSH to another host that runs Docker Desktop or Docker engine.</li> \
    <li>Clone, empty or delete a volume</li> \
    </ul> \
    <h2>Acknowledgements</h2> \
    <ul> \
    <li><a href=\"https://github.com/BretFisher/docker-vackup\">Vackup project by Bret Fisher</a></li> \
    <li><a href=\"https://www.youtube.com/watch?v=BHKp7Sc3VVc\">Building Vackup - LiveStream on YouTube</a></li> \
    <ul> \
    "\
    com.docker.extension.publisher-url="https://www.docker.com/" \
    com.docker.extension.additional-urls="[ \
    {\"title\":\"Support\", \"url\":\"https://github.com/docker/volumes-backup-extension/issues\"} \
    ]" \
    com.docker.desktop.extension.icon="https://raw.githubusercontent.com/docker/volumes-backup-extension/main/icon.svg" \
    com.docker.extension.changelog="<ul>\
    <li>Fixed current image vulnerabilities (CVEs) using Docker Scout.</li> \
    </ul>" \
    com.docker.extension.categories="volumes"

WORKDIR /
COPY docker-compose.yaml .
COPY metadata.json .
COPY icon.svg .
COPY --from=base /etc/ssl/certs /etc/ssl/certs
COPY --from=base /docker /usr/local/bin/docker
COPY --from=builder /backend/bin/service /
COPY --from=client-builder /ui/build ui
COPY --from=docker-credentials-client-builder output/dist ./host

RUN mkdir -p /vackup

RUN --mount=type=secret,id=BUGSNAG_API_KEY \
    BUGSNAG_API_KEY=$(cat /run/secrets/BUGSNAG_API_KEY); \
    echo "$BUGSNAG_API_KEY" > /tmp/bugsnag-api-key.txt

ENTRYPOINT ["/bin/sh", "-c", "BUGSNAG_API_KEY=$(cat /tmp/bugsnag-api-key.txt); rm -rf /tmp/bugsnag-api-key.txt; BUGSNAG_API_KEY=$BUGSNAG_API_KEY /service -socket /run/guest-services/ext.sock"]
