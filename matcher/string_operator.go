package matcher

import "strings"

// StringOperator is a function to equate two strings.
type StringOperator func(string, string) bool

// String operators.
var (
	StringEquals     StringOperator = stringsEqual
	StringStartsWith StringOperator = strings.HasPrefix
	StringEndsWith   StringOperator = strings.HasSuffix
	StringContains   StringOperator = strings.Contains
)

func stringsEqual(source, target string) bool {
	return source == target
}
