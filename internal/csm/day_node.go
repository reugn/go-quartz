package csm

import "time"

const (
	NLastDayOfMonth = 1
	NWeekday        = 2
)

type DayNode struct {
	c             CommonNode
	weekdayValues []int
	n             int
	month         csmNode
	year          csmNode
}

var _ csmNode = (*DayNode)(nil)

func NewMonthDayNode(value, min, max, n int, dayOfMonthValues []int, month, year csmNode) *DayNode {
	return &DayNode{
		c:             CommonNode{value, min, max, dayOfMonthValues},
		weekdayValues: make([]int, 0),
		n:             n,
		month:         month,
		year:          year,
	}
}

func NewWeekDayNode(value, min, max, n int, dayOfWeekValues []int, month, year csmNode) *DayNode {
	return &DayNode{
		c:             CommonNode{value, min, max, make([]int, 0)},
		weekdayValues: dayOfWeekValues,
		n:             n,
		month:         month,
		year:          year,
	}
}

func (n *DayNode) Value() int {
	return n.c.Value()
}

func (n *DayNode) Reset() {
	n.c.value = n.c.min
	n.findForward()
}

func (n *DayNode) Next() (overflowed bool) {
	if n.isWeekday() {
		if n.n == 0 {
			return n.nextWeekday()
		}
		return n.nextWeekdayN()
	}
	if n.n == 0 {
		return n.nextDay()
	}
	return n.nextDayN()
}

func (n *DayNode) nextWeekday() (overflowed bool) {
	// the weekday of the previous scheduled time
	weekday := n.getWeekday()

	// the offset in days from the previous to the next day
	offset := 7 + n.weekdayValues[0] - weekday
	// find the next value in the range (assuming weekdays is sorted)
	for _, value := range n.weekdayValues {
		if value > weekday {
			offset = value - weekday
			break
		}
	}

	// if the end of the values array is reached set to the first valid value
	return n.addDays(offset)
}

func (n *DayNode) nextDay() (overflowed bool) {
	return n.c.Next()
}

func (n *DayNode) findForward() result {
	if !n.isValid() {
		if n.Next() {
			return overflowed
		}
		return advanced
	}
	return unchanged
}

func (n *DayNode) isValid() bool {
	withinLimits := n.isValidDay()
	if n.isWeekday() {
		withinLimits = withinLimits && n.isValidWeekday()
	}
	return withinLimits
}

func (n *DayNode) isValidWeekday() bool {
	return contains(n.weekdayValues, n.getWeekday())
}

func (n *DayNode) isValidDay() bool {
	return n.c.isValid() && n.c.value <= n.max()
}

func (n *DayNode) isWeekday() bool {
	return len(n.weekdayValues) != 0
}

func (n *DayNode) getWeekday() int {
	date := makeDateTime(n.year.Value(), n.month.Value(), n.c.value)
	return int(date.Weekday())
}

func (n *DayNode) addDays(offset int) (overflowed bool) {
	overflowed = n.Value()+offset > n.max()
	today := makeDateTime(n.year.Value(), n.month.Value(), n.c.value)
	newDate := today.AddDate(0, 0, offset)
	n.c.value = newDate.Day()
	return
}

func (n *DayNode) max() int {
	month := time.Month(n.month.Value())
	year := n.year.Value()

	if month == time.December {
		month = 1
		year++
	} else {
		month++
	}

	date := makeDateTime(year, int(month), 0)
	return date.Day()
}

func (n *DayNode) nextDayN() (overflowed bool) {
	switch n.n {
	case NWeekday:
		n.nextWeekdayOfMonth()
	default:
		n.nextLastDayOfMonth()
	}
	return
}

func (n *DayNode) nextWeekdayOfMonth() {
	year := n.year.Value()
	month := n.month.Value()

	monthLastDate := lastDayOfMonth(year, month)
	date := n.c.values[0]
	if date > monthLastDate {
		date = monthLastDate
	}

	monthDate := makeDateTime(year, month, date)
	closest := closestWeekday(monthDate)
	if n.c.value >= closest {
		n.c.value = 0
		n.advanceMonth()
		n.nextWeekdayOfMonth()
		return
	}

	n.c.value = closest
}

func (n *DayNode) nextLastDayOfMonth() {
	year := n.year.Value()
	month := n.month.Value()

	firstDayOfMonth := makeDateTime(year, month, 1)
	offset := n.n
	if offset == NLastDayOfMonth {
		offset = 0
	}
	dayOfMonth := firstDayOfMonth.AddDate(0, 1, offset-1)

	if n.c.value >= dayOfMonth.Day() {
		n.c.value = 0
		n.advanceMonth()
		n.nextLastDayOfMonth()
		return
	}

	n.c.value = dayOfMonth.Day()
}

func (n *DayNode) nextWeekdayN() (overflowed bool) {
	n.c.value = n.getDayInMonth(n.daysOfWeekInMonth())
	return
}

func (n *DayNode) getDayInMonth(dates []int) int {
	if n.n > len(dates) {
		n.advanceMonth()
		return n.getDayInMonth(n.daysOfWeekInMonth())
	}

	var dayInMonth int
	if n.n > 0 {
		dayInMonth = dates[n.n-1]
	} else {
		dayInMonth = dates[len(dates)-1]
	}

	if n.c.value >= dayInMonth {
		n.c.value = 0
		n.advanceMonth()
		return n.getDayInMonth(n.daysOfWeekInMonth())
	}

	return dayInMonth
}

func (n *DayNode) advanceMonth() {
	if n.month.Next() {
		_ = n.year.Next()
	}
}

func (n *DayNode) daysOfWeekInMonth() []int {
	year := n.year.Value()
	month := n.month.Value()

	// the day of week specified for the node
	weekday := n.weekdayValues[0]

	dates := make([]int, 0, 5)
	// iterate through all the days of the month
	for day := 1; ; day++ {
		currentDate := makeDateTime(year, month, day)
		// stop if we have reached the next month
		if currentDate.Month() != time.Month(month) {
			break
		}
		// check if the current day is the required day of the week
		if int(currentDate.Weekday()) == weekday {
			dates = append(dates, day)
		}
	}

	return dates
}
