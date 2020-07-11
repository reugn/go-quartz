package quartz

import (
	"errors"
	"fmt"
	"time"
)

// Trigger is the Triggers interface.
// Triggers are the 'mechanism' by which Jobs are scheduled.
type Trigger interface {

	// NextFireTime returns the next time at which the Trigger is scheduled to fire.
	NextFireTime(prev int64) (int64, error)

	// Description returns a Trigger description.
	Description() string
}

// SimpleTrigger implements the quartz.Trigger interface; uses a time.Duration interval.
type SimpleTrigger struct {
	Interval time.Duration
}

// NewSimpleTrigger returns a new SimpleTrigger.
func NewSimpleTrigger(interval time.Duration) *SimpleTrigger {
	return &SimpleTrigger{interval}
}

// NextFireTime returns the next time at which the SimpleTrigger is scheduled to fire.
func (st *SimpleTrigger) NextFireTime(prev int64) (int64, error) {
	next := prev + st.Interval.Nanoseconds()
	return next, nil
}

// Description returns a SimpleTrigger description.
func (st *SimpleTrigger) Description() string {
	return fmt.Sprintf("SimpleTrigger with the interval %d.", st.Interval)
}

// RunOnceTrigger implements the quartz.Trigger interface. Could be triggered only once.
type RunOnceTrigger struct {
	Delay   time.Duration
	expired bool
}

// NewRunOnceTrigger returns a new RunOnceTrigger.
func NewRunOnceTrigger(delay time.Duration) *RunOnceTrigger {
	return &RunOnceTrigger{delay, false}
}

// NextFireTime returns the next time at which the RunOnceTrigger is scheduled to fire.
// Sets exprired to true afterwards.
func (st *RunOnceTrigger) NextFireTime(prev int64) (int64, error) {
	if !st.expired {
		next := prev + st.Delay.Nanoseconds()
		st.expired = true
		return next, nil
	}

	return 0, errors.New("RunOnce trigger is expired")
}

// Description returns a RunOnceTrigger description.
func (st *RunOnceTrigger) Description() string {
	status := "valid"
	if st.expired {
		status = "expired"
	}

	return fmt.Sprintf("RunOnceTrigger (%s).", status)
}
