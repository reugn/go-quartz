package quartz

// Matcher represents a predicate (boolean-valued function) of one argument.
// Matchers can be used in various Scheduler API methods to select the entities
// that should be operated.
// Standard Matcher implementations are located in the matcher package.
type Matcher[T any] interface {
	// IsMatch evaluates this matcher on the given argument.
	IsMatch(T) bool
}
