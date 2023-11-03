package debug

import "fmt"

func Printf(format string, args ...any) {
	if Debug {
		fmt.Printf(format, args...)
	}
}
