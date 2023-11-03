package softdelete

import (
	"database/sql"

	_ "unsafe"
)

//go:linkname convertAssignRows database/sql.convertAssignRows
func convertAssignRows(dest, src any, rows *sql.Rows) error
