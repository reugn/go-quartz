package csm

import "time"

type NodeID int

const (
	seconds NodeID = iota
	minutes
	hours
	days
	months
	years
)

type CronStateMachine struct {
	second csmNode
	minute csmNode
	hour   csmNode
	day    *DayNode
	month  csmNode
	year   csmNode
}

func NewCronStateMachine(second, minute, hour csmNode, day *DayNode, month, year csmNode) *CronStateMachine {
	return &CronStateMachine{second, minute, hour, day, month, year}
}

func (csm *CronStateMachine) Value() time.Time {
	return csm.ValueWithLocation(time.UTC)
}

func (csm *CronStateMachine) ValueWithLocation(loc *time.Location) time.Time {
	return time.Date(
		csm.year.Value(),
		time.Month(csm.month.Value()),
		csm.day.Value(),
		csm.hour.Value(),
		csm.minute.Value(),
		csm.second.Value(),
		0, loc,
	)
}

func (csm *CronStateMachine) NextTriggerTime(loc *time.Location) time.Time {
	csm.findForward()
	// Resetting the month or year can select a day that does not exist in
	// that month. Validate it before time.Date normalizes it into another month.
	// A full Gregorian cycle bounds expressions such as April 31 that never match.
	for attempts := 0; !csm.isValidDate(); attempts++ {
		if attempts == 400*12 {
			return time.Time{}
		}
		year, month := csm.year.Value(), csm.month.Value()
		csm.overflowFrom(months)
		if csm.year.Value() < year || (csm.year.Value() == year && csm.month.Value() <= month) {
			return time.Time{}
		}
	}
	return csm.ValueWithLocation(loc)
}
