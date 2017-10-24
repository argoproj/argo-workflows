package executor

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at path
	Load(sourceURL string, path string) error

	// Save uploads the path to a destination URL
	Save(path string, destURL string) (string, error)
}
