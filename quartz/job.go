package quartz

import (
	"context"
)

// Job represents an interface to be implemented by structs which
// represent a task to be performed.
// Some Job implementations can be found in the job package.
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated
	// with this job fires.
	Execute(context.Context) error

	// Description returns the description of the Job.
	Description() string
}
