package downloads

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type downloadHarness struct {
	dir, scratch, script string
	env                  []string
}

type outcome struct {
	message string
	status  int
}

// newDownloadHarness supplies isolated temporary files and a recording sleep
// command so the real POSIX retry script can be exercised without waiting.
func newDownloadHarness(t *testing.T) *downloadHarness {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the download helper runs in POSIX build containers and Linux CI")
	}
	dir := t.TempDir()
	bin, scratch := filepath.Join(dir, "bin"), filepath.Join(dir, "scratch")
	require.NoError(t, os.Mkdir(bin, 0o755))
	require.NoError(t, os.Mkdir(scratch, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "sleep"), []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$SLEEP_LOG\"\n"), 0o755))
	script, err := filepath.Abs("../retry-download.sh")
	require.NoError(t, err)
	return &downloadHarness{
		dir: dir, scratch: scratch, script: script,
		env: append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"TMPDIR="+scratch, "SLEEP_LOG="+filepath.Join(dir, "sleeps"), "COUNT_FILE="+filepath.Join(dir, "count")),
	}
}

// run executes the actual shell helper, preserving diagnostics and checking that
// its private attempt logs are removed on both success and failure.
func (h *downloadHarness) run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", append([]string{h.script}, args...)...)
	cmd.Dir, cmd.Env = h.dir, h.env
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), string(output))
	status := 0
	if err != nil {
		var exitError *exec.ExitError
		require.ErrorAs(t, err, &exitError, string(output))
		status = exitError.ExitCode()
	}
	files, readErr := os.ReadDir(h.scratch)
	require.NoError(t, readErr)
	for _, file := range files {
		assert.False(t, strings.HasPrefix(file.Name(), "argo-download."), "temporary download logs must be cleaned up")
	}
	return string(output), status
}

// sleeps returns the retry delays recorded in place of real sleeps.
func (h *downloadHarness) sleeps(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dir, "sleeps"))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	return strings.Fields(string(data))
}

// fixture makes a command that emits known package-manager diagnostics and exit
// codes on successive invocations, while recording its arguments and attempts.
func (h *downloadHarness) fixture(t *testing.T, outcomes ...outcome) string {
	t.Helper()
	var script strings.Builder
	script.WriteString("#!/bin/sh\ncount=0\nif [ -f \"$COUNT_FILE\" ]; then read -r count < \"$COUNT_FILE\"; fi\ncount=$((count + 1))\nprintf '%s\\n' \"$count\" > \"$COUNT_FILE\"\nprintf 'stdout attempt %s\\n' \"$count\"\nprintf 'argument=<%s>\\n' \"$@\"\ncase $count in\n")
	for i, result := range outcomes {
		h.env = append(h.env, fmt.Sprintf("MESSAGE_%d=%s", i+1, result.message), fmt.Sprintf("STATUS_%d=%d", i+1, result.status))
		fmt.Fprintf(&script, "%d) printf '%%s\\n' \"$MESSAGE_%d\" >&2; exit \"$STATUS_%d\";;\n", i+1, i+1, i+1)
	}
	script.WriteString("*) echo 'unexpected additional attempt' >&2; exit 99;;\nesac\n")
	path := filepath.Join(h.dir, "download.sh")
	require.NoError(t, os.WriteFile(path, []byte(script.String()), 0o600))
	return path
}

// attempts reads the number of times the fixture command was invoked.
func (h *downloadHarness) attempts(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dir, "count"))
	require.NoError(t, err)
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	return count
}

// TestRetryDownloadSuccess verifies single-attempt success, streamed diagnostics
// and argument preservation without evaluating shell metacharacters.
func TestRetryDownloadSuccess(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"download completed", 0})
	output, status := h.run(t, "sh", fixture, "space value", "$(echo unsafe)", "*")
	assert.Zero(t, status)
	assert.Equal(t, 1, h.attempts(t))
	assert.Empty(t, h.sleeps(t))
	for _, text := range []string{"stdout attempt 1", "download completed", "argument=<space value>", "argument=<$(echo unsafe)>", "argument=<*>"} {
		assert.Contains(t, output, text)
	}
}

// TestRetryDownloadTransientErrors covers the observed Yarn 502/503 errors and
// equivalent Go, npm, HTTP and temporary network diagnostics.
func TestRetryDownloadTransientErrors(t *testing.T) {
	for _, message := range []string{
		`error Error: https://registry.yarnpkg.com/select-hose/-/select-hose-2.0.0.tgz: Request failed "502 Bad Gateway"`,
		`error Error: https://registry.yarnpkg.com/resolve-pathname/-/resolve-pathname-3.0.0.tgz: Request failed "503 Service Unavailable"`,
		"go: example.org/mod: reading https://proxy.golang.org/example.org/mod/@v/v1.0.0.zip: 502 Bad Gateway",
		"HTTP/2 429", "HTTP/1.1 504 Gateway Timeout", "500 Internal Server Error", "408 Request Timeout",
		"curl: (22) The requested URL returned error: 503", "npm error code E503",
		"read ECONNRESET", "connect ETIMEDOUT", "getaddrinfo EAI_AGAIN", "connect ECONNREFUSED",
		"ENETUNREACH", "EHOSTUNREACH", "socket hang up", "TLS handshake timeout", "i/o timeout",
		"connection reset by peer", "Temporary failure in name resolution",
	} {
		t.Run(message, func(t *testing.T) {
			h := newDownloadHarness(t)
			fixture := h.fixture(t, outcome{message, 1}, outcome{"download completed", 0})
			output, status := h.run(t, "sh", fixture)
			assert.Zero(t, status, output)
			assert.Equal(t, 2, h.attempts(t))
			assert.Equal(t, []string{"5"}, h.sleeps(t))
			assert.Contains(t, output, message)
			assert.Contains(t, output, "download completed")
		})
	}
}

// TestRetryDownloadPermanentErrors verifies that authentication, missing packages,
// integrity failures and ordinary command errors are not retried.
func TestRetryDownloadPermanentErrors(t *testing.T) {
	for _, message := range []string{
		"401 Unauthorized", "403 Forbidden", "404 Not Found", "501 Not Implemented",
		"unable to verify the first certificate", "Integrity check failed", "checksum mismatch",
		"error Couldn't find any versions for package", "syntax error", "command not found",
		"package-502 version 503.0.0 is invalid", "HTTP/1.1 5030", "1502 Bad Gateway", "go: inconsistent vendoring",
	} {
		t.Run(message, func(t *testing.T) {
			h := newDownloadHarness(t)
			fixture := h.fixture(t, outcome{message, 23})
			output, status := h.run(t, "sh", fixture)
			assert.Equal(t, 23, status, output)
			assert.Equal(t, 1, h.attempts(t))
			assert.Empty(t, h.sleeps(t))
		})
	}
}

// TestRetryDownloadLimit verifies bounded backoff and preservation of
// the final command status when a download keeps failing.
func TestRetryDownloadLimit(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"502 Bad Gateway", 1}, outcome{"503 Service Unavailable", 2}, outcome{"504 Gateway Timeout", 7})
	output, status := h.run(t, "sh", fixture)
	assert.Equal(t, 7, status, output)
	assert.Equal(t, 3, h.attempts(t))
	assert.Equal(t, []string{"5", "10"}, h.sleeps(t))
}

// TestRetryDownloadRecoveryOnLastAttempt verifies that a third-attempt success is
// returned as success rather than masked by either prior failure.
func TestRetryDownloadRecoveryOnLastAttempt(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"502 Bad Gateway", 1}, outcome{"503 Service Unavailable", 2}, outcome{"download completed", 0})
	output, status := h.run(t, "sh", fixture)
	assert.Zero(t, status, output)
	assert.Equal(t, 3, h.attempts(t))
	assert.Equal(t, []string{"5", "10"}, h.sleeps(t))
}

// TestRetryDownloadClearsPreviousDiagnostics verifies that an old transient error
// cannot cause a subsequent permanent error to be retried.
func TestRetryDownloadClearsPreviousDiagnostics(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"502 Bad Gateway", 1}, outcome{"401 Unauthorized", 23})
	output, status := h.run(t, "sh", fixture)
	assert.Equal(t, 23, status, output)
	assert.Equal(t, 2, h.attempts(t))
	assert.Equal(t, []string{"5"}, h.sleeps(t))
}

// TestRetryDownloadSignalExit verifies that cancellation-like exit codes are not
// retried even when the command already printed a transient network diagnostic.
func TestRetryDownloadSignalExit(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"502 Bad Gateway", 130})
	output, status := h.run(t, "sh", fixture)
	assert.Equal(t, 130, status, output)
	assert.Equal(t, 1, h.attempts(t))
	assert.Empty(t, h.sleeps(t))
}

// TestRetryDownloadUsage verifies that omitting the command fails before execution.
func TestRetryDownloadUsage(t *testing.T) {
	h := newDownloadHarness(t)
	output, status := h.run(t)
	assert.Equal(t, 2, status)
	assert.Contains(t, output, "Usage:")
}

// TestRetryDownloadMissingCommand preserves command-not-found failures without retrying.
func TestRetryDownloadMissingCommand(t *testing.T) {
	h := newDownloadHarness(t)
	output, status := h.run(t, "argo-nonexistent-download-command")
	assert.Equal(t, 127, status, output)
	assert.Empty(t, h.sleeps(t))
}

// TestRetryDownloadStreamFailure verifies that a broken output pipeline cannot
// turn a failed invocation into an apparently successful download.
func TestRetryDownloadStreamFailure(t *testing.T) {
	h := newDownloadHarness(t)
	fixture := h.fixture(t, outcome{"download completed", 0})
	require.NoError(t, os.WriteFile(filepath.Join(h.dir, "bin", "tee"), []byte("#!/bin/sh\ncat >/dev/null\nexit 9\n"), 0o755))
	output, status := h.run(t, "sh", fixture)
	assert.Equal(t, 9, status, output)
	assert.Equal(t, 1, h.attempts(t))
	assert.Empty(t, h.sleeps(t))
}

// TestRetryDownloadRealYarn verifies recovery through an actual Yarn install when
// a local package server first returns the same HTTP 502 seen in CI.
func TestRetryDownloadRealYarn(t *testing.T) {
	h := newDownloadHarness(t)
	yarn, err := exec.LookPath("yarn")
	if err != nil {
		t.Skip("Yarn is not installed; the shell-helper tests still run")
	}
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	manifest := []byte(`{"name":"download-fixture","version":"1.0.0"}`)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: "package/package.json", Mode: 0o644, Size: int64(len(manifest))}))
	_, err = tarWriter.Write(manifest)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()
	packageJSON := fmt.Sprintf(`{"private":true,"dependencies":{"download-fixture":%q}}`, server.URL+"/fixture.tgz")
	require.NoError(t, os.WriteFile(filepath.Join(h.dir, "package.json"), []byte(packageJSON), 0o600))
	h.env = append(h.env, "HOME="+h.dir, "YARN_CACHE_FOLDER="+filepath.Join(h.dir, "cache"), "YARN_DISABLE_SELF_UPDATE_CHECK=true", "NO_PROXY=127.0.0.1,localhost")
	output, status := h.run(t, yarn, "--cwd", h.dir, "install", "--non-interactive", "--no-default-rc", "--ignore-scripts", "--network-timeout", "1000")
	require.Zero(t, status, output)
	assert.Contains(t, output, "502 Bad Gateway")
	assert.Equal(t, []string{"5"}, h.sleeps(t), "the wrapper, not a remote registry, must supply the retry")
	assert.Equal(t, int32(2), requests.Load())
	installed, err := os.ReadFile(filepath.Join(h.dir, "node_modules", "download-fixture", "package.json"))
	require.NoError(t, err)
	assert.Equal(t, string(manifest), string(installed))
}
