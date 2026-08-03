#!/usr/bin/env bash
set -euo pipefail

# Measures semaphore promotion cost before and after the grant-set change, by
# running the same benchmark harness against two real revisions -- no emulation of
# the old implementation.
#
# Usage:
#
#   ./hack/semaphore-before-after.sh                    # BEFORE_REF..HEAD
#   ./hack/semaphore-before-after.sh <before> <after>   # explicit revisions
#
# How it works: the harness (workflow/sync/semaphore_bench_test.go) only touches
# the semaphore's exported-to-package surface -- newInternalSemaphore, addToQueue,
# tryAcquire, release -- which is unchanged across the revisions being compared. So
# the same file compiles against either tree. The script creates a git worktree per
# revision, copies today's harness in, runs it, and joins the RESULT lines into a
# markdown table.
#
# Because both sides run byte-identical harness code, any difference in the numbers
# is a difference in semaphore.go and nothing else.

BEFORE_REF="${1:-}"
AFTER_REF="${2:-HEAD}"

# Default "before" is the merge-base with the upstream default branch, i.e. the
# revision this work branched from. Overridable since that is a guess about intent.
if [[ -z "$BEFORE_REF" ]]; then
  BEFORE_REF="$(git merge-base HEAD origin/main 2>/dev/null || true)"
  if [[ -z "$BEFORE_REF" ]]; then
    echo "error: could not determine a default 'before' revision (no origin/main?)." >&2
    echo "       pass one explicitly: $0 <before-ref> [after-ref]" >&2
    exit 1
  fi
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
HARNESS="workflow/sync/semaphore_bench_test.go"
TEST_NAME="TestSemaphorePromotionCost"

if [[ ! -f "$REPO_ROOT/$HARNESS" ]]; then
  echo "error: $HARNESS not found in the working tree." >&2
  exit 1
fi

# Resolve to full SHAs so the labels are unambiguous and so a moving ref cannot
# change meaning between the two runs.
before_sha="$(git rev-parse --verify "$BEFORE_REF^{commit}")"
after_sha="$(git rev-parse --verify "$AFTER_REF^{commit}")"

if [[ "$before_sha" == "$after_sha" ]]; then
  echo "error: before and after resolve to the same commit ($before_sha)." >&2
  exit 1
fi

WORKDIR="$(mktemp -d)"
cleanup() {
  # Remove worktrees before the directory holding them, or git keeps stale entries.
  for wt in "$WORKDIR"/before "$WORKDIR"/after; do
    [[ -d "$wt" ]] && git -C "$REPO_ROOT" worktree remove --force "$wt" >/dev/null 2>&1 || true
  done
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

# run_at <label> <sha> -> writes RESULT lines to $WORKDIR/<label>.txt
run_at() {
  local label="$1"
  local sha="$2"
  local wt="$WORKDIR/$label"

  echo "==> $label: $(git -C "$REPO_ROOT" log --oneline -1 "$sha")" >&2

  # A detached worktree leaves the user's checkout, index, and stash untouched.
  git -C "$REPO_ROOT" worktree add --detach --quiet "$wt" "$sha"

  # Today's harness, so both sides measure with identical instrumentation. This is
  # the point of the script: only semaphore.go differs between the runs.
  cp "$REPO_ROOT/$HARNESS" "$wt/$HARNESS"

  if ! (cd "$wt" && go test ./workflow/sync/ -run "$TEST_NAME" -count=1 -v \
        >"$WORKDIR/$label.log" 2>&1); then
    echo "error: harness failed to build or run at $label ($sha). Last 40 lines:" >&2
    tail -40 "$WORKDIR/$label.log" >&2
    exit 1
  fi

  grep -o 'RESULT .*' "$WORKDIR/$label.log" >"$WORKDIR/$label.txt" || true
  if [[ ! -s "$WORKDIR/$label.txt" ]]; then
    echo "error: no RESULT lines from $label ($sha) -- harness ran but reported nothing." >&2
    tail -40 "$WORKDIR/$label.log" >&2
    exit 1
  fi
}

run_at before "$before_sha"
run_at after "$after_sha"

echo >&2

# Join on scenario name and format. Kept in awk rather than shell arithmetic
# because the reduction factor needs floating point.
awk '
function commas(n, s, out, i, len) {
  s = sprintf("%d", n); len = length(s); out = ""
  for (i = 1; i <= len; i++) {
    out = out substr(s, i, 1)
    if ((len - i) % 3 == 0 && i < len) out = out ","
  }
  return out
}
function cell(rec, wf) {
  return sprintf("%s (%.1f/wf)", commas(rec), rec / wf)
}
# RESULT scenario=X limit=N workflows=N reconciles=N rounds=N
# Fills the global kv[]. Trailing params are awks local-variable idiom; kv is
# deliberately NOT among them, since the caller reads it.
function parse(line,   i, n, parts, p) {
  delete kv
  n = split(line, parts, " ")
  for (i = 2; i <= n; i++) {
    split(parts[i], p, "=")
    kv[p[1]] = p[2]
  }
}
FNR == NR {
  parse($0)
  order[++count] = kv["scenario"]
  limit[kv["scenario"]]  = kv["limit"]
  wfs[kv["scenario"]]    = kv["workflows"]
  before[kv["scenario"]] = kv["reconciles"]
  next
}
{
  parse($0)
  after[kv["scenario"]] = kv["reconciles"]
}
END {
  print "| limit | workflows | before | after | reduction |"
  print "|------:|----------:|-------:|------:|----------:|"
  for (i = 1; i <= count; i++) {
    s = order[i]
    if (!(s in after)) {
      printf("| %s | %s | %s | (missing) | - |\n", limit[s], wfs[s], cell(before[s], wfs[s]))
      continue
    }
    printf("| %s | %s | %s | %s | %.0fx |\n",
           limit[s], wfs[s], cell(before[s], wfs[s]), cell(after[s], wfs[s]),
           after[s] > 0 ? before[s] / after[s] : 0)
  }
}
' "$WORKDIR/before.txt" "$WORKDIR/after.txt"

echo
echo "before: $before_sha"
echo "after:  $after_sha"
echo "reconciles = total tryAcquire calls to drain the queue; /wf divides by the"
echo "workflow count, where 1.0/wf is the floor (each workflow admitted first look)."
