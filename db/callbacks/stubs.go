package callbacks

import (
	"database/sql"

	"gorm.io/gorm"

	_ "unsafe"
)

//go:linkname createClauses gorm.io/gorm/callbacks.createClauses
var createClauses []string

//go:linkname updateClauses gorm.io/gorm/callbacks.updateClauses
var updateClauses []string

//go:linkname deleteClauses gorm.io/gorm/callbacks.deleteClauses
var deleteClauses []string

//go:linkname checkVersion gorm.io/driver/mysql.checkVersion
func checkVersion(newVersion, oldVersion string) bool

//go:linkname hasReturning gorm.io/gorm/callbacks.hasReturning
func hasReturning(tx *gorm.DB, supportReturning bool) (bool, gorm.ScanMode)

//go:linkname convertAssignRows database/sql.convertAssignRows
func convertAssignRows(dest, src any, rows *sql.Rows) error
