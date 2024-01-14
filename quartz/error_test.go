package quartz

import (
	"errors"
	"fmt"
	"testing"

	"github.com/reugn/go-quartz/internal/assert"
)

func TestIllegalArgumentError(t *testing.T) {
	message := "argument is nil"
	err := illegalArgumentError(message)
	if !errors.Is(err, ErrIllegalArgument) {
		t.Fatal("error must match ErrIllegalArgument")
	}
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrIllegalArgument, message))
}

func TestCronParseError(t *testing.T) {
	message := "invalid field"
	err := cronParseError(message)
	if !errors.Is(err, ErrCronParse) {
		t.Fatal("error must match ErrCronParse")
	}
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrCronParse, message))
}

func TestJobNotFoundError(t *testing.T) {
	message := "for key"
	err := jobNotFoundError(message)
	if !errors.Is(err, ErrJobNotFound) {
		t.Fatal("error must match ErrJobNotFound")
	}
	assert.Equal(t, err.Error(), fmt.Sprintf("%s: %s", ErrJobNotFound, message))
}
