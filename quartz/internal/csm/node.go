package csm

type csmNode interface {
	// Returns the value held by the node.
	Value() int

	// Resets the node value to the minimum valid value.
	Reset()

	// Changes the node value to the next valid value.
	// Returns true if the value overflowed and false otherwise.
	Next() bool

	// Checks if the current node value is valid.
	// If it is not valid, find the next valid value.
	// Returns true if the value changed and false otherwise.
	findForward() result
}

type result int

const (
	unchanged result = iota
	advanced
	overflowed
)
