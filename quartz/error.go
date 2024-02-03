package quartz

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrIllegalArgument = errors.New("illegal argument")
	ErrIllegalState    = errors.New("illegal state")
	ErrCronParse       = errors.New("parse cron expression")
	ErrJobNotFound     = errors.New("job not found")
	ErrQueueEmpty      = errors.New("queue is empty")
	ErrTriggerExpired  = errors.New("trigger has expired")
)

// illegalArgumentError returns an illegal argument error with a custom
// error message, which unwraps to ErrIllegalArgument.
func illegalArgumentError(message string) error {
	return fmt.Errorf("%w: %s", ErrIllegalArgument, message)
}

// illegalStateError returns an illegal state error with a custom
// error message, which unwraps to ErrIllegalState.
func illegalStateError(message string) error {
	return fmt.Errorf("%w: %s", ErrIllegalState, message)
}

// cronParseError returns a cron parse error with a custom error message,
// which unwraps to ErrCronParse.
func cronParseError(message string) error {
	return fmt.Errorf("%w: %s", ErrCronParse, message)
}

// jobNotFoundError returns a job not found error with a custom error message,
// which unwraps to ErrJobNotFound.
func jobNotFoundError(message string) error {
	return fmt.Errorf("%w: %s", ErrJobNotFound, message)
}
