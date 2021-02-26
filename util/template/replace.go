package template

func Replace(s string, replaceMap map[string]string, allowUnresolved bool) (string, error) {
	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}
	return t.Replace(replaceMap, allowUnresolved)
}
