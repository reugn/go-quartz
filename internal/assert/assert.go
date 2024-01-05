package assert

import (
	"reflect"
	"testing"
)

func Equal[T any](t *testing.T, a T, b T) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("%v != %v", a, b)
	}
}

func NotEqual[T any](t *testing.T, a T, b T) {
	if reflect.DeepEqual(a, b) {
		t.Fatalf("%v == %v", a, b)
	}
}
