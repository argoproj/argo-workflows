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
