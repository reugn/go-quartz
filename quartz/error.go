package quartz

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrIllegalArgument = errors.New("illegal argument")
	ErrCronParse       = errors.New("parse cron expression")
	ErrTriggerExpired  = errors.New("trigger has expired")

	ErrIllegalState     = errors.New("illegal state")
	ErrQueueEmpty       = errors.New("queue is empty")
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyExists = errors.New("job already exists")
	ErrJobIsSuspended   = errors.New("job is suspended")
	ErrJobIsActive      = errors.New("job is active")
)

// newIllegalArgumentError returns an illegal argument error with a custom
// error message, which unwraps to ErrIllegalArgument.
func newIllegalArgumentError(message string) error {
	return fmt.Errorf("%w: %s", ErrIllegalArgument, message)
}

// newCronParseError returns a cron parse error with a custom error message,
// which unwraps to ErrCronParse.
func newCronParseError(message string) error {
	return fmt.Errorf("%w: %s", ErrCronParse, message)
}

// newIllegalStateError returns an illegal state error specifying it with err.
func newIllegalStateError(err error) error {
	return fmt.Errorf("%w: %w", ErrIllegalState, err)
}
