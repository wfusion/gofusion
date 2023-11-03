package sqlparser_test

import (
	"testing"

	"github.com/wfusion/gofusion/common/utils/sqlparser"
)

func TestPos_String(t *testing.T) {
	if got, want := (sqlparser.Pos{}).String(), `-`; got != want {
		t.Fatalf("String()=%q, want %q", got, want)
	}
}
