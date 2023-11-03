package softdelete

import (
	"gorm.io/gorm/clause"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	defaultEnabledFlag   = "soft_delete_enabled"
	statusEnabledFlag    = "soft_delete_enabled_status"
	timestampEnabledFlag = "soft_delete_enabled_timestamp"
	deletedAtEnabledFlag = "soft_delete_enabled_deletedat"
)

func IsClausesWithSoftDelete(clauses map[string]clause.Clause) (withSoftDelete bool) {
	utils.IfAny(
		func() (ok bool) { _, withSoftDelete = clauses[defaultEnabledFlag]; return withSoftDelete },
		func() (ok bool) { _, withSoftDelete = clauses[statusEnabledFlag]; return withSoftDelete },
		func() (ok bool) { _, withSoftDelete = clauses[timestampEnabledFlag]; return withSoftDelete },
		func() (ok bool) { _, withSoftDelete = clauses[deletedAtEnabledFlag]; return withSoftDelete },
	)
	return
}
