package quartz

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrIllegalArgument = errors.New("illegal argument")
	ErrCronParse       = errors.New("parse cron expression")
	ErrJobNotFound     = errors.New("job not found")
)

// illegalArgumentError returns an illegal argument error with a custom
// error message, which unwraps to ErrIllegalArgument.
func illegalArgumentError(message string) error {
	return fmt.Errorf("%w: %s", ErrIllegalArgument, message)
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
