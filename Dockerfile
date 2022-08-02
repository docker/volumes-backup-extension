FROM golang:1.17-alpine AS builder
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

FROM golang:1.17-alpine AS volume-share-client-builder
WORKDIR /output
RUN apk add build-base
COPY client .
RUN make cross

FROM busybox:1.35.0
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
COPY docker-compose.yaml .
COPY metadata.json .
COPY docker.svg .
COPY --from=builder /backend/bin/service /
COPY --from=client-builder /ui/build ui
COPY --from=volume-share-client-builder output/dist ./host

RUN mkdir -p /vackup

CMD /service -socket /run/guest-services/ext.sock