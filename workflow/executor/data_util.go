package executor

func inPlaceFilter(filter func(file string) bool, files *[]string) {
	keptFiles := 0
	for _, file := range *files {
		if filter(file) {
			(*files)[keptFiles] = file
			keptFiles++
		}
	}
	*files = (*files)[:keptFiles]
}

func groupBy(grouper func(file string) string, files []string) [][]string {
	var groups [][]string
	groupIds := make(map[string]int)
	for _, file := range files {
		group := grouper(file)
		id, ok := groupIds[group]
		if !ok {
			groupIds[group] = len(groups)
			id = len(groups)
			groups = append(groups, []string{})
		}
		groups[id] = append(groups[id], file)
	}
	return groups
}
