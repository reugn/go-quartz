package quartz

import (
	"fmt"
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
)

func TestError_IllegalArgument(t *testing.T) {
	message := "argument is nil"
	err := newIllegalArgumentError(message)

	assert.ErrorIs(t, err, ErrIllegalArgument)
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrIllegalArgument, message))
}

func TestError_CronParse(t *testing.T) {
	message := "invalid field"
	err := newCronParseError(message)

	assert.ErrorIs(t, err, ErrCronParse)
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrCronParse, message))
}

func TestError_IllegalState(t *testing.T) {
	err := newIllegalStateError(ErrJobAlreadyExists)

	assert.ErrorIs(t, err, ErrIllegalState)
	assert.ErrorIs(t, err, ErrJobAlreadyExists)
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrIllegalState, ErrJobAlreadyExists))
}
