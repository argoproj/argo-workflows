package common

import (
	"fmt"
	"io"
	"os"
)

// SaveStreamViaTempFile buffers reader to a temp file and hands the path to save,
// so drivers whose storage SDK needs a seekable input reuse their existing Save
// logic (bucket creation, key normalization, retries) unchanged instead of
// reimplementing it per driver.
func SaveStreamViaTempFile(reader io.Reader, pattern string, save func(path string) error) error {
	path, cleanup, err := BufferReaderToTempFile(reader, pattern)
	if err != nil {
		return err
	}
	defer cleanup()
	return save(path)
}

// BufferReaderToTempFile buffers reader into a new temp file (named per the os.CreateTemp
// pattern, e.g. "s3-upload-*") so its content can be re-read multiple times, which most
// storage SDKs require for retry/backoff. It returns the temp file's path and a cleanup
// function that removes it; cleanup is safe to call more than once. On error, any partially
// written temp file is removed before returning, so callers only need to defer cleanup
// after a nil error.
func BufferReaderToTempFile(reader io.Reader, pattern string) (path string, cleanup func(), err error) {
	noop := func() {}

	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", noop, fmt.Errorf("failed to create temp file: %w", err)
	}
	name := tmpFile.Name()
	cleanup = func() {
		_ = os.Remove(name)
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", noop, fmt.Errorf("failed to buffer stream to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("failed to close temp file: %w", err)
	}

	return name, cleanup, nil
}
