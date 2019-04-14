package quartz

import (
	"errors"
	"fmt"
	"time"
)

type Trigger interface {
	NextFireTime(prev int64) (int64, error)
	Description() string
}

type SimpleTrigger struct {
	Interval time.Duration
}

//Simple trigger to reschedule Job by Duration interval
func NewSimpleTrigger(interval time.Duration) *SimpleTrigger {
	return &SimpleTrigger{interval}
}

func (st *SimpleTrigger) NextFireTime(prev int64) (int64, error) {
	next := prev + st.Interval.Nanoseconds()
	return next, nil
}

func (st *SimpleTrigger) Description() string {
	return fmt.Sprintf("SimpleTrigger with interval %d", st.Interval)
}

type RunOnceTrigger struct {
	Delay   time.Duration
	expired bool
}

//New run once trigger with delay
func NewRunOnceTrigger(delay time.Duration) *RunOnceTrigger {
	return &RunOnceTrigger{delay, false}
}

func (st *RunOnceTrigger) NextFireTime(prev int64) (int64, error) {
	if !st.expired {
		next := prev + st.Delay.Nanoseconds()
		st.expired = true
		return next, nil
	}
	return 0, errors.New("RunOnce trigger is expired")
}

func (st *RunOnceTrigger) Description() string {
	status := "valid"
	if st.expired {
		status = "expired"
	}
	return fmt.Sprintf("RunOnceTrigger (%s)", status)
}
