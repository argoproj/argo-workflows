#!/bin/sh
# Retry idempotent package-download commands, not builds or tests, up to 3 times.
if [ "$#" -eq 0 ]; then
    echo "Usage: sh hack/retry-download.sh COMMAND [ARG ...]" >&2
    exit 2
fi

attempt=1
until "$@"; do
    status=$?
    [ "$attempt" -ge 3 ] && exit "$status"
    delay=$((attempt * 5))
    echo "Package download attempt $attempt/3 failed (exit $status); retrying in ${delay}s." >&2
    sleep "$delay" || exit $?
    attempt=$((attempt + 1))
done
