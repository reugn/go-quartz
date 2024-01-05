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
	return csm.ValueWithLocation(loc)
}
