package softdelete

import (
	"log"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
	"github.com/wfusion/gofusion/config"
)

var (
	PatchGormDeleteAtOnce = new(sync.Once)
)

func PatchGormDeleteAt() (patches *gomonkey.Patches) {
	PatchGormDeleteAtOnce.Do(func() {
		pid := syscall.Getpid()
		_, err := utils.Catch(func() {
			patches = gomonkey.ApplyMethod(gorm.SoftDeleteDeleteClause{},
				"ModifyStatement", gormSoftDeleteDeleteClauseModifyStatement)
			log.Printf("%v [Gofusion] %s patch gorm.SoftDeleteDeleteClause success", pid, config.ComponentDB)
		})
		if err != nil {
			log.Printf("%v [Gofusion] %s patch gorm.SoftDeleteDeleteClause failed: %s",
				pid, config.ComponentDB, errors.Cause(err))
		}
	})

	return
}

func gormSoftDeleteDeleteClauseModifyStatement(sd gorm.SoftDeleteDeleteClause, stmt *gorm.Statement) {
	if stmt.Statement.Unscoped || stmt.SQL.Len() > 0 {
		return
	}

	curTime := stmt.DB.NowFunc()
	setClauses := clause.Set{{Column: clause.Column{Name: sd.Field.DBName}, Value: curTime}}
	if clauses, ok := stmt.Clauses[setClauses.Name()]; ok {
		if exprClauses, ok := clauses.Expression.(clause.Set); ok {
			setClauses = append(setClauses, exprClauses...)
		}
	}
	stmt.AddClause(setClauses)
	stmt.SetColumn(sd.Field.DBName, curTime, true)

	gorm.SoftDeleteQueryClause(sd).ModifyStatement(stmt)
}
