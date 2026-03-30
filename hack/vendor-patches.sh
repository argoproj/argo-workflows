#!/usr/bin/env bash
# Apply patches to vendored dependencies after `go mod vendor`.
#
# Kubernetes v1.35 (k8s.io/* v0.35) moved ProtoMessage() on generated types
# behind the kubernetes_protomessage_one_more_release build tag; k8s v1.36
# removes the method entirely. Without ProtoMessage(), these types no longer
# satisfy protoiface.MessageV1, so protoMessageV2Of() panics instead of
# returning nil. Returning nil lets the caller fall through to
# LegacyLoadMessageDesc / aberrantLoadMessageDesc, which builds descriptors from
# struct tags — the designed fallback for non-standard proto types.
#
# Only protoMessageV2Of is patched. The identical panic string also appears in
# the exported ProtoMessageV1Of, which must keep panicking loudly, so the
# substitution below is anchored to the protoMessageV2Of function body. The
# script is idempotent (an already-patched tree is a no-op) and never leaves the
# file half-rewritten: the substitution happens on a copy that only replaces the
# original once it is verified.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

FILE="$REPO_ROOT/vendor/google.golang.org/protobuf/internal/impl/api_export.go"
PANIC='panic(fmt.Sprintf("message %T is neither a v1 or v2 Message", m))'
PATCHED_MARKER='patched by hack/vendor-patches.sh: return nil instead of panicking'

if grep -qF "$PATCHED_MARKER" "$FILE"; then
  exit 0
fi

TMP="$(mktemp "$FILE.XXXXXX")"
trap 'rm -f "$TMP"' EXIT
cp "$FILE" "$TMP"

count=$(perl -0777 -i -pe '
  our $count = s/(func \(Export\) protoMessageV2Of\(.*?)panic\(fmt\.Sprintf\("message %T is neither a v1 or v2 Message", m\)\)/${1}\/\/ patched by hack\/vendor-patches.sh: return nil instead of panicking\n\t\treturn nil/s;
  END { print STDERR $count }
' "$TMP" 2>&1 >/dev/null) || true

case "$count" in
  1) ;;
  0)
    echo "vendor-patches: protoMessageV2Of panic not found in $FILE — has google.golang.org/protobuf changed? Update this script." >&2
    exit 1
    ;;
  *)
    echo "vendor-patches: substitution matched $count times in $FILE (expected exactly 1) — refusing to patch. Update this script." >&2
    exit 1
    ;;
esac

# ProtoMessageV1Of must keep its panic (and its use of the fmt import).
if ! grep -qF "$PANIC" "$TMP"; then
  echo "vendor-patches: expected ProtoMessageV1Of to keep its panic in $FILE — refusing to patch. Update this script." >&2
  exit 1
fi

mv "$TMP" "$FILE"
trap - EXIT
