package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/wfusion/gofusion/http/consts"
)

func Gateway(c *gin.Context) {
	consts.SetReqStartTime(c)
}
