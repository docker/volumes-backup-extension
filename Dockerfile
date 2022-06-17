FROM --platform=$BUILDPLATFORM node:17.7-alpine3.14 AS client-builder
WORKDIR /ui
# cache packages in layer
COPY ui/package.json /ui/package.json
COPY ui/package-lock.json /ui/package-lock.json
RUN --mount=type=cache,target=/usr/src/app/.npm \
    npm set cache /usr/src/app/.npm && \
    npm ci
# install
COPY ui /ui
RUN npm run build

FROM alpine:3.16
LABEL org.opencontainers.image.title="vackup-docker-extension" \
    org.opencontainers.image.description="Easily backup and restore docker volumes." \
    org.opencontainers.image.vendor="Felipe" \
    com.docker.desktop.extension.api.version=">= 0.2.3" \
    com.docker.extension.screenshots="" \
    com.docker.extension.detailed-description="" \
    com.docker.extension.publisher-url="https://github.com/felipecruz91/vackup-docker-extension" \
    com.docker.extension.additional-urls="[{\"title\":\"Author\", \"url\":\"https://twitter.com/felipecruz\"}]" \
    com.docker.extension.changelog=""

WORKDIR /
COPY metadata.json .
COPY docker.svg .
COPY --from=client-builder /ui/build ui

RUN mkdir -p /vackup
