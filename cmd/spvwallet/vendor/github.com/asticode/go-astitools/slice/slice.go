package astislice

// InStringSlice checks whether a string is in a string slice
func InStringSlice(i string, s []string) (found bool) {
	for _, v := range s {
		if v == i {
			return true
		}
	}
	return
}
