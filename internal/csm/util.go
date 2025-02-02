package csm

import (
	"time"
)

// contains returns true if the element is included in the slice.
func contains[T comparable](slice []T, element T) bool {
	for _, e := range slice {
		if element == e {
			return true
		}
	}
	return false
}

// closestWeekday returns the day of the closest weekday within the month of
// the given time t.
func closestWeekday(t time.Time) int {
	if isWeekday(t) {
		return t.Day()
	}

	for i := 1; i <= 7; i++ {
		prevDay := t.AddDate(0, 0, -i)
		if prevDay.Month() == t.Month() && isWeekday(prevDay) {
			return prevDay.Day()
		}

		nextDay := t.AddDate(0, 0, i)
		if nextDay.Month() == t.Month() && isWeekday(nextDay) {
			return nextDay.Day()
		}
	}

	return t.Day()
}

func isWeekday(t time.Time) bool {
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

func lastDayOfMonth(year, month int) int {
	firstDayOfMonth := makeDateTime(year, month, 1)
	return firstDayOfMonth.AddDate(0, 1, -1).Day()
}

func makeDateTime(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
