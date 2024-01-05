package quartz

import (
	"context"
)

// Job represents an interface to be implemented by structs which
// represent a 'job' to be performed.
type Job interface {
	// Execute is called by a Scheduler when the Trigger associated
	// with this job fires.
	Execute(context.Context) error

	// Description returns the description of the Job.
	Description() string
}
