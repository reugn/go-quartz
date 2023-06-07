package quartz_test

import (
	"reflect"
	"testing"

	"github.com/reugn/go-quartz/quartz"
)

func assertEqual[T any](t *testing.T, a T, b T) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("%v != %v", a, b)
	}
}

func assertNotEqual[T any](t *testing.T, a T, b T) {
	t.Helper()
	if reflect.DeepEqual(a, b) {
		t.Fatalf("%v == %v", a, b)
	}
}

func TestUtils(t *testing.T) {
	hash := quartz.HashCode("foo")
	assertEqual(t, hash, 2851307223)
}
