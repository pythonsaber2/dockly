# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/dockly ./cmd/dockly

FROM alpine:3.24
RUN apk add --no-cache ca-certificates git docker-cli && addgroup -S dockly && adduser -S -G dockly dockly
COPY --from=build /out/dockly /usr/local/bin/dockly
RUN mkdir -p /var/lib/dockly && chown -R dockly:dockly /var/lib/dockly
VOLUME ["/var/lib/dockly"]
EXPOSE 8080
# Docker socket access normally requires root inside the container. Operators using
# a socket proxy can run this image as the unprivileged dockly user instead.
ENTRYPOINT ["dockly"]
