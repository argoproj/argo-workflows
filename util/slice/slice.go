package slice

func RemoveString(slice []string, element string) []string {
	for i, v := range slice {
		if element == v {
			ret := make([]string, 0, len(slice)-1)
			ret = append(ret, slice[:i]...)
			return append(ret, slice[i+1:]...)
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
