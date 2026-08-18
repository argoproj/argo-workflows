package utils

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// ValidateUploadedArtifactKey checks that key is exactly the format the upload
// endpoint generates for namespace: uploads/{namespace}/{uuid}/{filename}. It
// rejects path traversal, absolute paths, empty segments, and any key outside
// the upload prefix, since a client-supplied key is otherwise applied to the
// artifact location without further checks.
//
// This is defense-in-depth, not a proof of ownership: a valid-looking key
// naming another user's upload under the same namespace still passes.
func ValidateUploadedArtifactKey(namespace, key string) error {
	prefix := "uploads/" + namespace + "/"
	if !strings.HasPrefix(key, prefix) {
		return fmt.Errorf("artifact key %q must start with %q", key, prefix)
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("artifact key %q must not contain '..'", key)
	}
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("artifact key %q must not be an absolute path", key)
	}
	if path.Clean(key) != key {
		return fmt.Errorf("artifact key %q is not in canonical form", key)
	}

	parts := strings.Split(key, "/")
	if len(parts) != 4 {
		return fmt.Errorf("artifact key %q must have exactly 4 segments: uploads/{namespace}/{uuid}/{filename}", key)
	}
	if slices.Contains(parts, "") {
		return fmt.Errorf("artifact key %q must not contain empty segments", key)
	}

	uuidSegment := parts[2]
	if _, err := uuid.Parse(uuidSegment); err != nil {
		return fmt.Errorf("artifact key %q must have a valid UUID segment: %w", key, err)
	}

	filename := parts[3]
	if path.Base(filename) != filename {
		return fmt.Errorf("artifact key %q must have a bare filename segment", key)
	}

	return nil
}
