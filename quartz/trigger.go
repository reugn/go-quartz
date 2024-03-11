package quartz

import (
	"fmt"
	"time"
)

// Trigger represents the mechanism by which Jobs are scheduled.
type Trigger interface {
	// NextFireTime returns the next time at which the Trigger is scheduled to fire.
	NextFireTime(prev int64) (int64, error)

	// Description returns the description of the Trigger.
	Description() string
}

// SimpleTrigger implements the quartz.Trigger interface; uses a fixed interval.
type SimpleTrigger struct {
	Interval time.Duration
}

// Verify SimpleTrigger satisfies the Trigger interface.
var _ Trigger = (*SimpleTrigger)(nil)

// NewSimpleTrigger returns a new SimpleTrigger using the given interval.
func NewSimpleTrigger(interval time.Duration) *SimpleTrigger {
	return &SimpleTrigger{
		Interval: interval,
	}
}

// NextFireTime returns the next time at which the SimpleTrigger is scheduled to fire.
func (st *SimpleTrigger) NextFireTime(prev int64) (int64, error) {
	next := prev + st.Interval.Nanoseconds()
	return next, nil
}

// Description returns the description of the trigger.
func (st *SimpleTrigger) Description() string {
	return fmt.Sprintf("SimpleTrigger%s%s", Sep, st.Interval)
}

// RunOnceTrigger implements the quartz.Trigger interface.
// This type of Trigger can only be fired once and will expire immediately.
type RunOnceTrigger struct {
	Delay   time.Duration
	Expired bool
}

// Verify RunOnceTrigger satisfies the Trigger interface.
var _ Trigger = (*RunOnceTrigger)(nil)

// NewRunOnceTrigger returns a new RunOnceTrigger with the given delay time.
func NewRunOnceTrigger(delay time.Duration) *RunOnceTrigger {
	return &RunOnceTrigger{
		Delay: delay,
	}
}

// NextFireTime returns the next time at which the RunOnceTrigger is scheduled to fire.
// Sets expired to true afterwards.
func (ot *RunOnceTrigger) NextFireTime(prev int64) (int64, error) {
	if !ot.Expired {
		next := prev + ot.Delay.Nanoseconds()
		ot.Expired = true
		return next, nil
	}

	return 0, ErrTriggerExpired
}

// Description returns the description of the trigger.
func (ot *RunOnceTrigger) Description() string {
	status := "valid"
	if ot.Expired {
		status = "expired"
	}

	return fmt.Sprintf("RunOnceTrigger%s%s%s%s", Sep, ot.Delay, Sep, status)
}
