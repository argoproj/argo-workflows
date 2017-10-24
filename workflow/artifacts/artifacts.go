package executor

// ArtifactInterface is the interface for loading and saving of artifacts
type ArtifactInterface interface {
	// Load accepts an artifact source URL and places it at path
	Load(sourceURL string, path string) error
	// Save uploads the path to a destination URL
	Save(path string, destURL string) (string, error)
}
