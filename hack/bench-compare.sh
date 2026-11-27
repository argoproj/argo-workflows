#!/usr/bin/env bash
# Compare the Go benchmarks of a base commit with the checked-out tree.
#
# Usage: hack/bench-compare.sh BASE_REF OUT_DIR
#
# Every package outside vendor/ and test/e2e/ that declares a Benchmark
# function in the checked-out tree is built twice: once from BASE_REF, checked
# out into a temporary worktree, and once from the checked-out tree. The two
# builds run alternately, BENCH_COUNT rounds each, so any drift in the machine
# while they run lands on both sides. benchstat then compares them.
#
# A change only counts when benchstat finds it significant and it is worse
# than the threshold for its unit:
#   BENCH_FAIL_THRESHOLDS    regressions to flag (OUT_DIR/status is "regressed")
#   BENCH_REPORT_THRESHOLDS  regressions to list without flagging
# Both are space-separated unit=percent pairs. A unit in neither list is only
# shown in the full comparison.
#
# Results in OUT_DIR: base.txt and head.txt (raw go test output),
# comparison.txt (benchstat), regressions.md, report.md, summary.md, status.
# The script exits non-zero only when it fails to build or run something.
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "Usage: $0 BASE_REF OUT_DIR" >&2
	exit 2
fi
base_ref=$1
out=$2

BENCH_COUNT=${BENCH_COUNT:-6}
BENCH_TIME=${BENCH_TIME:-0.5s}
BENCH_FAIL_THRESHOLDS=${BENCH_FAIL_THRESHOLDS:-"allocs/op=5 B/op=10"}
BENCH_REPORT_THRESHOLDS=${BENCH_REPORT_THRESHOLDS:-"sec/op=20"}
# Pinned like the Makefile's tools; go run caches the build.
BENCHSTAT=${BENCHSTAT:-"go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e"}

# Both trees build in module mode (vendor/ is not checked in, so a local one
# may be stale) and see no cluster.
export GOFLAGS="${GOFLAGS:-} -mod=mod"
export KUBECONFIG=/dev/null

head_src=$(git rev-parse --show-toplevel)
base_sha=$(git rev-parse --verify "$base_ref^{commit}")
head_sha=$(git rev-parse HEAD)
head_label=${head_sha:0:12}
if ! git diff --quiet HEAD; then
	head_label+=" + uncommitted changes"
fi

rm -rf "$out"
mkdir -p "$out/bin/base" "$out/bin/head"
out=$(cd "$out" && pwd)
base_src="$out/base-src"

cleanup() {
	git -C "$head_src" worktree remove --force "$base_src" 2>/dev/null || true
}
trap cleanup EXIT
git -C "$head_src" worktree add --quiet --detach "$base_src" "$base_sha"

# Packages with benchmarks in head. One that only gains benchmarks in head is
# still built from base when it exists there, so benchstat shows head alone.
mapfile -t pkgs < <(
	git -C "$head_src" grep -l -E '^func Benchmark' -- '*_test.go' ':!:vendor/' ':!:test/e2e/' |
		xargs -n1 dirname | sort -u
)
if [ "${#pkgs[@]}" -eq 0 ]; then
	echo "No benchmarks found" >&2
	exit 1
fi

bin_name() { echo "${1//\//_}.test"; }

for pkg in "${pkgs[@]}"; do
	bin=$(bin_name "$pkg")
	echo "Building $pkg"
	(cd "$head_src" && go test -c -o "$out/bin/head/$bin" "./$pkg")
	if [ -d "$base_src/$pkg" ]; then
		(cd "$base_src" && go test -c -o "$out/bin/base/$bin" "./$pkg") ||
			echo "warning: $pkg does not build at $base_sha, comparing head only" >&2
	fi
done

# Runs one round of a side's benchmarks, appending to OUT_DIR/SIDE.txt.
# Test binaries run in their package directory, as go test runs them.
run_side() {
	local side=$1 src=$2 pkg bin
	for pkg in "${pkgs[@]}"; do
		bin="$out/bin/$side/$(bin_name "$pkg")"
		[ -x "$bin" ] || continue
		(cd "$src/$pkg" && "$bin" -test.run='^$' -test.bench=. -test.benchmem \
			-test.benchtime="$BENCH_TIME" -test.count=1) >>"$out/$side.txt"
	done
}

: >"$out/base.txt"
: >"$out/head.txt"
for ((round = 1; round <= BENCH_COUNT; round++)); do
	echo "Round $round/$BENCH_COUNT"
	# Swap the order every round so neither side always runs first.
	if ((round % 2)); then
		run_side base "$base_src"
		run_side head "$head_src"
	else
		run_side head "$head_src"
		run_side base "$base_src"
	fi
done

(cd "$out" && $BENCHSTAT base=base.txt head=head.txt) >"$out/comparison.txt"
(cd "$out" && $BENCHSTAT -format csv base=base.txt head=head.txt) 2>/dev/null >"$out/comparison.csv"

# Lists the significant regressions over the thresholds as markdown table rows.
# benchstat's CSV has a ",base,,head,,," row then a ",UNIT,CI,UNIT,CI,vs base,P"
# row before each unit's table, and "pkg: ..." lines between packages. Its
# "vs base" column is "~" unless the change is significant. Every unit it
# reports (sec/op, B/op, allocs/op) is worse when it goes up.
regressions() {
	awk -F, -v thresholds="$1" '
		BEGIN {
			n = split(thresholds, pairs, " ")
			for (i = 1; i <= n; i++) {
				split(pairs[i], kv, "=")
				limit[kv[1]] = kv[2] + 0
			}
		}
		/^pkg: / { pkg = substr($0, 6); sub(/^github.com\/argoproj\/argo-workflows\/v4\//, "", pkg); next }
		$1 == "" && $3 == "CI" { unit = $2; next }
		$1 == "" || $1 == "geomean" || NF < 7 { next }
		$6 ~ /^\+[0-9.]+%$/ && (unit in limit) {
			pct = substr($6, 2); sub(/%$/, "", pct)
			if (pct + 0 > limit[unit]) printf "| `%s` | `%s` | %s | %s | %s |\n", pkg, $1, unit, $6, $7
		}
	' "$out/comparison.csv"
}

table_header='| Package | Benchmark | Unit | Change | Significance |
|---|---|---|---|---|'
regressions "$BENCH_FAIL_THRESHOLDS" >"$out/regressions.md"
regressions "$BENCH_REPORT_THRESHOLDS" >"$out/report.md"

if [ -s "$out/regressions.md" ]; then
	echo regressed >"$out/status"
else
	echo pass >"$out/status"
fi

{
	echo "## Benchmarks: \`${base_sha:0:12}\` → \`$head_label\`"
	echo
	echo "$BENCH_COUNT rounds of \`-benchtime=$BENCH_TIME\` per side, run alternately."
	echo
	if [ -s "$out/regressions.md" ]; then
		echo "### Regressions (over $BENCH_FAIL_THRESHOLDS)"
		echo
		echo "$table_header"
		cat "$out/regressions.md"
	else
		echo "No regressions over $BENCH_FAIL_THRESHOLDS."
	fi
	echo
	if [ -s "$out/report.md" ]; then
		echo "### Report only (over $BENCH_REPORT_THRESHOLDS)"
		echo
		echo "$table_header"
		cat "$out/report.md"
		echo
	fi
	echo "<details><summary>Full benchstat comparison</summary>"
	echo
	echo '```'
	cat "$out/comparison.txt"
	echo '```'
	echo
	echo "</details>"
} >"$out/summary.md"

cat "$out/comparison.txt"
echo
echo "Status: $(cat "$out/status") (see $out/summary.md)"
