package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/wfusion/gofusion/common/utils"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			userID, traceID string
		)
		utils.IfAny(
			func() bool { traceID = c.GetHeader(fmkCtx.KeyTraceID); return traceID != "" },
			func() bool { traceID = c.GetHeader("HTTP_TRACE_ID"); return traceID != "" },
			func() bool {
				traceID = utils.LookupByFuzzyKeyword[string](c.GetHeader, "trace_id")
				return traceID != ""
			},
			func() bool { traceID = utils.NginxID(); return traceID != "" },
		)
		c.Header("Trace-Id", traceID)
		c.Set(fmkCtx.KeyTraceID, traceID)

		utils.IfAny(
			func() bool { userID = c.GetHeader(fmkCtx.KeyUserID); return userID != "" },
			func() bool {
				userID = utils.LookupByFuzzyKeyword[string](c.GetHeader, "user_id")
				return userID != ""
			},
			func() bool {
				userID = utils.LookupByFuzzyKeyword[string](c.GetQuery, "user_id")
				return userID != ""
			},
			func() bool {
				userID = utils.LookupByFuzzyKeyword[string](c.GetPostForm, "user_id")
				return userID != ""
			},
		)
		c.Set(fmkCtx.KeyUserID, userID)
		c.Next()
	}
}
