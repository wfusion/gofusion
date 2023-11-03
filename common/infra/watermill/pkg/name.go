package pkg

import (
	"fmt"
	"strings"
)

// StructName returns a normalized name of the passed structure.
func StructName(v any) string {
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	s := fmt.Sprintf("%T", v)
	// trim the pointer marker, if any
	return strings.TrimLeft(s, "*")
}
