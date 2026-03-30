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
#
# This runs inside the Docker builder image too, so it must only use tools the
# alpine build stage has: bash, busybox awk/grep/mktemp. No perl.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

FILE="$REPO_ROOT/vendor/google.golang.org/protobuf/internal/impl/api_export.go"
PANIC='panic(fmt.Sprintf("message %T is neither a v1 or v2 Message", m))'
PATCHED_MARKER='patched by hack/vendor-patches.sh: return nil instead of panicking'

if ! grep -qF "$PATCHED_MARKER" "$FILE"; then
  # Drift guard: expect the panic exactly twice — once in ProtoMessageV1Of
  # (which must keep it) and once in protoMessageV2Of (which we patch).
  count=$(grep -cF "$PANIC" "$FILE" || true)
  if [ "$count" != 2 ]; then
    echo "vendor-patches: expected the panic exactly twice in $FILE, found $count — has google.golang.org/protobuf changed? Update this script." >&2
    exit 1
  fi

  TMP="$(mktemp "$FILE.XXXXXX")"
  trap 'rm -f "$TMP"' EXIT

  awk -v panic="$PANIC" -v marker="$PATCHED_MARKER" '
    index($0, "func (Export) protoMessageV2Of(") { in_fn = 1 }
    in_fn && !done && index($0, panic) {
      print "\t\t// " marker
      print "\t\treturn nil"
      done = 1
      next
    }
    { print }
    END { exit done ? 0 : 1 }
  ' "$FILE" > "$TMP" || {
    echo "vendor-patches: protoMessageV2Of panic not found after its function declaration in $FILE — has google.golang.org/protobuf changed? Update this script." >&2
    exit 1
  }

  # ProtoMessageV1Of must keep its panic (and its use of the fmt import).
  if ! grep -qF "$PANIC" "$TMP"; then
    echo "vendor-patches: expected ProtoMessageV1Of to keep its panic in $FILE — refusing to patch. Update this script." >&2
    exit 1
  fi

  mv "$TMP" "$FILE"
  trap - EXIT
fi

# Kubernetes mis-tags PodLogOptions.Stream (a *string) with the varint wire
# type in its protobuf struct tag; the generated k8s marshaller correctly emits
# it as a length-delimited string (wire type 2), so the tag is metadata-only —
# but google.golang.org/protobuf's aberrant descriptor derivation (used for the
# gateway's query-parameter population) trusts the tag and panics with
# "invalid Go type string for field k8s_io.api.core.v1.PodLogOptions.stream"
# on every ?podLogOptions.*= log request. Fix the tag to match reality.
# (k8s has a few more type-vs-tag mismatches, e.g. *int32 fields tagged
# "bytes", but none are reachable via query population and the marshal paths
# use the generated gogo fast paths, so only this one needs patching.)
FILE2="$REPO_ROOT/vendor/k8s.io/api/core/v1/types.go"
BAD_TAG='Stream *string `json:"stream,omitempty" protobuf:"varint,10,opt,name=stream"`'
GOOD_TAG='Stream *string `json:"stream,omitempty" protobuf:"bytes,10,opt,name=stream"`'

if ! grep -qF "$GOOD_TAG" "$FILE2"; then
  count=$(grep -cF "$BAD_TAG" "$FILE2" || true)
  if [ "$count" != 1 ]; then
    echo "vendor-patches: expected the mis-tagged PodLogOptions.Stream exactly once in $FILE2, found $count — has k8s.io/api changed (upstream fix?)? Update this script." >&2
    exit 1
  fi
  TMP2="$(mktemp "$FILE2.XXXXXX")"
  trap 'rm -f "$TMP2"' EXIT
  awk -v bad="$BAD_TAG" -v good="$GOOD_TAG" '
    n = index($0, bad) {
      print substr($0, 1, n-1) good substr($0, n+length(bad))
      next
    }
    { print }
  ' "$FILE2" > "$TMP2"
  grep -qF "$GOOD_TAG" "$TMP2" || { echo "vendor-patches: PodLogOptions.Stream tag fix did not apply in $FILE2" >&2; exit 1; }
  mv "$TMP2" "$FILE2"
  trap - EXIT
fi
