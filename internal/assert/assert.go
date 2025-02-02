package assert

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Equal verifies equality of two objects.
func Equal[T any](t *testing.T, a T, b T) {
	if !reflect.DeepEqual(a, b) {
		t.Helper()
		t.Fatalf("%v != %v", a, b)
	}
}

// NotEqual verifies objects are not equal.
func NotEqual[T any](t *testing.T, a T, b T) {
	if reflect.DeepEqual(a, b) {
		t.Helper()
		t.Fatalf("%v == %v", a, b)
	}
}

// IsNil verifies that the object is nil.
func IsNil(t *testing.T, obj any) {
	if obj != nil {
		value := reflect.ValueOf(obj)
		switch value.Kind() {
		case reflect.Ptr, reflect.Map, reflect.Slice,
			reflect.Interface, reflect.Func, reflect.Chan:
			if value.IsNil() {
				return
			}
		}
		t.Helper()
		t.Fatalf("%v is not nil", obj)
	}
}

// ErrorContains checks whether the given error contains the specified string.
func ErrorContains(t *testing.T, err error, str string) {
	if err == nil {
		t.Helper()
		t.Fatalf("Error is nil")
	} else if !strings.Contains(err.Error(), str) {
		t.Helper()
		t.Fatalf("Error does not contain string: %s", str)
	}
}

// ErrorIs checks whether any error in err's tree matches target.
func ErrorIs(t *testing.T, err error, target error) {
	if !errors.Is(err, target) {
		t.Helper()
		t.Fatalf("Error type mismatch: %v != %v", err, target)
	}
}

// Panics checks whether the given function panics.
func Panics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Helper()
			t.Fatalf("Function did not panic")
		}
	}()
	f()
}
