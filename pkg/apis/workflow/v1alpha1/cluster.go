package v1alpha1

// ClusterName holds the name of a given cluster where operations are being done
type ClusterName string

// String converts a ClusterName to its underlying string
func (name ClusterName) String() string {
	return string(name)
}
