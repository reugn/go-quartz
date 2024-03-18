package quartz

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	listRune  = ','
	stepRune  = '/'
	rangeRune = '-'
)

// Sep is the serialization delimiter; the default is a double colon.
var Sep = "::"

func translateLiterals(glossary, literals []string) ([]int, error) {
	intValues := make([]int, 0, len(literals))
	for _, literal := range literals {
		index, err := normalize(literal, glossary)
		if err != nil {
			return nil, err
		}
		intValues = append(intValues, index)
	}
	return intValues, nil
}

func extractRangeValues(parsed []string) ([]string, []string) {
	values := make([]string, 0, len(parsed))
	rangeValues := make([]string, 0)
	for _, v := range parsed {
		if strings.ContainsRune(v, rangeRune) { // range value
			rangeValues = append(rangeValues, v)
		} else {
			values = append(values, v)
		}
	}
	return values, rangeValues
}

func extractStepValues(parsed []string) ([]string, []string) {
	values := make([]string, 0, len(parsed))
	stepValues := make([]string, 0)
	for _, v := range parsed {
		if strings.ContainsRune(v, stepRune) { // step value
			stepValues = append(stepValues, v)
		} else {
			values = append(values, v)
		}
	}
	return values, stepValues
}

func fillRangeValues(from, to int) ([]int, error) {
	if to < from {
		return nil, cronParseError("fill range values")
	}
	length := (to - from) + 1
	rangeValues := make([]int, length)
	for i, j := from, 0; i <= to; i, j = i+1, j+1 {
		rangeValues[j] = i
	}
	return rangeValues, nil
}

func fillStepValues(from, step, max int) ([]int, error) {
	if max < from || step == 0 {
		return nil, cronParseError("fill step values")
	}
	length := ((max - from) / step) + 1
	stepValues := make([]int, length)
	for i, j := from, 0; i <= max; i, j = i+step, j+1 {
		stepValues[j] = i
	}
	return stepValues, nil
}

func normalize(field string, glossary []string) (int, error) {
	intVal, err := strconv.Atoi(field)
	if err != nil {
		return translateLiteral(glossary, field)
	}
	return intVal, nil
}

func inScope(value, min, max int) bool {
	if value >= min && value <= max {
		return true
	}
	return false
}

func translateLiteral(glossary []string, literal string) (int, error) {
	upperCaseLiteral := strings.ToUpper(literal)
	for i, value := range glossary {
		if value == upperCaseLiteral {
			return i, nil
		}
	}
	return 0, cronParseError(fmt.Sprintf("unknown literal %s", literal))
}

func invalidCronFieldError(t, field string) error {
	return cronParseError(fmt.Sprintf("invalid %s field %s", t, field))
}

// NowNano returns the current Unix time in nanoseconds.
func NowNano() int64 {
	return time.Now().UnixNano()
}
