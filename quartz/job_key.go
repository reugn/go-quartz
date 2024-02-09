package quartz

import "fmt"

const (
	DefaultGroup = "default"
)

// JobKey represents the identifier of a scheduled job.
// Keys are composed of both a name and group, and the name must be unique
// within the group.
// If only a name is specified then the default group name will be used.
type JobKey struct {
	name  string
	group string
}

// NewJobKey returns a new NewJobKey using the given name.
func NewJobKey(name string) *JobKey {
	return &JobKey{
		name:  name,
		group: DefaultGroup,
	}
}

// NewJobKeyWithGroup returns a new NewJobKey using the given name and group.
func NewJobKeyWithGroup(name, group string) *JobKey {
	if group == "" { // use default if empty
		group = DefaultGroup
	}
	return &JobKey{
		name:  name,
		group: group,
	}
}

// String returns string representation of the JobKey.
func (jobKey *JobKey) String() string {
	return fmt.Sprintf("%s%s%s", jobKey.group, Sep, jobKey.name)
}

// Equals indicates whether some other JobKey is "equal to" this one.
func (jobKey *JobKey) Equals(that *JobKey) bool {
	return jobKey.name == that.name &&
		jobKey.group == that.group
}

// Name returns the name of the JobKey.
func (jobKey *JobKey) Name() string {
	return jobKey.name
}

// Group returns the group of the JobKey.
func (jobKey *JobKey) Group() string {
	return jobKey.group
}
