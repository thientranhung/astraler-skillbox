package operations

import "fmt"

// Target uniquely identifies the object an operation acts on.
type Target struct {
	Type string
	ID   int64
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%d", t.Type, t.ID)
}
