package quartz_test

import (
	"testing"
	"time"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

var fromEpoch int64 = 1577836800000000000

func TestSimpleTrigger(t *testing.T) {
	trigger := quartz.NewSimpleTrigger(time.Second * 5)
	trigger.Description()

	next, err := trigger.NextFireTime(fromEpoch)
	assert.Equal(t, next, 1577836805000000000)
	assert.Equal(t, err, nil)

	next, err = trigger.NextFireTime(next)
	assert.Equal(t, next, 1577836810000000000)
	assert.Equal(t, err, nil)

	next, err = trigger.NextFireTime(next)
	assert.Equal(t, next, 1577836815000000000)
	assert.Equal(t, err, nil)
}

func TestRunOnceTrigger(t *testing.T) {
	trigger := quartz.NewRunOnceTrigger(time.Second * 5)
	trigger.Description()

	next, err := trigger.NextFireTime(fromEpoch)
	assert.Equal(t, next, 1577836805000000000)
	assert.Equal(t, err, nil)

	next, err = trigger.NextFireTime(next)
	trigger.Description()
	assert.Equal(t, next, 0)
	assert.NotEqual(t, err, nil)
}
