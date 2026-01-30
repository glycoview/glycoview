package common

func VisitAll(m map[string]string, fn func(key, value string)) {
	for k, v := range m {
		fn(k, v)
	}
}
