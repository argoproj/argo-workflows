#!/bin/sh
# Retry idempotent package-download commands, not builds or tests. Keep output
# streaming and preserve the command's exit status without requiring Bash.
set -u

if [ "$#" -eq 0 ]; then
    echo "Usage: sh hack/retry-download.sh COMMAND [ARG ...]" >&2
    exit 2
fi

workdir=$(mktemp -d "${TMPDIR:-/tmp}/argo-download.XXXXXX") || exit 1
trap 'rm -rf "$workdir"' 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

# Match download diagnostics, not arbitrary numbers in package names or versions.
http_errors='(^|[^0-9])(408 Request Timeout|429 Too Many Requests|500 Internal Server Error|502 Bad Gateway|503 Service Unavailable|504 Gateway Timeout)([^[:alpha:]]|$)'
http_codes='HTTP(/[0-9.]+)?[[:space:]]+(408|429|500|502|503|504)([[:space:]]|$)|returned error: (408|429|500|502|503|504)([^0-9]|$)|code E(408|429|500|502|503|504)([^0-9]|$)'
network_errors='ECONNRESET|ETIMEDOUT|EAI_AGAIN|ECONNREFUSED|ENETUNREACH|EHOSTUNREACH|socket hang up|TLS handshake timeout|i/o timeout|connection reset by peer|temporary failure in name resolution|stream error: stream ID [0-9]+; INTERNAL_ERROR; received from peer'
permanent_http_errors='(^|[^0-9])(401 Unauthorized|403 Forbidden|404 Not Found|501 Not Implemented)([^[:alpha:]]|$)'
permanent_http_codes='HTTP(/[0-9.]+)?[[:space:]]+(401|403|404|501)([[:space:]]|$)|returned error: (401|403|404|501)([^0-9]|$)|code E(401|403|404|501)([^0-9]|$)'
permanent_errors="unable to verify the first certificate|integrity check failed|checksum mismatch|couldn't find any versions for package|syntax error|command not found|inconsistent vendoring"

attempt=1
while :; do
    # POSIX pipelines expose only tee's status. Save the download's status in a
    # private file; truncating it also prevents reusing a prior attempt's status.
    : > "$workdir/status"
    {
        "$@"
        printf '%s\n' "$?" > "$workdir/status"
    } 2>&1 | tee "$workdir/output"
    stream_status=$?
    if [ "$stream_status" -ne 0 ]; then
        exit "$stream_status"
    fi
    IFS= read -r status < "$workdir/status" || exit 1
    if [ "$status" -eq 0 ]; then
        exit 0
    fi
    # A transient diagnostic must not override a permanent failure in the same attempt.
    if grep -Eiq "$permanent_http_errors|$permanent_http_codes|$permanent_errors" "$workdir/output"; then
        exit "$status"
    fi
    if [ "$status" -ge 128 ] || [ "$attempt" -ge 3 ] || ! grep -Eiq "$http_errors|$http_codes|$network_errors" "$workdir/output"; then
        exit "$status"
    fi

    delay=$((attempt * 5))
    printf 'Package download attempt %s/3 failed (exit %s); retrying in %ss.\n' "$attempt" "$status" "$delay" >&2
    sleep "$delay" || exit $?
    attempt=$((attempt + 1))
done
