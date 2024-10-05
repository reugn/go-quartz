package quartz_test

import (
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

func TestQueueErrors(t *testing.T) {
	t.Parallel()
	queue := quartz.NewJobQueue()
	jobKey := quartz.NewJobKey("job1")

	var err error
	_, err = queue.Pop()
	assert.ErrorIs(t, err, quartz.ErrQueueEmpty)

	_, err = queue.Head()
	assert.ErrorIs(t, err, quartz.ErrQueueEmpty)

	_, err = queue.Get(jobKey)
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)

	_, err = queue.Remove(jobKey)
	assert.ErrorIs(t, err, quartz.ErrJobNotFound)
}
