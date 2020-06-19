package slice

func RemoveString(slice []string, element string) []string {
	n := len(slice)
	if n == 1 {
		return []string{}
	}
	for i, v := range slice {
		if element == v {
			if n-2 > i {
				slice = append(slice[:i], slice[i+1:]...)
			} else {
				slice = slice[:i]
			}
			return slice
		}
	}
	return slice
}

func ContainsString(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
