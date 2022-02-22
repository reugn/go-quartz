package quartz

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)

func indexes(search []string, target []string) ([]int, error) {
	searchIndexes := make([]int, 0, len(search))
	for _, a := range search {
		index := intVal(target, a)
		if index == -1 {
			return nil, fmt.Errorf("invalid cron field: %s", a)
		}
		searchIndexes = append(searchIndexes, index)
	}

	return searchIndexes, nil
}

func sliceAtoi(sa []string) ([]int, error) {
	si := make([]int, 0, len(sa))
	for _, a := range sa {
		i, err := strconv.Atoi(a)
		if err != nil {
			return si, err
		}

		si = append(si, i)
	}

	return si, nil
}

func fillRange(from, to int) ([]int, error) {
	if to < from {
		return nil, cronError("fillRange")
	}

	len := (to - from) + 1
	arr := make([]int, len)

	for i, j := from, 0; i <= to; i, j = i+1, j+1 {
		arr[j] = i
	}

	return arr, nil
}

func fillStep(from, step, max int) ([]int, error) {
	if max < from {
		return nil, cronError("fillStep")
	}

	len := ((max - from) / step) + 1
	arr := make([]int, len)

	for i, j := from, 0; i <= max; i, j = i+step, j+1 {
		arr[j] = i
	}

	return arr, nil
}

func normalize(field string, dict []string) int {
	i, err := strconv.Atoi(field)
	if err == nil {
		return i
	}

	return intVal(dict, field)
}

func inScope(i, min, max int) bool {
	if i >= min && i <= max {
		return true
	}

	return false
}

func cronError(cause string) error {
	return fmt.Errorf("invalid cron expression: %s", cause)
}

// alignDigit justifies single digit values.
func alignDigit(next int) string {
	if next < 10 {
		return "0" + strconv.Itoa(next)
	}

	return strconv.Itoa(next)
}

func step(prev, next, max int) int {
	diff := next - prev
	if diff < 0 {
		return diff + max
	}

	return diff
}

func intVal(target []string, search string) int {
	uSearch := strings.ToUpper(search)
	for i, v := range target {
		if v == uSearch {
			return i
		}
	}

	return -1 // TODO: return error
}

// atoi implements an unsafe strconv.Atoi.
func atoi(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}

func maxDays(month, year int) int {
	if month == 2 && isLeapYear(year) {
		return 29
	}

	return daysInMonth[month]
}

// isLeapYear determines if the given year is a leap year.
// nolint:gocritic
func isLeapYear(year int) bool {
	if year%4 != 0 {
		return false
	} else if year%100 != 0 {
		return true
	} else if year%400 != 0 {
		return false
	}
	return true
}

// NowNano returns the current UTC Unix time in nanoseconds.
func NowNano() int64 {
	return time.Now().UTC().UnixNano()
}

func isOutdated(_time int64) bool {
	return _time < NowNano()-(time.Second*30).Nanoseconds()
}

// HashCode calculates and returns a hash code for the given string.
func HashCode(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}
