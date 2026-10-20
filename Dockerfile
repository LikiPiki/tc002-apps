# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
ARG APP=worker
WORKDIR /src
COPY go.mod go.sum ./
COPY apps/server/ ./apps/server/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./apps/server/cmd/${APP}

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/app /usr/local/bin/app
USER app
ENTRYPOINT ["/usr/local/bin/app"]
