# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
ARG APP=clockctl
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
COPY apps/tools/ ./apps/tools/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Version=${VERSION} -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Commit=${COMMIT}" \
    -o /out/app ./apps/tools/cmd/${APP}

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/app /usr/local/bin/app
WORKDIR /app
COPY configs/config.toml ./configs/config.toml
USER app
ENTRYPOINT ["/usr/local/bin/app"]
