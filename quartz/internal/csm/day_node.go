package csm

import "time"

var _ csmNode = (*DayNode)(nil)

type DayNode struct {
	c             CommonNode
	weekdayValues []int
	month         *csmNode
	year          *csmNode
}

func NewMonthdayNode(value, min, max int, dayOfMonthValues []int, month, year csmNode) *DayNode {
	return &DayNode{CommonNode{value, min, max, dayOfMonthValues}, make([]int, 0), &month, &year}
}

func NewWeekdayNode(value, min, max int, dayOfWeekValues []int, month, year csmNode) *DayNode {
	return &DayNode{CommonNode{value, min, max, make([]int, 0)}, dayOfWeekValues, &month, &year}
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
		return n.nextWeekday()
	}
	return n.nextDay()
}

func (n *DayNode) nextWeekday() (overflowed bool) {
	weekday := n.getWeekday()

	amount := 7 + n.weekdayValues[0] - weekday
	// Find the next value in the range (assuming weekdays is sorted)
	for _, value := range n.weekdayValues {
		if value > weekday {
			amount = value - weekday
			break
		}
	}

	// If the end of the values array is reached set to the first valid value
	return n.addDays(amount)
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
	return contained(n.getWeekday(), n.weekdayValues)
}

func (n *DayNode) isValidDay() bool {
	return n.c.isValid() && n.c.value <= n.max()
}

func (n *DayNode) isWeekday() bool {
	return len(n.weekdayValues) != 0
}

func (n *DayNode) getWeekday() int {
	date := time.Date((*n.year).Value(), time.Month((*n.month).Value()), n.c.value, 0, 0, 0, 0, time.UTC)
	return int(date.Weekday())
}

func (n *DayNode) addDays(amount int) (overflowed bool) {
	overflowed = n.Value()+amount > n.max()
	today := time.Date((*n.year).Value(), time.Month((*n.month).Value()), n.c.value, 0, 0, 0, 0, time.UTC)
	newDate := today.AddDate(0, 0, amount)
	n.c.value = newDate.Day()
	return
}

func (n *DayNode) max() int {
	month := time.Month((*n.month).Value())
	year := (*n.year).Value()

	if month == time.December {
		month = 1
		year++
	} else {
		month++
	}

	date := time.Date(year, month, 0, 0, 0, 0, 0, time.UTC)
	return date.Day()
}
