package quartz_test

import (
	"fmt"
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
	"github.com/reugn/go-quartz/quartz"
)

func TestJobKey(t *testing.T) {
	name := "jobName"
	jobKey1 := quartz.NewJobKey(name)
	assert.Equal(t, jobKey1.Name(), name)
	assert.Equal(t, jobKey1.Group(), quartz.DefaultGroup)
	assert.Equal(t, jobKey1.String(), fmt.Sprintf("%s::%s", quartz.DefaultGroup, name))

	jobKey2 := quartz.NewJobKeyWithGroup(name, "")
	assert.Equal(t, jobKey2.Name(), name)
	assert.Equal(t, jobKey2.Group(), quartz.DefaultGroup)
	assert.Equal(t, jobKey2.String(), fmt.Sprintf("%s::%s", quartz.DefaultGroup, name))

	if !jobKey1.Equals(jobKey2) {
		t.Fatal("job keys must be equal")
	}
}
