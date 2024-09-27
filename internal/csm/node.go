package csm

type csmNode interface {
	// Value returns the value held by the node.
	Value() int

	// Reset resets the node value to the minimum valid value.
	Reset()

	// Next changes the node value to the next valid value.
	// It returns true if the value overflowed and false otherwise.
	Next() bool

	// findForward checks if the current node value is valid.
	// If it is not valid, find the next valid value.
	findForward() result
}

type result int

const (
	unchanged result = iota
	advanced
	overflowed
)
