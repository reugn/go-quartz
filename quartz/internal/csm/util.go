package csm

// Returns true if the element is included in the slice.
func contained[T comparable](element T, slice []T) bool {
	for _, e := range slice {
		if element == e {
			return true
		}
	}
	return false
}
