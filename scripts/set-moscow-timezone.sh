#!/bin/sh
set -eu

: "${AWTRIX_URL:?Set AWTRIX_URL to the clock HTTP URL, e.g. http://192.168.1.50}"

# AWTRIX needs both the IANA display name and the POSIX rule. In POSIX TZ
# notation the offset sign is reversed: MSK-3 means UTC+3, without DST.
curl --fail --silent --show-error --connect-timeout 5 --max-time 15 \
    --request PUT "${AWTRIX_URL%/}/api/v1/system" \
    --header 'Content-Type: application/json' \
    --data '{"tzName":"Europe/Moscow","tz":"MSK-3"}' \
    --output /dev/null

printf '%s\n' "Clock timezone saved: Europe/Moscow (UTC+3) at $AWTRIX_URL"
