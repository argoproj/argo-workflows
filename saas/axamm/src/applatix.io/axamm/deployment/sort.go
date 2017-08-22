package deployment

type DeploymentSorter []*Deployment

// Len is part of sort.Interface.
func (s DeploymentSorter) Len() int {
	return len(s)
}

// Swap is part of sort.Interface.
func (s DeploymentSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Swap is part of sort.Interface.
// DESC ordering
func (s DeploymentSorter) Less(i, j int) bool {
	return s[i].CreateTime > s[j].CreateTime
}
