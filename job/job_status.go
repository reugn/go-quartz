package job

// Status represents a Job status.
type Status int8

const (
	// StatusNA is the initial Job status.
	StatusNA Status = iota

	// StatusOK indicates that the Job completed successfully.
	StatusOK

	// StatusFailure indicates that the Job failed.
	StatusFailure
)
