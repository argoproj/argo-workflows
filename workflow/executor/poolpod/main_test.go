package poolpod

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestMain runs as a helper subprocess when GO_POOLPOD_HELPER is set; see
// runHelper.
func TestMain(m *testing.M) {
	if helper := os.Getenv("GO_POOLPOD_HELPER"); helper != "" { //nolint:forbidigo // env is the only channel to a re-exec'd test helper process
		os.Exit(runHelper(helper))
	}
	os.Exit(m.Run())
}

// runHelper runs one helper and returns its exit code:
//
//   - "take" takes the task in GO_POOLPOD_TASK, as main does, prints
//     "TAKEN\n" and sleeps until killed.
//   - "ignore-term" ignores SIGTERM, prints "READY\n" and sleeps until
//     killed.
func runHelper(helper string) int {
	switch helper {
	case "take":
		task := NewTask(os.Getenv("GO_POOLPOD_TASK")) //nolint:forbidigo // env is the only channel to a re-exec'd test helper process
		run, err := task.Take(context.Background())
		if err != nil {
			_, _ = os.Stderr.WriteString("take failed: " + err.Error() + "\n")
			return 2
		}
		_, _ = os.Stdout.WriteString("TAKEN\n")
		time.Sleep(time.Minute)
		// The lock's file would be closed if it were collected.
		runtime.KeepAlive(run)
		return 0
	case "ignore-term":
		signal.Ignore(syscall.SIGTERM)
		_, _ = os.Stdout.WriteString("READY\n")
		time.Sleep(time.Minute)
		return 0
	}
	_, _ = os.Stderr.WriteString("unknown helper " + helper + "\n")
	return 2
}
