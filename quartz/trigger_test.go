package quartz_test

import (
	"testing"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

var from_epoch int64 = 1577836800000000000

func TestSimpleTrigger(t *testing.T) {
	trigger := quartz.NewSimpleTrigger(time.Second * 5)
	next, err := trigger.NextFireTime(from_epoch)
	assertEqualInt64(t, next, 1577836805000000000)
	assertEqual(t, err, nil)
	next, err = trigger.NextFireTime(next)
	assertEqualInt64(t, next, 1577836810000000000)
	assertEqual(t, err, nil)
	next, err = trigger.NextFireTime(next)
	assertEqualInt64(t, next, 1577836815000000000)
	assertEqual(t, err, nil)
}

func TestRunOnceTrigger(t *testing.T) {
	trigger := quartz.NewRunOnceTrigger(time.Second * 5)
	next, err := trigger.NextFireTime(from_epoch)
	assertEqualInt64(t, next, 1577836805000000000)
	assertEqual(t, err, nil)
	next, err = trigger.NextFireTime(next)
	assertEqualInt64(t, next, 0)
	assertNotEqual(t, err, nil)
}
