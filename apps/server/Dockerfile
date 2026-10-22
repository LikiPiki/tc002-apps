# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
COPY apps/server/ ./apps/server/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Version=${VERSION} -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Commit=${COMMIT}" \
    -o /out/tc002-server ./apps/server/cmd/server

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/tc002-server /usr/local/bin/tc002-server
WORKDIR /app
USER app
ENTRYPOINT ["/usr/local/bin/tc002-server"]

FROM build AS integration-build
RUN CGO_ENABLED=0 go test -c -o /out/server-check ./apps/server/internal/weatherprovider

FROM runtime AS integration
COPY --from=integration-build /out/server-check /usr/local/bin/server-check
ENTRYPOINT ["/usr/local/bin/server-check", "-test.v", "-test.run=TestWeatherClockIntegration"]

FROM runtime AS app
