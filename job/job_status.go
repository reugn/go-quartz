package job

// Status represents a Job status.
type Status int8

const (
	// StatusNA is the initial Job status.
	StatusNA Status = iota

	// StatusOK indicates the Job completed successfully.
	StatusOK

	// StatusFailure indicates the Job failed.
	StatusFailure
)
