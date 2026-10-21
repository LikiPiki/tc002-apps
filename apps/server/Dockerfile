# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
ARG APP=worker
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
COPY apps/server/ ./apps/server/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./apps/server/cmd/${APP}

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/app /usr/local/bin/app
WORKDIR /app
COPY configs/config.toml ./configs/config.toml
USER app
ENTRYPOINT ["/usr/local/bin/app"]

FROM build AS weather-integration-build
RUN CGO_ENABLED=0 go test -c -o /out/weather-check ./apps/server/internal/weatherprovider

FROM runtime AS weather-integration
COPY --from=weather-integration-build /out/weather-check /usr/local/bin/weather-check
ENTRYPOINT ["/usr/local/bin/weather-check", "-test.v", "-test.run=TestWeatherClockIntegration"]

FROM runtime AS app
