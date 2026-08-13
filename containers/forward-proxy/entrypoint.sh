#!/bin/sh
set -eu

: "${SQUID_ALLOWED_DEMO_HOST_AND_PORT:?SQUID_ALLOWED_DEMO_HOST_AND_PORT must be set}"

case "${SQUID_ALLOWED_DEMO_HOST_AND_PORT}" in
    *:*)
        SQUID_ALLOWED_DEMO_HOST=${SQUID_ALLOWED_DEMO_HOST_AND_PORT%:*}
        SQUID_ALLOWED_DEMO_PORT=${SQUID_ALLOWED_DEMO_HOST_AND_PORT##*:}
        ;;
    *)
        echo "SQUID_ALLOWED_DEMO_HOST_AND_PORT must be in host:port format" >&2
        exit 1
        ;;
esac

if [ -z "${SQUID_ALLOWED_DEMO_HOST}" ]; then
    echo "SQUID_ALLOWED_DEMO_HOST_AND_PORT must include a host" >&2
    exit 1
fi

case "${SQUID_ALLOWED_DEMO_PORT}" in
    *[!0-9]* | '')
        echo "SQUID_ALLOWED_DEMO_HOST_AND_PORT must include a numeric port" >&2
        exit 1
        ;;
esac

export SQUID_ALLOWED_DEMO_HOST SQUID_ALLOWED_DEMO_PORT

perl -pe '
    s/\$\{SQUID_ALLOWED_DEMO_HOST\}/$ENV{SQUID_ALLOWED_DEMO_HOST}/g;
    s/\$\{SQUID_ALLOWED_DEMO_PORT\}/$ENV{SQUID_ALLOWED_DEMO_PORT}/g;
' \
    < /etc/squid/squid.conf.template \
    > /tmp/squid.conf

exec /usr/local/bin/entrypoint.sh -f /tmp/squid.conf "$@"
