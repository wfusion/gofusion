package consts

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

const (
	ctxReqStartAtKey = "http:req_start_at"
	ctxReqCostKey    = "http:req_cost"
)

func SetReqStartTime(c *gin.Context) {
	c.Set(ctxReqStartAtKey, time.Now())
}

func GetReqCost(c *gin.Context) time.Duration {
	if cost, ok := c.Get(ctxReqCostKey); ok {
		return cast.ToDuration(cost)
	}

	start := time.Now()
	if _, ok := c.Get(ctxReqStartAtKey); ok {
		start = c.GetTime(ctxReqStartAtKey)
	}
	cost := time.Since(start)
	c.Set(ctxReqCostKey, cost)
	return cost
}
